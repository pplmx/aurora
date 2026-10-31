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
		{"deprecated IPv6 site-local", "http://[fec0::1]/x", true},
		{"IPv4-compatible loopback", "http://[::127.0.0.1]/x", true},
		{"IPv6 protocol assignment", "http://[2001:10::1]/x", true},
		{"IPv4 multicast", "http://224.0.0.1/x", true},
		{"documentation IP", "http://203.0.113.1/x", true},
		{"6to4 loopback", "http://[2002:7f00:1::]/x", true},
		{"localhost hostname (DNS)", "http://localhost:9/secret", true},
		{"public IP allowed", "http://8.8.8.8/x", false},
		{"public IPv6 allowed", "http://[2606:4700:4700::1111]/x", false},
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

	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("redirect loop error = %v, want ErrTooManyRedirects", err)
	}
	if transport.calls != maxRedirects {
		t.Fatalf("redirect loop made %d requests, want %d", transport.calls, maxRedirects)
	}
}

type redirectedRequest struct {
	header http.Header
	host   string
}

type headerRedirectTransport struct {
	location string
	requests []redirectedRequest
}

func (t *headerRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, redirectedRequest{
		header: req.Header.Clone(),
		host:   req.Host,
	})
	status := http.StatusOK
	header := make(http.Header)
	if req.URL.Path == "/start" {
		status = http.StatusFound
		header.Set("Location", t.location)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestCheckRedirectAllowed_ScopesCustomHeadersToOrigin(t *testing.T) {
	cases := []struct {
		name       string
		location   string
		wantAPIKey string
		wantHost   string
	}{
		{name: "same origin keeps headers", location: "http://8.8.8.8/next", wantAPIKey: "secret", wantHost: "feed.vendor.example"},
		{name: "cross origin drops headers", location: "http://1.1.1.1/next"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &headerRedirectTransport{location: tc.location}
			client := &http.Client{
				Transport:     transport,
				CheckRedirect: checkRedirectAllowed,
			}
			req, err := http.NewRequest(http.MethodGet, "http://8.8.8.8/start", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-API-Key", "secret")
			req.Host = "feed.vendor.example"

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("redirect request: %v", err)
			}
			_ = resp.Body.Close()

			if len(transport.requests) != 2 {
				t.Fatalf("made %d requests, want 2", len(transport.requests))
			}
			if got := transport.requests[1].header.Get("X-API-Key"); got != tc.wantAPIKey {
				t.Fatalf("redirected X-API-Key = %q, want %q", got, tc.wantAPIKey)
			}
			if got := transport.requests[1].host; got != tc.wantHost {
				t.Fatalf("redirected Host = %q, want %q", got, tc.wantHost)
			}
		})
	}
}

func TestSameURLOrigin(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "same host and scheme", a: "https://example.com/a", b: "https://example.com/b", want: true},
		{name: "implicit default port", a: "https://example.com/a", b: "https://example.com:443/b", want: true},
		{name: "different scheme", a: "https://example.com/a", b: "http://example.com/b", want: false},
		{name: "different host", a: "https://example.com/a", b: "https://sub.example.com/b", want: false},
		{name: "different port", a: "https://example.com/a", b: "https://example.com:8443/b", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := url.Parse(tc.a)
			b, _ := url.Parse(tc.b)
			if got := sameURLOrigin(a, b); got != tc.want {
				t.Fatalf("sameURLOrigin(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
