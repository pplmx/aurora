package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

// The fetcher's budget had tests for how it is configured and for the SSRF
// guard, but none proved that a slow or hanging upstream is actually cut off.
// An oracle source that never answers would otherwise pin a scheduler slot and
// its goroutine for as long as the upstream stayed silent.

// slowServer answers only once its request context ends, and reports whether
// that happened, so a test can tell "the client gave up" from "the handler
// returned on its own".
func slowServer(t *testing.T) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	released := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			close(released)
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	return srv, released
}

// TestFetchDataContext_HonoursTheClientTimeout proves the configured timeout
// bounds a request whose upstream never answers.
func TestFetchData_HonoursTheClientTimeout(t *testing.T) {
	srv, released := slowServer(t)
	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, 150*time.Millisecond)
	require.NoError(t, err)

	start := time.Now()
	_, err = fetcher.FetchData(&oracle.DataSource{ID: "slow", URL: srv.URL})
	elapsed := time.Since(start)

	require.Error(t, err, "a hanging upstream must not return data")
	assert.Less(t, elapsed, 5*time.Second,
		"the request must be cut off near the configured timeout, took %v", elapsed)

	// The server sees its context end, which is what releases the handler and
	// its goroutine rather than leaving it parked.
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("the upstream handler was never released")
	}
}

// TestFetchDataContext_CancellationInterruptsAnInFlightRequest proves a
// cancelled caller stops the fetch promptly. The API server's shutdown path
// relies on this: srv.Close() tears the database pool down underneath any
// request still running, so a fetch that ignores cancellation would finish
// against a closed pool.
func TestFetchDataContext_CancellationInterruptsAnInFlightRequest(t *testing.T) {
	srv, released := slowServer(t)
	// A generous timeout: only the caller's cancellation can end this request.
	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, time.Minute)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := fetcher.FetchDataContext(ctx, &oracle.DataSource{ID: "slow", URL: srv.URL})
		done <- err
	}()

	// Give the request time to reach the server, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err, "a cancelled fetch must not return data")
		assert.True(t,
			errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
			"the cancellation must be in the error chain, got %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled fetch must return instead of waiting for the upstream")
	}

	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the caller must release the upstream handler")
	}
}

// TestFetchDataContext_AlreadyCancelledMakesNoRequest proves a cancelled
// caller does not open a connection at all — a shutdown that cancels the
// scheduler must not start new work on its way out.
func TestFetchDataContext_AlreadyCancelledMakesNoRequest(t *testing.T) {
	var reached bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	t.Cleanup(srv.Close)

	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, time.Minute)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = fetcher.FetchDataContext(ctx, &oracle.DataSource{ID: "cancelled", URL: srv.URL})
	require.Error(t, err)
	assert.False(t, reached, "a cancelled caller must not reach the upstream")
}

// TestFetchData_TimesOutOnASlowBody covers the half a client timeout alone does
// not: headers arrive at once, then the body trickles. The response deadline
// covers the whole exchange, so this must still fail rather than hang.
func TestFetchData_TimesOutOnASlowBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Never finish the body.
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, 200*time.Millisecond)
	require.NoError(t, err)

	start := time.Now()
	_, err = fetcher.FetchData(&oracle.DataSource{ID: "trickle", URL: srv.URL})
	require.Error(t, err, "a body that never completes must not be accepted")
	assert.Less(t, time.Since(start), 5*time.Second)
}

// TestFetchDataContext_ConnectsToAnUnreachableHost proves a connection that
// never establishes is bounded too: a source pointing at a blackholed address
// must fail rather than hold the scheduler.
func TestFetchDataContext_ConnectsToAnUnreachableHost(t *testing.T) {
	// A listener that accepts the TCP connection and never speaks HTTP is the
	// closest local stand-in for a blackhole.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Hold the connection open without answering.
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	fetcher, err := newLoopbackFetcherWithTimeout(10, time.Minute, 200*time.Millisecond)
	require.NoError(t, err)

	start := time.Now()
	_, err = fetcher.FetchData(&oracle.DataSource{
		ID: "silent", URL: "http://" + ln.Addr().String() + "/price",
	})
	require.Error(t, err, "a silent peer must not be waited on forever")
	assert.Less(t, time.Since(start), 5*time.Second)
}
