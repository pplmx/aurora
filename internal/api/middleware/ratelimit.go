package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// windowBucket tracks request count within a fixed time window for one client.
type windowBucket struct {
	count int
	start time.Time
}

// limiterSweepThreshold is how many tracked keys are tolerated before Allow
// sweeps expired buckets. Together with window-bounded expiry this keeps the
// buckets map proportional to keys active within a window, never the total
// number of distinct keys ever seen (TASK-136, ISS-129).
const limiterSweepThreshold = 1024

// FixedWindowLimiter is a concurrency-safe fixed-window rate limiter keyed by
// an arbitrary client identifier (remote address or API key). It is the
// building block behind the REST API rate limiting middleware (v1.19).
//
// The limiter allows at most `limit` requests per `window` per key. On the
// `limit`+1th request within a window it rejects (returns false). The window is
// reset lazily per key so idle keys are forgotten and memory stays bounded by
// the number of active clients.
type FixedWindowLimiter struct {
	mu      sync.Mutex
	buckets map[string]*windowBucket
	limit   int
	window  time.Duration
	now     func() time.Time
}

// NewFixedWindowLimiter returns a limiter permitting `limit` requests per
// `window` per key. Positive limit/window required; values <= 0 fall back to
// sane defaults (120/min). now is injectable for deterministic tests.
func NewFixedWindowLimiter(limit int, window time.Duration, now func() time.Time) *FixedWindowLimiter {
	if limit <= 0 {
		limit = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &FixedWindowLimiter{
		buckets: make(map[string]*windowBucket),
		limit:   limit,
		window:  window,
		now:     now,
	}
}

// Allow reports whether a request for key is within the rate budget. It is
// safe for concurrent use.
func (l *FixedWindowLimiter) Allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	// Evict timers whose window fully elapsed so the buckets map stays bounded
	// by the number of keys active within a window instead of growing forever
	// with every distinct client/forwarded-IP ever seen (TASK-136, ISS-129).
	if len(l.buckets) >= limiterSweepThreshold {
		l.sweepExpired(now)
	}

	b := l.buckets[key]
	if b == nil || now.Sub(b.start) >= l.window {
		l.buckets[key] = &windowBucket{count: 1, start: now}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// sweepExpired deletes buckets whose window has fully elapsed. Called only
// while holding l.mu, from Allow when the map grows past
// limiterSweepThreshold.
func (l *FixedWindowLimiter) sweepExpired(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.start) >= l.window {
			delete(l.buckets, key)
		}
	}
}

// Reset clears all tracked state (mostly useful for tests).
func (l *FixedWindowLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = make(map[string]*windowBucket)
}

type peerIPKey struct{}

// PeerIP returns middleware that records the socket-level peer address in the
// request context BEFORE chi's RealIP middleware rewrites r.RemoteAddr from
// the X-Forwarded-For / X-Real-IP / True-Client-IP headers. It MUST be
// registered outer to middleware.RealIP. The rate limiter reads this value so
// its per-client budget is keyed on the untrusted, unspoofable direct peer
// rather than on a client-supplied header (v1.69, ISS-073).
func PeerIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), peerIPKey{}, r.RemoteAddr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func peerFromContext(r *http.Request) (string, bool) {
	peer, ok := r.Context().Value(peerIPKey{}).(string)
	return peer, ok
}

// trustedProxy is one entry of the operator's trusted-proxy allow-list: either
// a CIDR network or a single IP.
type trustedProxy struct {
	net    *net.IPNet
	single net.IP
}

func (p trustedProxy) contains(ip net.IP) bool {
	if p.net != nil {
		return p.net.Contains(ip)
	}
	return p.single != nil && p.single.Equal(ip)
}

func hostOnly(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// parseTrustedProxies decodes the configured allow-list into matchers.
// Malformed entries are dropped (fail-safe: an unparseable proxy is simply
// never trusted, which only ever over-restricts, never under-restricts).
func parseTrustedProxies(cfgs []string) []trustedProxy {
	out := make([]trustedProxy, 0, len(cfgs))
	for _, c := range cfgs {
		if _, ipnet, err := net.ParseCIDR(c); err == nil {
			out = append(out, trustedProxy{net: ipnet})
			continue
		}
		if ip := net.ParseIP(c); ip != nil {
			out = append(out, trustedProxy{single: ip})
		}
	}
	return out
}

func isTrusted(ip net.IP, proxies []trustedProxy) bool {
	for _, p := range proxies {
		if p.contains(ip) {
			return true
		}
	}
	return false
}

// clientRateKey returns the identifier used to rate-limit a request: the
// client's socket peer recorded by PeerIP (the true, unspoofable source).
// Only when that peer is in the configured trusted-proxy allow-list does the
// key fall back to r.RemoteAddr (which middleware.RealIP has already set from
// the proxy's X-Forwarded-For / X-Real-IP / True-Client-IP header) — i.e. the
// forwarded client behind our own proxy. A directly-connected attacker who
// rotates those headers just rotates nothing: the key stays their peer, so
// the budget still applies (v1.69, ISS-073).
func clientRateKey(r *http.Request, trusted []trustedProxy) string {
	peerHost := hostOnly(r.RemoteAddr)
	if peer, ok := peerFromContext(r); ok {
		peerHost = hostOnly(peer)
	}
	if ip := net.ParseIP(peerHost); ip != nil && isTrusted(ip, trusted) {
		return hostOnly(r.RemoteAddr)
	}
	return peerHost
}

// RateLimit returns middleware that rejects clients exceeding their per-window
// budget with HTTP 429 and a Retry-After header. Rejected responses are JSON
// and increment no handler-specific state. The middleware is safe for
// concurrent use. trustedProxies names the reverse proxies/CDNs whose
// forwarded client headers the limiter may believe (see clientRateKey).
//
// Register PeerIP outer to middleware.RealIP (as the router does) so the
// socket peer is captured before header substitution.
func RateLimit(limiter *FixedWindowLimiter, trustedProxies []string) func(http.Handler) http.Handler {
	trusted := parseTrustedProxies(trustedProxies)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientRateKey(r, trusted)) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded",
					"code":  "RATE_LIMITED",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
