package oracle

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// defaultQueryLimit is the default maximum number of data records to return
// when querying oracle data.
const defaultQueryLimit = 10

// MaxSourceIntervalSeconds bounds a fetch-scheduling interval. The scheduler
// converts Interval to a time.Duration (`int64` ns); an interval >= ~9.22e9 s
// would overflow to a negative duration that the scheduler treats as "always
// due", turning that source into a fetch-storm on every check tick (TASK-232,
// ISS-230). 30 days is far above any realistic polling cadence and comfortably
// below the overflow point.
const MaxSourceIntervalSeconds = 30 * 24 * 60 * 60

// Length bounds for oracle source free-text fields. AddSource previously
// persisted these unbounded (name/type/method/path/headers — only interval
// and URL scheme were checked) while the token/voting/lottery surfaces cap
// their free-text inputs; a key-holding caller could grow rows and list/detail
// responses without bound. These caps are enforced at the shared domain edge
// so REST/CLI/TUI/web all inherit them (TASK-271, ISS-267).
const (
	MaxSourceNameLength   = 100
	MaxSourceTypeLength   = 50
	MaxSourceMethodLength = 10
	MaxSourcePathLength   = 500
	MaxSourceHeadersLen   = 2000
	MaxSourceURLLength    = 2000
)

// allowedSourceSchemes is the set of URL schemes an oracle source
// may use. We block file:// (would let a hostile source read the
// host filesystem), javascript: (XSS-shaped payloads, not that the
// CLI renders them, but defense in depth), data: (can encode huge
// payloads), and exotic schemes Go's http.Client would just refuse
// anyway. http and https are the legitimate use cases.
var allowedSourceSchemes = map[string]struct{}{
	"http":  {},
	"https": {},
}

// blockedCIDRs are the address ranges a source URL's host must never resolve
// to: loopback, RFC1918 private, link-local (incl. cloud metadata
// 169.254.169.254), CGNAT, IPv6 loopback/link-local/unique-local, and the
// unspecified address. A source pointing at one of these would let the
// fetcher reach internal/private services hosting the data — the SSRF
// primitive. This mirrors the redirect guard in internal/infra/http; the
// domain-level check additionally covers the INITIAL request, which the
// fetcher's redirect policy alone does not.
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

// hostBlocked reports whether a URL host (optionally host:port) resolves to
// a blocked address. IP literals are checked directly and hostnames are
// resolved (bounded by a short timeout) so `localhost`, cloud-metadata names,
// or any hostname that resolves into private space is rejected too.
//
// A host that cannot be resolved (or resolves to no addresses) is reported as
// NOT blocked: it is unreachable, so it cannot be a conduit to a private
// address — the fetch would simply fail at dial time. Rejecting all
// unresolvable hosts would wrongly refuse otherwise-valid public endpoints on
// hosts/environments where DNS is temporarily unavailable.
func hostBlocked(host string) (bool, error) {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// URL hosts wrap IPv6 literals in brackets ("[::1]"), and net.ParseIP does
	// not accept brackets. Strip them so a literal loopback/private IPv6 host
	// is recognized deterministically. Without this, "[::1]", "[fc00::1]" and
	// "[fe80::1]" fell through to the DNS-only path, which cannot resolve a
	// bracketed literal and so reported them as NOT blocked — an SSRF bypass.
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedIP(ip), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, nil // unresolvable → not a usable SSRF vector
	}
	for _, a := range addrs {
		if isBlockedIP(a.IP) {
			return true, nil
		}
	}
	return false, nil
}

// validateSourceURL rejects URLs that would let a hostile source
// escape the HTTP(S) boundary. Returns nil if the URL is acceptable.
//
// AddSource callers should call this before persisting; the
// validation is part of the service contract, not the repo, so the
// same rules apply regardless of storage backend.
func validateSourceURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if _, ok := allowedSourceSchemes[scheme]; !ok {
		return fmt.Errorf("disallowed scheme %q (only http/https allowed)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	// SSRF: reject any host that is, or resolves to, a private/loopback
	// address. Without this, a source literally pointing at
	// 169.254.169.254 (cloud metadata) or 127.0.0.1 would pass the
	// scheme-only check and the fetcher would dial it directly.
	blocked, err := hostBlocked(u.Host)
	if err != nil {
		return fmt.Errorf("verify host: %w", err)
	}
	if blocked {
		return fmt.Errorf("host %q resolves to a blocked (private/internal) address", u.Host)
	}
	return nil
}

// ValidateSourceURL is the exported contract-level URL validator, used both
// when a source is added (AddSource) and re-applied at fetch time by the app
// layer so a source whose hostname has since been rebound into private /
// loopback / cloud-metadata space is refused before any dial (TASK-067,
// ISS-059). Keeping the SSRF policy in one place means the domain rule is
// authoritative regardless of which caller/use case triggers a fetch.
func ValidateSourceURL(raw string) error {
	return validateSourceURL(raw)
}

type Service interface {
	AddSource(source *DataSource) error
	EnableSource(id string) error
	DisableSource(id string) error
	DeleteSource(id string) error
	FetchData(source *DataSource) (*OracleData, error)
	QueryData(sourceID string, limit int) ([]*OracleData, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) AddSource(source *DataSource) error {
	if source.Name == "" {
		return ErrInvalidSource
	}
	// Length bounds mirror the token/voting validators: previously only non-empty
	// name + interval + URL scheme were checked, so a key-holding caller could
	// grow rows/list responses without bound (TASK-271, ISS-267).
	switch {
	case len(source.Name) > MaxSourceNameLength:
		return fmt.Errorf("%w: name too long", ErrInvalidSource)
	case len(source.Type) > MaxSourceTypeLength:
		return fmt.Errorf("%w: type too long", ErrInvalidSource)
	case len(source.Method) > MaxSourceMethodLength:
		return fmt.Errorf("%w: method too long", ErrInvalidSource)
	case len(source.Path) > MaxSourcePathLength:
		return fmt.Errorf("%w: path too long", ErrInvalidSource)
	case len(source.Headers) > MaxSourceHeadersLen:
		return fmt.Errorf("%w: headers too long", ErrInvalidSource)
	case len(source.URL) > MaxSourceURLLength:
		return fmt.Errorf("%w: url too long", ErrInvalidSource)
	}
	// A fetch-scheduling interval cannot be negative; only ==0 is defaulted
	// (to 60) by the use case, so reject negatives here at the contract
	// boundary alongside the other AddSource validations. An upper bound also
	// lives here: values >= ~9.22e9 s overflow the scheduler's time.Duration
	// arithmetic to a negative interval (always-due -> fetch storm), and anything
	// above 30 days is a misconfiguration regardless (TASK-232, ISS-230).
	if source.Interval < 0 || source.Interval > MaxSourceIntervalSeconds {
		return fmt.Errorf("%w: invalid interval", ErrInvalidSource)
	}
	if err := validateSourceURL(source.URL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSource, err)
	}
	if source.ID == "" {
		source.ID = generateID()
	}
	source.CreatedAt = time.Now().Unix()
	source.Enabled = true
	return s.repo.SaveSource(source)
}

func (s *service) EnableSource(id string) error {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return err
	}
	if source == nil {
		return ErrSourceNotFound
	}
	source.Enabled = true
	return s.repo.UpdateSource(source)
}

func (s *service) DisableSource(id string) error {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return err
	}
	if source == nil {
		return ErrSourceNotFound
	}
	source.Enabled = false
	return s.repo.UpdateSource(source)
}

func (s *service) DeleteSource(id string) error {
	// Deleting an unknown id must fail loudly (REST 404, CLI non-zero exit),
	// not report success — mirror the EnableSource/DisableSource existence
	// check so the handler's ErrSourceNotFound branch is reachable (TASK-233,
	// ISS-231). The delete itself remains an idempotent DELETE by id.
	source, err := s.repo.GetSource(id)
	if err != nil {
		return err
	}
	if source == nil {
		return ErrSourceNotFound
	}
	return s.repo.DeleteSource(id)
}

func (s *service) FetchData(source *DataSource) (*OracleData, error) {
	if !source.Enabled {
		return nil, ErrSourceDisabled
	}
	data := &OracleData{
		ID:          generateID(),
		SourceID:    source.ID,
		Value:       "sample-value",
		RawResponse: "{}",
		Timestamp:   time.Now().Unix(),
		BlockHeight: 0,
	}
	return data, s.repo.SaveData(data)
}

func (s *service) QueryData(sourceID string, limit int) ([]*OracleData, error) {
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	return s.repo.GetDataBySource(sourceID, limit)
}

// generateID produces a unique identifier for oracle sources and data
// entries. The previous implementations used time.Now() with second, then
// nanosecond, precision — but time.Now() resolution is not monotonic enough
// across all platforms (Windows can return identical values for rapid calls),
// so a timestamp alone can still collide, silently overwriting a row keyed by
// that ID. We keep a timestamp prefix for human readability/debugging and
// append a 128-bit cryptographically random suffix, making collisions
// negligible regardless of platform clock resolution.
func generateID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand never fails in practice; fall back to a monotonic
		// nanosecond timestamp rather than returning an empty ID.
		return time.Now().Format("20060102150405.000000000")
	}
	return time.Now().Format("20060102150405.000000000") + "-" + hex.EncodeToString(b[:])
}

type OracleError struct {
	Message string
}

func (e *OracleError) Error() string {
	return e.Message
}

var (
	ErrInvalidSource  = &OracleError{Message: "invalid source"}
	ErrSourceNotFound = &OracleError{Message: "source not found"}
	ErrSourceDisabled = &OracleError{Message: "source is disabled"}
)
