package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP rewrites r.RemoteAddr to the client address when — and only when —
// the direct TCP peer is one of the operator's trusted proxies. It replaces
// chi's middleware.RealIP, which was deprecated for trusting
// X-Forwarded-For / X-Real-IP / True-Client-IP unconditionally (GHSA-3fxj-
// 6jh8-hvhx and follow-ups): a directly-connected client could set any of
// those headers and be logged — and, before PeerIP existed, rate-limited —
// under an address it never owned.
//
// The trust model matches clientRateKey: PeerIP must be registered outer so
// the socket peer is snapshotted before this middleware runs. The forwarded
// chain is then walked right-to-left, skipping hops that are themselves
// trusted proxies, and the first untrusted address wins. Leftmost-first
// parsing (what chi did) hands the attacker control: everything left of the
// first untrusted hop is client-supplied.
//
// With no trusted proxies configured (the default) r.RemoteAddr is left
// untouched, so forwarded headers can never change what the API reports.
func ClientIP(trustedProxies []string) func(http.Handler) http.Handler {
	trusted := parseTrustedProxies(trustedProxies)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ip := forwardedClientIP(r, trusted); ip != "" {
				// Keep the peer's port so the log line still looks like an
				// address:port, the shape every other layer expects.
				_, port, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					port = "0"
				}
				r.RemoteAddr = net.JoinHostPort(ip, port)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// forwardedClientIP returns the address the request was forwarded for, or ""
// when the peer is not a trusted proxy / no usable header is present.
func forwardedClientIP(r *http.Request, trusted []trustedProxy) string {
	peer, ok := peerFromContext(r)
	if !ok {
		peer = r.RemoteAddr
	}
	peerIP := net.ParseIP(hostOnly(peer))
	if peerIP == nil || !isTrusted(peerIP, trusted) {
		return ""
	}

	// X-Forwarded-For is the standard chain: client, proxy1, proxy2, ...
	if chain := r.Header.Get("X-Forwarded-For"); chain != "" {
		hops := strings.Split(chain, ",")
		for i := len(hops) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(hops[i]))
			if ip == nil {
				// A malformed hop means the rest of the chain cannot be
				// attributed; fall back to the peer rather than guess.
				return ""
			}
			if !isTrusted(ip, trusted) {
				return ip.String()
			}
		}
		// Every hop is a trusted proxy (e.g. a single-proxy deployment whose
		// header carries only the proxy): no client address to report.
		return ""
	}

	for _, header := range []string{"X-Real-IP", "True-Client-IP"} {
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get(header))); ip != nil {
			return ip.String()
		}
	}
	return ""
}
