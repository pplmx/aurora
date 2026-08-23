package middleware

import (
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

// Reset clears all tracked state (mostly useful for tests).
func (l *FixedWindowLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = make(map[string]*windowBucket)
}

// clientRateKey returns the identifier used to rate-limit a request: the
// client's remote address host, falling back to the peer address if it has no
// port. The X-Real-IP / X-Forwarded-For headers from the RealIP middleware
// already substituted r.RemoteAddr, so this is the effective client.
func clientRateKey(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host
}

// RateLimit returns middleware that rejects clients exceeding their per-window
// budget with HTTP 429 and a Retry-After header. Rejected responses are JSON
// and increment no handler-specific state. The middleware is safe for
// concurrent use.
func RateLimit(limiter *FixedWindowLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientRateKey(r)) {
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
