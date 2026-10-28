package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/netguard"
	"github.com/spf13/viper"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	maxRedirects       = 10
)

var (
	ErrRateLimited    = oracle.ErrRateLimited
	ErrInvalidTimeout = errors.New("timeout must be greater than 0")
	ErrHTTPError      = errors.New("http error response")
	ErrInvalidJSON    = errors.New("invalid JSON response")
	ErrEmptyResponse  = errors.New("empty response body")
	ErrPathExtraction = errors.New("path extraction failed")
	// ErrResponseTooLarge is returned when a remote response exceeds
	// maxResponseBytes. Without a cap, a malicious or buggy oracle
	// source could OOM the process by streaming a multi-GB body.
	ErrResponseTooLarge = errors.New("response body exceeds maximum size")
	// ErrBlockedDestination is returned when an initial or redirected request
	// would reach an internal/private address (SSRF) or a non-HTTP(S) scheme.
	// Blocks a hostile source from probing 127.0.0.1, 169.254.169.254,
	// RFC1918 space, etc., even when DNS changes between validation and dial.
	ErrBlockedDestination = errors.New("blocked destination")
	ErrTooManyRedirects   = errors.New("too many redirects")
)

// maxResponseBytes caps the size of any oracle-source response we read.
// 10 MiB is generous for price feeds / on-chain data and small enough
// to keep the process memory-safe even under sustained attack.
const maxResponseBytes = 10 * 1024 * 1024

func isBlockedIP(ip net.IP) bool {
	return netguard.IsBlockedIP(ip)
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func parseIPLiteralHost(host string) net.IP {
	host = hostWithoutPort(host)
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return net.ParseIP(host[1 : len(host)-1])
	}
	return nil
}

// isBlockedHost reports whether host (optionally host:port) is an IP literal
// inside a blocked range.
func isBlockedHost(host string) bool {
	ip := parseIPLiteralHost(host)
	if ip == nil {
		return false
	}
	return isBlockedIP(ip)
}

// validateDialDestination is the last line of SSRF defense. The URL-level
// checks resolve a hostname before the request, but net/http resolves it again
// when dialing, which leaves a DNS-rebinding window between the two lookups.
// Dialer.ControlContext sees the concrete address immediately before connect,
// so a public answer that later resolves to a private address is still refused.
func validateDialDestination(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: invalid dial address %q", ErrBlockedDestination, address)
	}
	if i := strings.LastIndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("%w: unresolved dial address %q", ErrBlockedDestination, address)
	}
	if isBlockedIP(ip) {
		return fmt.Errorf("%w: dial target %q", ErrBlockedDestination, address)
	}
	return nil
}

type proxyDialState struct {
	proxied atomic.Bool
}

type proxyDialContextKey struct{}

func checkDialDestination(ctx context.Context, _, address string, _ syscall.RawConn) error {
	if state, _ := ctx.Value(proxyDialContextKey{}).(*proxyDialState); state != nil && state.proxied.Load() {
		return nil
	}
	return validateDialDestination(address)
}

// redirectHostBlocked decides whether a redirect target host is out of bounds.
// It handles IP literals directly and resolves hostnames so redirects aimed at
// localhost, cloud-metadata names, or other internal DNS entries are refused
// just like literal private IPs. An unresolved/invalid host is treated as
// blocked (we must not follow a redirect whose destination we cannot verify).
// Resolution is bounded by a short timeout so a poisoned/stuck resolver cannot
// hang the fetch indefinitely.
func redirectHostBlocked(host string) (bool, error) {
	host = hostWithoutPort(host)
	if ip := parseIPLiteralHost(host); ip != nil {
		return isBlockedIP(ip), nil
	}
	if strings.ContainsAny(host, "[]") {
		return false, fmt.Errorf("invalid bracketed host %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, fmt.Errorf("resolve %q: %w", host, err)
	}
	for _, a := range addrs {
		if isBlockedIP(a.IP) {
			return true, nil
		}
	}
	return false, nil
}

// checkRedirectAllowed is the http.Client redirect policy: it refuses
// redirects to non-HTTP(S) schemes and to addresses in blockedCIDRs (checked
// by literal IP or by resolving the target hostname), while returning nil
// for everything else. Because a non-nil CheckRedirect replaces net/http's
// default callback, this policy also enforces the standard ten-hop cap itself.
func checkRedirectAllowed(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("%w: stopped after %d redirects", ErrTooManyRedirects, maxRedirects)
	}
	scheme := strings.ToLower(req.URL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: scheme %q", ErrBlockedDestination, req.URL.Scheme)
	}
	blocked, err := redirectHostBlocked(req.URL.Host)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBlockedDestination, err)
	}
	if blocked {
		return fmt.Errorf("%w: %q", ErrBlockedDestination, req.URL.Host)
	}
	// net/http strips Authorization and Cookie on a cross-origin redirect, but
	// not arbitrary credentials such as the X-API-Key header documented for
	// oracle sources. The transport re-adds its safe defaults (User-Agent,
	// Accept and request ID), so start each cross-origin hop with no inherited
	// headers rather than forwarding an upstream secret to another origin.
	if len(via) > 0 && via[0] != nil && !sameURLOrigin(via[0].URL, req.URL) {
		req.Header = make(http.Header)
	}
	return nil
}

func sameURLOrigin(a, b *url.URL) bool {
	if a == nil || b == nil || !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return effectiveURLPort(a) == effectiveURLPort(b)
}

func effectiveURLPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// limiterSweepThreshold bounds the RateLimiter's requests map: when it grows
// past this many tracked sources, Allow sweeps keys idle beyond the window.
// Without it every sourceID ever fetched stayed in the map forever, since
// Reset was never wired to the source lifecycle (TASK-136, ISS-129).
const limiterSweepThreshold = 1024

type Fetcher struct {
	client      *http.Client
	rateLimiter *RateLimiter
	userAgent   string
}

const defaultUserAgent = "Aurora/1.0"

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (r *RateLimiter) Allow(sourceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// Reclaim keys idle past the window so the requests map stays bounded by
	// sources active within a window rather than every source ever fetched
	// (Reset exists but was never called, so keys leaked forever; TASK-136,
	// ISS-129). Per-source slices are pruned below, so an expired key has no
	// remaining recent entry.
	if len(r.requests) >= limiterSweepThreshold {
		for key, ts := range r.requests {
			if len(ts) == 0 || ts[len(ts)-1].Before(windowStart) {
				delete(r.requests, key)
			}
		}
	}

	requests := r.requests[sourceID]
	valid := make([]time.Time, 0, len(requests))
	for _, t := range requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		r.requests[sourceID] = valid
		return false
	}

	valid = append(valid, now)
	r.requests[sourceID] = valid
	return true
}

func (r *RateLimiter) Reset(sourceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, sourceID)
}

func (r *RateLimiter) Remaining(sourceID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	count := 0
	for _, t := range r.requests[sourceID] {
		if t.After(windowStart) {
			count++
		}
	}

	remaining := r.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func NewFetcher(opts ...FetcherOption) *Fetcher {
	limit := viper.GetInt("http.rateLimit.requests")
	if limit <= 0 {
		limit = 10
	}

	// Routes the window and timeout through config.DurationSeconds (not raw
	// viper.GetDuration): a bare TOML number like `window = 10` / `timeout = 60`
	// becomes 10s / 60s instead of 10ns / 60ns, where a 60ns HTTP timeout would
	// fail every oracle fetch and a 10ns window would reset the limiter's budget
	// on every request (TASK-118, ISS-110).
	window := config.DurationSeconds("http.rateLimit.window", time.Minute)
	timeout := config.DurationSeconds("http.timeout", defaultHTTPTimeout)

	f := &Fetcher{
		client: &http.Client{
			Timeout:       timeout,
			CheckRedirect: checkRedirectAllowed,
		},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	for _, opt := range opts {
		opt(f)
	}
	f.client.Transport = newSecurityTransport(f.userAgent)
	return f
}

type FetcherOption func(*Fetcher)

// WithUserAgent overrides the User-Agent sent with every fetch. The default is
// defaultUserAgent; this exists so a caller can identify itself to a feed.
func WithUserAgent(ua string) FetcherOption {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

// NewFetcherWithTimeout builds a fetcher with an explicit request budget and
// client timeout. It is the only constructor that can report ErrInvalidTimeout:
// a non-positive timeout would make every oracle fetch fail immediately, so it
// is rejected instead of silently falling back to the default.
func NewFetcherWithTimeout(limit int, window, timeout time.Duration) (*Fetcher, error) {
	if timeout <= 0 {
		return nil, ErrInvalidTimeout
	}
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	f := &Fetcher{
		client: &http.Client{
			Timeout:       timeout,
			CheckRedirect: checkRedirectAllowed,
		},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	f.client.Transport = newSecurityTransport(f.userAgent)
	return f, nil
}

// WithRateLimit overrides the request budget. Non-positive values fall back to
// the same defaults NewFetcher applies (10 requests per minute), so a caller
// can pass config values through without pre-validating them.
func WithRateLimit(limit int, window time.Duration) FetcherOption {
	return func(f *Fetcher) {
		if limit <= 0 {
			limit = 10
		}
		if window <= 0 {
			window = time.Minute
		}
		f.rateLimiter = NewRateLimiter(limit, window)
	}
}

type securityTransport struct {
	transport http.RoundTripper
	userAgent string
	dialer    *net.Dialer
}

func newSecurityTransport(userAgent string) *securityTransport {
	return newSecurityTransportFrom(secureBaseTransport(), userAgent)
}

func newSecurityTransportFrom(transport *http.Transport, userAgent string) *securityTransport {
	dialer := newSecurityDialer()
	transport.DialContext = dialer.DialContext
	// A custom TLS dial hook would bypass DialContext (and therefore the
	// destination guard) for non-proxied HTTPS requests.
	transport.DialTLSContext = nil
	transport.DialTLS = nil //nolint:staticcheck // Intentional: this deprecated hook bypasses the guarded DialContext.
	if proxy := transport.Proxy; proxy != nil {
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			proxyURL, err := proxy(req)
			if err == nil && proxyURL != nil {
				if state, _ := req.Context().Value(proxyDialContextKey{}).(*proxyDialState); state != nil {
					state.proxied.Store(true)
				}
			}
			return proxyURL, err
		}
	}
	return &securityTransport{transport: transport, userAgent: userAgent, dialer: dialer}
}

// secureBaseTransport clones the standard transport so this fetcher owns its
// connection pool. If an embedding application replaced http.DefaultTransport
// with a non-standard RoundTripper, fall back to explicit net/http defaults
// rather than silently dropping the dial guard. A proxy selected by that
// transport is a trusted operator egress path: the dial guard sees the proxy
// address, so the proxy's own egress policy must complement the pre-request URL
// validation.
func secureBaseTransport() *http.Transport {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		return base.Clone()
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}

func newSecurityDialer() *net.Dialer {
	dialer := &net.Dialer{
		Timeout:        30 * time.Second,
		KeepAlive:      30 * time.Second,
		ControlContext: checkDialDestination,
	}
	return dialer
}

func (t *securityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Record the concrete proxy decision made by this transport. A private
	// endpoint is normal for an operator-configured proxy, while a direct
	// dial must still pass the destination guard.
	req = req.WithContext(context.WithValue(req.Context(), proxyDialContextKey{}, &proxyDialState{}))
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	if req.Header.Get("Content-Type") == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Request-ID", uuid.New().String())
	return t.transport.RoundTrip(req)
}

func (f *Fetcher) Get(url string) ([]byte, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", sanitizeFetchError(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrHTTPError, resp.StatusCode)
	}

	body, err := readBounded(resp.Body, maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

func (f *Fetcher) FetchData(source *oracle.DataSource) (*oracle.OracleData, error) {
	return f.FetchDataContext(context.Background(), source)
}

// FetchDataContext is the cancellable form of FetchData used by the oracle
// scheduler: the request is built with http.NewRequestWithContext so an
// in-flight fetch is interrupted when the scheduler is shut down, instead of
// blocking up to the client timeout while srv.Close() is tearing the DB pool
// down underneath it (TASK-134, ISS-127).
func (f *Fetcher) FetchDataContext(ctx context.Context, source *oracle.DataSource) (*oracle.OracleData, error) {
	return f.FetchDataWithValidationContext(ctx, source, false)
}

func (f *Fetcher) FetchDataWithValidationContext(ctx context.Context, source *oracle.DataSource, validateJSON bool) (*oracle.OracleData, error) {
	if !f.rateLimiter.Allow(source.ID) {
		return nil, fmt.Errorf("%w: source %s has exceeded rate limit", ErrRateLimited, source.ID)
	}

	if source.URL == "" {
		return nil, oracle.ErrInvalidSource
	}

	method := source.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", sanitizeFetchError(err))
	}

	// Apply the source's configured request headers. Without this the
	// persisted Headers field (e.g. an Authorization bearer for a private
	// price API) was silently ignored — the request went out with only the
	// transport defaults.
	if err := applySourceHeaders(req, source.Headers); err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", sanitizeFetchError(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrHTTPError, resp.StatusCode)
	}

	body, err := readBounded(resp.Body, maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) == 0 {
		return nil, ErrEmptyResponse
	}

	if validateJSON {
		if !json.Valid(body) {
			return nil, fmt.Errorf("%w: response is not valid JSON", ErrInvalidJSON)
		}
	}

	value := string(body)
	if source.Path != "" {
		var ok bool
		value, ok = extractByPath(string(body), source.Path)
		if !ok {
			return nil, oracle.ErrInvalidSource
		}
		// A resolved empty string is a legitimate (if unusual) value: ok is
		// true, so a healthy 200 whose leaf happens to be "" is recorded as a
		// data point instead of misreported as an invalid source (TASK-237,
		// ISS-235).
	}

	return &oracle.OracleData{
		ID:          uuid.New().String(),
		SourceID:    source.ID,
		Value:       value,
		RawResponse: string(body),
		Timestamp:   time.Now().Unix(),
	}, nil
}

// sanitizeFetchError removes URL-bearing net/http wrappers before an error is
// logged or exposed through Oracle feed health. Source URLs can contain
// credentials in userinfo, path segments, or query parameters; url.Error keeps
// the complete URL unless the password is explicitly redacted. Preserve the
// underlying cause so sentinel checks such as ErrBlockedDestination keep
// working without leaking the destination string.
func sanitizeFetchError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}
	cause := urlErr.Err
	if cause == nil {
		cause = errors.New("request failed")
	}
	return fmt.Errorf("request failed: %w", sanitizeFetchError(cause))
}

// extractByPath resolves a dotted path in a JSON object to its scalar leaf. It
// returns (value, false) — and the caller fails the fetch with ErrInvalidSource
// — when the body is not a JSON object, the path does not resolve, traversal
// hits a scalar before the path is exhausted, or the resolved node is a nested
// object/array/null (persisting any of those with %v would store Go
// representation garbage as the oracle value; TASK-237, ISS-235). A resolved
// empty string is a REAL value: ok stays true so a healthy 200 that carries ""
// is recorded instead of being miscategorized as an invalid source.
func extractByPath(jsonStr, path string) (string, bool) {
	var data map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		// The configured source is not a JSON object, so there is nothing to
		// extract: report a failed extraction rather than returning the entire
		// raw body as the field value (TASK-076, ISS-068).
		return "", false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		// Decode accepts one value at a time; require the response to contain
		// exactly that value so trailing JSON cannot be silently ignored.
		return "", false
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if v, exists := m[part]; exists {
				current = v
			} else {
				// Path part missing: fail closed instead of silently adopting
				// the whole body as the extracted value.
				return "", false
			}
		} else {
			return "", false
		}
	}

	switch result := current.(type) {
	case string:
		return result, true
	case json.Number:
		// UseNumber above keeps the exact JSON lexeme. Decoding into float64
		// rounded integers above 2^53 and silently changed the persisted value.
		return result.String(), true
	case bool:
		return fmt.Sprintf("%v", result), true
	default:
		// Nested object/array/null: not a scalar leaf — the operator's path
		// most likely needs a deeper key. Fail closed rather than persist
		// %v of a Go map/array on-chain.
		return "", false
	}
}

// readBounded reads up to max+1 bytes from r and returns ErrResponseTooLarge
// if the body exceeds max. Reading exactly max+1 (not max) is what
// makes this safe: a body of exactly max bytes is allowed, while a body
// one byte over is rejected without allocating the full payload.
func readBounded(r io.Reader, max int64) ([]byte, error) {
	lr := io.LimitReader(r, max+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > max {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

// applySourceHeaders parses the source's JSON map of request headers and sets
// them on the outgoing request. An empty string means no headers. Header names
// and values are taken from the JSON — an operator already holding source-add
// authority supplies them, so this is defense-in-depth, NOT the trust
// boundary — and CR/LF is rejected outright: Go JSON escape sequences ("\r\n")
// decode to literal CR/LF bytes, so a newline could otherwise be smuggled into
// a header line toward the upstream provider on the shared keep-alive
// connection. Modern net/http refuses to WRITE such a request (surfacing as a
// confusing runtime fetch error), so we fail at apply time with a clear
// invalid-source instead (TASK-239, ISS-237). An unparseable headers payload
// is likewise treated as invalid source data rather than silently dropping
// the operator's intent.
func applySourceHeaders(req *http.Request, headersJSON string) error {
	headers, err := oracle.ParseSourceHeaders(headersJSON)
	if err != nil {
		return err
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	return nil
}
