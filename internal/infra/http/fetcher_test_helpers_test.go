package http

import "time"

// newLoopbackFetcher keeps the production redirect policy and header wrapper
// while removing only the dial-time private-address guard. Tests that exercise
// request handling against httptest servers need this explicit opt-out; the
// guard itself is covered by fetcher_dial_guard_test.go.
func newLoopbackFetcher(opts ...FetcherOption) *Fetcher {
	return allowTestLoopback(NewFetcher(opts...))
}

func newLoopbackFetcherWithConfig(limit int, window time.Duration) *Fetcher {
	return allowTestLoopback(NewFetcher(WithRateLimit(limit, window)))
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
		// Keep the production clone (and therefore its isolated connection
		// pool), but remove only the destination guard for local test servers.
		// Replacing the wrapped transport with http.DefaultTransport made
		// every test fetcher share global sockets and prevented lifecycle tests
		// from proving that this fetcher owned and released its own pool.
		if security.dialer != nil {
			security.dialer.ControlContext = nil
		}
	}
	return fetcher
}
