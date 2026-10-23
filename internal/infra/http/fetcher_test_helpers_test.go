package http

import (
	"net/http"
	"time"
)

// newLoopbackFetcher keeps the production redirect policy and header wrapper
// while removing only the dial-time private-address guard. Tests that exercise
// request handling against httptest servers need this explicit opt-out; the
// guard itself is covered by fetcher_dial_guard_test.go.
func newLoopbackFetcher(opts ...FetcherOption) *Fetcher {
	return allowTestLoopback(NewFetcher(opts...))
}

func newLoopbackFetcherWithConfig(limit int, window time.Duration) *Fetcher {
	return allowTestLoopback(NewFetcherWithConfig(limit, window))
}

func newLoopbackFetcherWithTimeout(limit int, window, timeout time.Duration) (*Fetcher, error) {
	fetcher, err := NewFetcherWithTimeout(limit, window, timeout)
	if err != nil {
		return nil, err
	}
	return allowTestLoopback(fetcher), nil
}

func allowTestLoopback(fetcher *Fetcher) *Fetcher {
	if security, ok := fetcher.client.Transport.(*securityTransport); ok {
		security.transport = http.DefaultTransport
		security.dialer = nil
	}
	return fetcher
}
