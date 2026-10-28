package http

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// redirectReq builds a minimal *http.Request whose URL is rawurl. checkRedirectAllowed
// only reads req.URL.Scheme and req.URL.Host, so no live server is needed —
// these tests run even where loopback listeners are blocked (which is why the
// integration-served redirect tests skip in sandboxes, leaving this policy at
// 0% coverage).
func redirectReq(rawurl string) *http.Request {
	u, _ := url.Parse(rawurl)
	return &http.Request{URL: u}
}

// TestCheckRedirectAllowed directly exercises the SSRF redirect policy. The
// policy must refuse non-HTTP(S) schemes, IP literals in blocked ranges, and
// hostnames that resolve into blocked space (DNS), while allowing public
// destinations.
func TestCheckRedirectAllowed(t *testing.T) {
	tests := []struct {
		name    string
		rawurl  string
		wantErr bool
	}{
		{"non-http scheme", "file:///etc/passwd", true},
		{"non-http scheme ftp", "ftp://example.com/x", true},
		{"loopback IP literal", "http://127.0.0.1:9/secret", true},
		{"RFC1918 private IP", "http://10.0.0.5/x", true},
		{"cloud metadata IP", "http://169.254.169.254/latest", true},
		{"link-local IPv6", "http://[fe80::1]/x", true},
		{"localhost hostname (DNS)", "http://localhost:9/secret", true},
		{"public IP allowed", "http://8.8.8.8/x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRedirectAllowed(redirectReq(tt.rawurl), nil)
			if tt.wantErr {
				if !errors.Is(err, ErrBlockedDestination) {
					t.Fatalf("want ErrBlockedDestination, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("want nil error for allowed redirect, got %v", err)
			}
		})
	}
}

// TestCheckRedirectAllowed_CapsChain ensures the custom redirect policy also
// enforces the standard ten-hop limit. net/http applies its default cap only
// when CheckRedirect is nil, so installing a custom policy removed the bound
// unless the callback enforces it itself.
func TestCheckRedirectAllowed_CapsChain(t *testing.T) {
	req := redirectReq("http://8.8.8.8/x")
	if err := checkRedirectAllowed(req, make([]*http.Request, maxRedirects)); !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("redirect policy at ten hops = %v, want ErrTooManyRedirects", err)
	}
	if err := checkRedirectAllowed(req, make([]*http.Request, maxRedirects-1)); err != nil {
		t.Fatalf("redirect policy below the hop cap = %v, want nil", err)
	}
}

type redirectLoopTransport struct {
	calls int
}

func (t *redirectLoopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	return &http.Response{
		StatusCode: http.StatusFound,
		Header:     http.Header{"Location": []string{req.URL.String()}},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestCheckRedirectAllowed_StopsClientLoop(t *testing.T) {
	transport := &redirectLoopTransport{}
	client := &http.Client{
		Transport:     transport,
		CheckRedirect: checkRedirectAllowed,
	}
	req, err := http.NewRequest(http.MethodGet, "http://8.8.8.8/loop", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Do(req); !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("redirect loop error = %v, want ErrTooManyRedirects", err)
	}
	if transport.calls != maxRedirects {
		t.Fatalf("redirect loop made %d requests, want %d", transport.calls, maxRedirects)
	}
}
