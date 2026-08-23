package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/spf13/viper"
)

const defaultHTTPTimeout = 10 * time.Second

var (
	ErrRateLimited    = errors.New("rate limit exceeded")
	ErrInvalidTimeout = errors.New("timeout must be greater than 0")
	ErrHTTPError      = errors.New("http error response")
	ErrInvalidJSON    = errors.New("invalid JSON response")
	ErrEmptyResponse  = errors.New("empty response body")
	ErrPathExtraction = errors.New("path extraction failed")
	// ErrResponseTooLarge is returned when a remote response exceeds
	// maxResponseBytes. Without a cap, a malicious or buggy oracle
	// source could OOM the process by streaming a multi-GB body.
	ErrResponseTooLarge = errors.New("response body exceeds maximum size")
	// ErrBlockedDestination is returned when a redirect would take the
	// fetcher to an internal/private address (SSRF) or to a non-HTTP(S)
	// scheme. Blocks loops where a compromised or hostile source URL
	// bounces us at 127.0.0.1, 169.254.169.254, RFC1918 space, etc.
	ErrBlockedDestination = errors.New("redirect to blocked destination")
)

// maxResponseBytes caps the size of any oracle-source response we read.
// 10 MiB is generous for price feeds / on-chain data and small enough
// to keep the process memory-safe even under sustained attack.
const maxResponseBytes = 10 * 1024 * 1024

// blockedCIDRs are the address ranges the fetcher refuses to be redirected
// into: loopback, RFC1918 private, link-local (incl. cloud metadata
// 169.254.169.254), CGNAT, IPv6 loopback/link-local/unique-local, and the
// unspecified address. A query to one of these through a redirect is the
// classic SSRF primitive for probing or reading internal services.
var blockedCIDRs = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
}

func isBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true // unparseable address — refuse rather than guess
	}
	addr = addr.Unmap()
	for _, p := range blockedCIDRs {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// isBlockedHost reports whether host (optionally host:port) is an IP literal
// inside a blocked range.
func isBlockedHost(host string) bool {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return isBlockedIP(ip)
}

// redirectHostBlocked decides whether a redirect target host is out of bounds.
// It handles IP literals directly and resolves hostnames so redirects aimed at
// localhost, cloud-metadata names, or other internal DNS entries are refused
// just like literal private IPs. An unresolved/invalid host is treated as
// blocked (we must not follow a redirect whose destination we cannot verify).
// Resolution is bounded by a short timeout so a poisoned/stuck resolver cannot
// hang the fetch indefinitely.
func redirectHostBlocked(host string) (bool, error) {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedIP(ip), nil
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
// (default behaviour: follow, cap at 10 hops) for everything else.
func checkRedirectAllowed(req *http.Request, via []*http.Request) error {
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
	return nil
}

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

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

	window := viper.GetDuration("http.rateLimit.window")
	if window <= 0 {
		window = time.Minute
	}

	timeout := viper.GetDuration("http.timeout")
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

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
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f
}

func NewFetcherWithConfig(limit int, window time.Duration) *Fetcher {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	f := &Fetcher{
		client: &http.Client{
			Timeout:       defaultHTTPTimeout,
			CheckRedirect: checkRedirectAllowed,
		},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f
}

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
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f, nil
}

type FetcherOption func(*Fetcher)

func WithUserAgent(ua string) FetcherOption {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

type securityTransport struct {
	transport http.RoundTripper
	userAgent string
}

func (t *securityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if req.Header.Get("Content-Type") == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Request-ID", uuid.New().String())
	return t.transport.RoundTrip(req)
}

func (f *Fetcher) Get(url string) ([]byte, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
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
	return f.FetchDataWithValidation(source, false)
}

func (f *Fetcher) FetchDataWithValidation(source *oracle.DataSource, validateJSON bool) (*oracle.OracleData, error) {
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

	req, err := http.NewRequest(method, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
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
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d for URL %s", ErrHTTPError, resp.StatusCode, source.URL)
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
		value = extractByPath(string(body), source.Path)
		if value == "" {
			return nil, oracle.ErrInvalidSource
		}
	}

	return &oracle.OracleData{
		ID:          uuid.New().String(),
		SourceID:    source.ID,
		Value:       value,
		RawResponse: string(body),
		Timestamp:   time.Now().Unix(),
	}, nil
}

func extractByPath(jsonStr, path string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// The configured source is not a JSON object, so there is nothing to
		// extract: report a failed extraction (empty) rather than returning
		// the entire raw body as the field value (TASK-076, ISS-068).
		return ""
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
				return ""
			}
		} else {
			return ""
		}
	}

	if result, ok := current.(string); ok {
		return result
	}
	if result, ok := current.(float64); ok {
		return fmt.Sprintf("%v", result)
	}
	return fmt.Sprintf("%v", current)
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
// and values are taken verbatim from the JSON (settable only by a source
// operator / API caller, so newline injection is not an out-of-band vector
// here — JSON cannot carry a raw CRLF untransformed). An unparseable headers
// payload is treated as invalid source data rather than silently dropping
// the operator's intent.
func applySourceHeaders(req *http.Request, headersJSON string) error {
	if strings.TrimSpace(headersJSON) == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
		return fmt.Errorf("%w: invalid headers json", oracle.ErrInvalidSource)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return nil
}
