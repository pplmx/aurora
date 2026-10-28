package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

// TestFetcherCloseReleasesIdleConnections proves a completed oracle fetch does
// not leave its keep-alive socket behind after the owning process is closing.
// The security wrapper is part of the transport stack, so it must forward the
// standard CloseIdleConnections hook to the real net/http transport.
func TestFetcherCloseReleasesIdleConnections(t *testing.T) {
	skipIfLoopbackBlocked(t)

	states := make(chan http.ConnState, 16)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"price":42}`))
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		select {
		case states <- state:
		default:
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = fetcher.Close() })

	_, err = fetcher.FetchData(&oracle.DataSource{
		ID:   "lifecycle",
		URL:  srv.URL,
		Path: "price",
	})
	require.NoError(t, err)
	waitForConnState(t, states, http.StateIdle)

	require.NoError(t, fetcher.Close())
	waitForConnState(t, states, http.StateClosed)
}

func waitForConnState(t *testing.T, states <-chan http.ConnState, want http.ConnState) {
	t.Helper()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case state := <-states:
			if state == want {
				return
			}
		case <-timer.C:
			t.Fatalf("connection never reached state %s", want)
		}
	}
}
