package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsBlockedHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.1:8080", true},
		{"[::1]:8080", true},
		{"169.254.169.254", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"100.64.0.1", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"93.184.216.34", false},
		{"example.com", false}, // hostname: not an IP literal, out of scope here
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isBlockedHost(tt.host); got != tt.want {
				t.Errorf("isBlockedHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

// TestFetcher_RedirectToPrivateIPBlocked proves a redirection that would land
// the fetcher on a loopback/private address is refused — the SSRF primitive
// where a hostile source bounces us at 127.0.0.1 or cloud metadata.
func TestFetcher_RedirectToPrivateIPBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:9/secret", http.StatusFound)
	}))
	defer server.Close()

	fetcher := NewFetcherWithConfig(100, time.Minute)
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error when a redirect targets a private address")
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("expected ErrBlockedDestination, got %v", err)
	}
}

// TestFetcher_RedirectToNonHTTPSchemeBlocked proves a redirect that would
// switch the fetcher to a non-HTTP(S) scheme is refused (e.g. file://).
func TestFetcher_RedirectToNonHTTPSchemeBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	}))
	defer server.Close()

	fetcher := NewFetcherWithConfig(100, time.Minute)
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error when a redirect switches to a non-HTTP scheme")
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("expected ErrBlockedDestination, got %v", err)
	}
}
