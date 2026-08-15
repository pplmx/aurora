package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// loopbackBlocked records whether this environment permits TCP connections
// to a local httptest server. It is computed once in TestMain before any
// test runs.
var loopbackBlocked bool

func TestMain(m *testing.M) {
	loopbackBlocked = !loopbackReachable()
	if loopbackBlocked {
		fmt.Fprintln(os.Stderr,
			"note: loopback TCP is unavailable in this environment; network-dependent fetcher tests will be skipped")
	}
	os.Exit(m.Run())
}

// loopbackReachable reports whether the environment permits dialing a local
// httptest server. Some sandboxes and host firewalls block even 127.0.0.1
// connections; there the fetcher tests cannot exercise a real request path.
func loopbackReachable() bool {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(server.URL)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

// skipIfLoopbackBlocked skips a test that needs a reachable local HTTP
// server when the environment blocks loopback TCP. Where networking works
// (developer machines, CI) every guarded test still runs with all of its
// assertions intact — the skip only converts an environmental red into a
// visible skip, it never weakens a test that can actually run.
func skipIfLoopbackBlocked(t *testing.T) {
	t.Helper()
	if loopbackBlocked {
		t.Skip("loopback TCP is blocked in this environment; the local httptest server is unreachable")
	}
}
