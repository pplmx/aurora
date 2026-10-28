package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// observeRemoteAddr runs PeerIP -> ClientIP exactly as the router does and
// reports the r.RemoteAddr the inner handler would log.
func observeRemoteAddr(trusted []string, remoteAddr string, headers map[string]string) string {
	var seen string
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.RemoteAddr
	})
	handler = ClientIP(trusted)(handler)
	handler = PeerIP(handler)

	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	handler.ServeHTTP(httptest.NewRecorder(), r)
	return seen
}

// TestClientIP_IgnoresForwardedHeadersFromUntrustedPeer is the security half:
// with no trusted proxy configured, a client that forges X-Forwarded-For,
// X-Real-IP or True-Client-IP must still be logged under its own socket
// address. chi's RealIP (what this replaced) rewrote r.RemoteAddr from those
// headers for every peer and is deprecated for exactly that spoofing bug.
func TestClientIP_IgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	headers := map[string]string{
		"X-Forwarded-For": "10.0.0.1",
		"X-Real-IP":       "10.0.0.2",
		"True-Client-IP":  "10.0.0.3",
	}
	got := observeRemoteAddr(nil, "203.0.113.42:5555", headers)
	assert.Equal(t, "203.0.113.42:5555", got,
		"an untrusted peer must never have its forwarded headers believed")

	// Same for a peer that is merely adjacent to the allow-list.
	got = observeRemoteAddr([]string{"203.0.113.10"}, "203.0.113.42:5555", headers)
	assert.Equal(t, "203.0.113.42:5555", got,
		"only the exact configured proxy may be trusted")
}

// TestClientIP_BelievesChainBehindTrustedProxy covers the useful case: the
// operator's own proxy forwards for a real client, so logs carry the client
// address (and keep the peer's port shape).
func TestClientIP_BelievesChainBehindTrustedProxy(t *testing.T) {
	got := observeRemoteAddr([]string{"203.0.113.10"},
		"203.0.113.10:9999", map[string]string{"X-Forwarded-For": "198.51.100.7"})
	assert.Equal(t, "198.51.100.7:9999", got)
}

// TestClientIP_WalksChainRightToLeft pins the parsing direction. In
// "spoofed, real-client, our-proxy" the attacker controls the leftmost entry;
// only the rightmost untrusted hop is attributable, so it must win.
func TestClientIP_WalksChainRightToLeft(t *testing.T) {
	got := observeRemoteAddr([]string{"203.0.113.10"},
		"203.0.113.10:9999",
		map[string]string{"X-Forwarded-For": "10.9.9.9, 198.51.100.7"})
	assert.Equal(t, "198.51.100.7:9999", got,
		"the first untrusted hop from the right is the only attributable client")
}

// TestClientIP_SkipsChainedTrustedProxies covers a proxy pair: the header
// lists the client, then our inner proxy. Both proxies must be skipped so the
// client address is reported.
func TestClientIP_SkipsChainedTrustedProxies(t *testing.T) {
	trusted := []string{"203.0.113.0/24"}
	got := observeRemoteAddr(trusted, "203.0.113.10:9999",
		map[string]string{"X-Forwarded-For": "198.51.100.7, 203.0.113.11"})
	assert.Equal(t, "198.51.100.7:9999", got)
}

// TestClientIP_RejectsMalformedChain fails closed: an unparseable hop makes
// the attribution unknowable, so the socket peer is reported rather than a
// partially-trusted guess.
func TestClientIP_RejectsMalformedChain(t *testing.T) {
	got := observeRemoteAddr([]string{"203.0.113.10"}, "203.0.113.10:9999",
		map[string]string{"X-Forwarded-For": "198.51.100.7, not-an-ip"})
	assert.Equal(t, "203.0.113.10:9999", got)
}

// TestClientIP_FallsBackToSingleValueHeaders keeps the non-standard headers
// working behind a trusted proxy, but only there.
func TestClientIP_FallsBackToSingleValueHeaders(t *testing.T) {
	for _, header := range []string{"X-Real-IP", "True-Client-IP"} {
		got := observeRemoteAddr([]string{"203.0.113.10"}, "203.0.113.10:9999",
			map[string]string{header: "198.51.100.7"})
		assert.Equal(t, "198.51.100.7:9999", got, "%s must name the client", header)
	}
}

// TestClientIP_TrustsIPv6Proxy pins CIDR matching for a v6 allow-list entry,
// since the router's configuration is CIDR-based and v6 peers arrive
// bracketed in r.RemoteAddr.
func TestClientIP_TrustsIPv6Proxy(t *testing.T) {
	got := observeRemoteAddr([]string{"::1/128"}, "[::1]:9999",
		map[string]string{"X-Forwarded-For": "2001:db8::7"})
	require.Equal(t, "[2001:db8::7]:9999", got)
}

// TestClientIP_BoundsForwardedChain keeps a trusted proxy's append-only header
// from making request middleware walk an attacker-sized chain. A client plus
// 31 trusted hops fits the limit; one more hop is malformed and falls back to
// the socket peer instead of reporting the leftmost caller-supplied address.
func TestClientIP_BoundsForwardedChain(t *testing.T) {
	trusted := []string{"203.0.113.0/24"}
	chain := func(proxyHops int) string {
		hops := make([]string, 0, proxyHops+1)
		hops = append(hops, "198.51.100.7")
		for i := 0; i < proxyHops; i++ {
			hops = append(hops, "203.0.113.11")
		}
		return strings.Join(hops, ", ")
	}

	require.Equal(t, "198.51.100.7:9999",
		observeRemoteAddr(trusted, "203.0.113.10:9999", map[string]string{
			"X-Forwarded-For": chain(maxForwardedForHops - 1),
		}))
	require.Equal(t, "203.0.113.10:9999",
		observeRemoteAddr(trusted, "203.0.113.10:9999", map[string]string{
			"X-Forwarded-For": chain(maxForwardedForHops),
		}))
}
