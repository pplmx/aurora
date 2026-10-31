package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestFetcher_RedirectToPrivateIPBlocked proves a redirection that would land
// the fetcher on a loopback/private address is refused — the SSRF primitive
// where a hostile source bounces us at 127.0.0.1 or cloud metadata.
func TestFetcher_RedirectToPrivateIPBlocked(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:9/secret", http.StatusFound)
	}))
	defer server.Close()

	fetcher := newLoopbackFetcherWithConfig(100, time.Minute)
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	}))
	defer server.Close()

	fetcher := newLoopbackFetcherWithConfig(100, time.Minute)
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error when a redirect switches to a non-HTTP scheme")
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("expected ErrBlockedDestination, got %v", err)
	}
}

// TestFetcher_RedirectToLocalhostBlocked proves a hostname redirect to
// localhost (which resolves into loopback space) is refused — the bypass that
// a literal-IP-only check would have left open.
func TestFetcher_RedirectToLocalhostBlocked(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://localhost:9/internal", http.StatusFound)
	}))
	defer server.Close()

	fetcher := newLoopbackFetcherWithConfig(100, time.Minute)
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error when a redirect targets localhost")
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("expected ErrBlockedDestination, got %v", err)
	}
}

// TestFetcher_RedirectToUnresolvableHostBlocked proves a redirect to a host
// that cannot be resolved is refused rather than followed into the unknown.
func TestFetcher_RedirectToUnresolvableHostBlocked(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://does-not-exist-aurora.invalid/x", http.StatusFound)
	}))
	defer server.Close()

	fetcher := newLoopbackFetcherWithConfig(100, time.Minute)
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error when a redirect target cannot be resolved")
	}
	if !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("expected ErrBlockedDestination, got %v", err)
	}
}
