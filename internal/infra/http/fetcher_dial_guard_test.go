package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFetcherConstructorsInstallDialGuard(t *testing.T) {
	tests := []struct {
		name   string
		create func() (*Fetcher, error)
	}{
		{name: "NewFetcher", create: func() (*Fetcher, error) { return NewFetcher(), nil }},
		{
			name: "NewFetcher with rate-limit option",
			create: func() (*Fetcher, error) {
				return NewFetcher(WithRateLimit(10, time.Minute)), nil
			},
		},
		{
			name: "NewFetcherWithTimeout",
			create: func() (*Fetcher, error) {
				return NewFetcherWithTimeout(10, time.Minute, time.Second)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher, err := tt.create()
			if err != nil {
				t.Fatalf("create fetcher: %v", err)
			}
			security, ok := fetcher.client.Transport.(*securityTransport)
			if !ok {
				t.Fatalf("transport = %T, want *securityTransport", fetcher.client.Transport)
			}
			if security.dialer == nil || security.dialer.ControlContext == nil {
				t.Fatal("fetcher transport has no guarded dialer")
			}
		})
	}
}

func TestNewFetcher_DialContextBlocksPrivateDestination(t *testing.T) {
	fetcher := NewFetcher()
	security, ok := fetcher.client.Transport.(*securityTransport)
	if !ok {
		t.Fatalf("default transport = %T, want *securityTransport", fetcher.client.Transport)
	}
	transport, ok := security.transport.(*http.Transport)
	if !ok {
		t.Fatalf("inner transport = %T, want *http.Transport", security.transport)
	}
	if transport.DialContext == nil {
		t.Fatal("inner transport has no DialContext")
	}
	if security.dialer == nil {
		t.Fatal("security transport has no guarded dialer")
	}

	// net.Dialer runs ControlContext only after the OS socket exists. Sandboxes
	// may reject that earlier, so exercise the attached policy directly and then
	// run the real dial where loopback sockets are available.
	err := security.dialer.ControlContext(context.Background(), "tcp", "127.0.0.1:1", nil)
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("ControlContext(127.0.0.1:1) error = %v, want ErrBlockedDestination", err)
	}
	if loopbackBlocked {
		return
	}

	conn, err := transport.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	if conn != nil {
		_ = conn.Close()
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("DialContext(127.0.0.1:1) error = %v, want ErrBlockedDestination", err)
	}
}

func TestCheckDialDestination_AllowsTrustedProxyEndpoint(t *testing.T) {
	dialer := newSecurityDialer()
	state := &proxyDialState{}
	state.proxied.Store(true)
	ctx := context.WithValue(context.Background(), proxyDialContextKey{}, state)
	if err := dialer.ControlContext(ctx, "tcp", "10.0.0.5:3128", nil); err != nil {
		t.Fatalf("trusted proxy dial = %v, want nil", err)
	}
}

func TestNewSecurityTransportFromOverridesTLSDialHooks(t *testing.T) {
	base := &http.Transport{
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, nil
		},
		DialTLS: func(string, string) (net.Conn, error) { //nolint:staticcheck // Construct the bypass this regression must reject.
			return nil, nil
		},
	}

	security := newSecurityTransportFrom(base, "test")
	if base.DialTLSContext != nil {
		t.Fatal("DialTLSContext still bypasses the destination guard")
	}
	if base.DialTLS != nil { //nolint:staticcheck // Verify the deprecated bypass was cleared.
		t.Fatal("DialTLS still bypasses the destination guard")
	}
	if security.dialer == nil || security.dialer.ControlContext == nil {
		t.Fatal("security transport has no guarded dialer")
	}
}

func TestNewSecurityTransportFromTracksConfiguredProxy(t *testing.T) {
	proxyURL := &url.URL{Scheme: "http", Host: "10.0.0.5:3128"}
	base := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
	}
	security := newSecurityTransportFrom(base, "test")

	req, err := http.NewRequest(http.MethodGet, "http://example.com/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	state := &proxyDialState{}
	req = req.WithContext(context.WithValue(req.Context(), proxyDialContextKey{}, state))

	selected, err := base.Proxy(req)
	if err != nil {
		t.Fatalf("proxy lookup: %v", err)
	}
	if selected != proxyURL {
		t.Fatalf("proxy = %v, want %v", selected, proxyURL)
	}
	if !state.proxied.Load() {
		t.Fatal("configured proxy decision was not recorded")
	}
	if err := security.dialer.ControlContext(req.Context(), "tcp", proxyURL.Host, nil); err != nil {
		t.Fatalf("configured proxy dial = %v, want nil", err)
	}
}

// TestSecureBaseTransportKeepsTheDialGuard covers both branches of the base
// transport choice. The fetcher builds its dial guard into the transport, so a
// base that dropped that guard would leave the SSRF check applied only to the
// pre-request URL validation — the gap the dial guard exists to close when DNS
// changes between validation and connect.
func TestSecureBaseTransportKeepsTheDialGuard(t *testing.T) {
	// Standard case: the fetcher owns a clone, so its modifications (the dial
	// guard and the proxy decision hook) cannot leak into other users of
	// http.DefaultTransport.
	cloned := secureBaseTransport()
	require.NotNil(t, cloned, "a base transport is always returned")
	require.NotSame(t, http.DefaultTransport, cloned,
		"the fetcher must own its pool, not share the default one")
	require.NotNil(t, cloned.Proxy, "proxy resolution stays enabled")

	// Fallback case: an embedding application replaced DefaultTransport with a
	// non-standard RoundTripper. The function must return explicit net/http
	// defaults rather than a nil or a type it cannot guard.
	original := http.DefaultTransport
	http.DefaultTransport = nonStandardRoundTripper{}
	t.Cleanup(func() { http.DefaultTransport = original })

	fallback := secureBaseTransport()
	require.NotNil(t, fallback, "a non-standard default must not yield nil")
	require.NotNil(t, fallback.Proxy, "proxy resolution stays enabled in the fallback")
	require.True(t, fallback.ForceAttemptHTTP2, "the fallback keeps HTTP/2")
	require.Positive(t, fallback.MaxIdleConns)
	require.Positive(t, fallback.TLSHandshakeTimeout)
}

// nonStandardRoundTripper stands in for an application that installed its own
// http.RoundTripper, which the fetcher cannot clone as a *http.Transport.
type nonStandardRoundTripper struct{}

func (nonStandardRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("not used by this test")
}
