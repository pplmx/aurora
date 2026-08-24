package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowLimiter_LimitAndWindowReset(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewFixedWindowLimiter(3, time.Minute, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("request %d within budget should be allowed", i+1)
		}
	}
	if l.Allow("ip") {
		t.Fatal("4th request should be rejected")
	}

	// A different key is unaffected.
	if !l.Allow("other") {
		t.Fatal("different key should have its own budget")
	}

	// Advancing past the window resets the original key.
	now = now.Add(61 * time.Second)
	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("request %d in new window should be allowed", i+1)
		}
	}
}

func TestFixedWindowLimiter_Concurrent(t *testing.T) {
	l := NewFixedWindowLimiter(50, time.Minute, nil)
	const goroutines = 20
	const each = 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				if l.Allow(fmt.Sprintf("client-%d", id)) {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}(g)
	}
	wg.Wait()
	// 20 clients * 5 each = 100, all under the per-client limit of 50.
	assert.Equal(t, goroutines*each, allowed, "every allowed request should be within its own client budget")
}

// rateLimitChain composes the middleware exactly as the router does (minus
// logging/metrics): PeerIP outer (snapshots the socket peer before header
// substitution), then chi's RealIP (rewrites r.RemoteAddr from forwarded
// headers), then RateLimit innermost.
func rateLimitChain(limiter *FixedWindowLimiter, trusted []string) http.Handler {
	return PeerIP(middleware.RealIP(RateLimit(limiter, trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))))
}

func TestRateLimit_MiddlewareReturns429(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewFixedWindowLimiter(2, time.Minute, func() time.Time { return now })
	handler := rateLimitChain(l, nil)

	// httptest requests share a RemoteAddr, so keying is by that address.
	req := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr
	}

	require.Equal(t, http.StatusOK, req().Code)
	require.Equal(t, http.StatusOK, req().Code)
	third := req()
	assert.Equal(t, http.StatusTooManyRequests, third.Code)
	assert.Equal(t, "60", third.Header().Get("Retry-After"))
}

// TestRateLimit_RejectsSpoofedForwardedHeaderRotation is the ISS-073
// regression: with no trusted proxy configured, the budget is keyed on the
// true socket peer, so an attacker rotating X-Forwarded-For / X-Real-IP /
// True-Client-IP per request passes through the same bucket and hits 429.
// Pre-fix the key came from r.RemoteAddr AFTER RealIP trusted the header, so
// every rotation produced a fresh key and the limit was trivially bypassed.
func TestRateLimit_RejectsSpoofedForwardedHeaderRotation(t *testing.T) {
	now := time.Unix(2000, 0)
	l := NewFixedWindowLimiter(2, time.Minute, func() time.Time { return now })
	handler := rateLimitChain(l, nil)

	send := func(hdr string) int {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("X-Forwarded-For", hdr)
		// One attacker, one fixed socket peer — the part they cannot spoof.
		r.RemoteAddr = "203.0.113.42:5555"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr.Code
	}

	require.Equal(t, http.StatusOK, send("10.0.0.1"))
	require.Equal(t, http.StatusOK, send("10.0.0.2"))
	// Third request from the same peer, fourth distinct spoofed header: the
	// header rotation must NOT grant a fresh budget.
	assert.Equal(t, http.StatusTooManyRequests, send("10.0.0.3"))
	assert.Equal(t, http.StatusTooManyRequests, send("10.0.0.4"))
}

// TestRateLimit_TrustsForwardedClientBehindConfiguredProxy pins the other
// side of the trust model: when the socket peer IS the configured reverse
// proxy, chi RealIP's rewritten r.RemoteAddr (proxy's X-Forwarded-For) is the
// per-client key — so distinct real clients behind OUR proxy keep distinct
// budgets (no over-limiting from one shared proxy IP).
func TestRateLimit_TrustsForwardedClientBehindConfiguredProxy(t *testing.T) {
	now := time.Unix(3000, 0)
	l := NewFixedWindowLimiter(1, time.Minute, func() time.Time { return now })
	handler := rateLimitChain(l, []string{"203.0.113.10"})

	send := func(hdr string) int {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("X-Forwarded-For", hdr)
		// The operator's own proxy as the direct peer.
		r.RemoteAddr = "203.0.113.10:9999"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr.Code
	}

	// Limit is 1 per client; two different forwarded clients both pass, each
	// with its own budget.
	require.Equal(t, http.StatusOK, send("198.51.100.7"))
	require.Equal(t, http.StatusOK, send("198.51.100.8"))
	// The same forwarded client reuses the SAME budget: second hit is 429.
	assert.Equal(t, http.StatusTooManyRequests, send("198.51.100.7"))
}

// TestRateLimit_UntrustedProxyPeerOverRestricts not the peer's own proxy:
// a spoofing client directly reaching the server claiming a proxy address in
// the list is still keyed on... its own peer unless the peer matches. A
// directly-connected attacker whose peer is NOT trusted cannot claim a
// trusted proxy's clients (their socket peer stays their own).
func TestRateLimit_UntrustedPeerNeverTrustsHeaders(t *testing.T) {
	now := time.Unix(4000, 0)
	l := NewFixedWindowLimiter(1, time.Minute, func() time.Time { return now })
	handler := rateLimitChain(l, []string{"203.0.113.10"})

	send := func(remoteAddr, hdr string) int {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("X-Forwarded-For", hdr)
		r.RemoteAddr = remoteAddr
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr.Code
	}

	// An attacker NOT connecting from the trusted proxy still keys on their
	// own peer regardless of the header they forge.
	require.Equal(t, http.StatusOK, send("8.8.4.4:1234", "198.51.100.7"))
	// Same peer, forging a different forwarded client: blocked.
	assert.Equal(t, http.StatusTooManyRequests, send("8.8.4.4:1234", "198.51.100.8"))
}
