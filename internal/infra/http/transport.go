// The HTTP transport every fetch goes through: the guarded dialer, the header
// defaults, and the close hook that forwards keep-alive shutdown. The redirect
// policy stays in fetcher.go beside the request path it guards.
package http

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type securityTransport struct {
	transport http.RoundTripper
	userAgent string
	dialer    *net.Dialer
}

// CloseIdleConnections forwards the http.Client shutdown hook to the wrapped
// transport. Without this method, Client.CloseIdleConnections treats the
// security wrapper as unsupported and silently leaves every keep-alive socket
// open.
func (t *securityTransport) CloseIdleConnections() {
	if t == nil || t.transport == nil {
		return
	}
	if closer, ok := t.transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
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
	if _, configured := req.Header["User-Agent"]; !configured {
		req.Header.Set("User-Agent", t.userAgent)
	}
	if _, configured := req.Header["Accept"]; !configured {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	// Content-Type deliberately keeps its empty-means-absent fallback. Explicit
	// presence must remain distinguishable for User-Agent, where an empty value
	// tells Go not to send that header.
	if req.Header.Get("Content-Type") == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Request-ID", uuid.New().String())
	return t.transport.RoundTrip(req)
}
