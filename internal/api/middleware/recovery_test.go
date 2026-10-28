package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type hijackTrackingWriter struct {
	http.ResponseWriter
	headerWrites int
}

func (w *hijackTrackingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (w *hijackTrackingWriter) WriteHeader(code int) {
	w.headerWrites++
	w.ResponseWriter.WriteHeader(code)
}

// TestRecovery_PanicBeforeWrite_Sends500 covers the happy path:
// handler panics before touching the response, so the recovery
// middleware sends a clean 500 JSON body.
func TestRecovery_PanicBeforeWrite_Sends500(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	body := rr.Body.String()
	assert.Contains(t, body, "internal server error")
	assert.Contains(t, body, "INTERNAL_ERROR")
}

// TestRecovery_PanicAfterWrite_AbortsResponse verifies that recovery neither
// appends a 500 body to an already-written response nor returns normally. A
// normal return would let net/http finish a chunked response as if the handler
// had completed, making a truncated payload look valid to the client.
func TestRecovery_PanicAfterWrite_AbortsResponse(t *testing.T) {
	const expectedBody = "streaming in progress"
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
		panic("oops")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusOK, rr.Code,
		"recovery must not change the status code the handler already set")
	assert.Equal(t, expectedBody, rr.Body.String(),
		"recovery must not append a JSON 500 body to an already-written response")
}

// TestRecovery_PanicAfterFlush_AbortsResponse covers the case where Flush
// commits the response headers before the handler explicitly writes a header
// or body. Recovery must treat that flush as the start of the response.
func TestRecovery_PanicAfterFlush_AbortsResponse(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		panic("oops after flush")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String(),
		"recovery must not append a JSON 500 body after Flush committed the response")
}

func TestRecovery_PanicAfterHijack_AbortsResponse(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := w.(http.Hijacker).Hijack(); err != nil {
			t.Fatalf("Hijack failed: %v", err)
		}
		panic("oops after hijack")
	}))

	rr := httptest.NewRecorder()
	tracked := &hijackTrackingWriter{ResponseWriter: rr}
	req := httptest.NewRequest("GET", "/upgrade", nil)
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(tracked, req)
	})

	assert.Zero(t, tracked.headerWrites,
		"recovery must not write HTTP headers after the connection was hijacked")
}

func TestRecovery_ExplicitAbortDoesNotBecome500(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/abort", nil)
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(rr, req)
	})
	assert.Empty(t, rr.Body.String())
}

func TestRecovery_PartialChunkedResponseDoesNotLookComplete(t *testing.T) {
	const expectedBody = "partial"
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback listeners are unavailable in this sandbox: %v", err)
	}
	server := httptest.NewUnstartedServer(Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, expectedBody)
		w.(http.Flusher).Flush()
		panic("oops after a flushed chunk")
	})))
	server.Listener = ln
	server.Start()
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	assert.Equal(t, expectedBody, string(body))
	assert.Error(t, readErr,
		"a truncated chunked response must end with a transport error, not a clean EOF")
}

// TestRecovery_PreservesFlusher ensures the writeTracker wrapper
// passes Flush through to handlers that stream responses (SSE,
// long-poll, etc.). Without this, any streaming handler wrapped
// in Recovery would silently stop being flushable and clients
// would hang waiting for buffered bytes.
//
// We verify by asserting the wrapped ResponseWriter still
// implements http.Flusher (the contract Recovery promises to
// preserve) and that Flush actually propagates to the underlying
// recorder.
func TestRecovery_PreservesFlusher(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("wrapped ResponseWriter does not implement http.Flusher")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("chunk-1"))
		flusher.Flush()
		_, _ = w.Write([]byte("chunk-2"))
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	handler.ServeHTTP(rr, req)

	if rr.Body.String() != "chunk-1chunk-2" {
		t.Errorf("expected 'chunk-1chunk-2', got %q", rr.Body.String())
	}
}

// TestRecovery_PreservesContentType confirms the wrapper does not
// interfere with headers the handler sets — a regression guard
// for the type assertion / delegation in writeTracker.WriteHeader.
func TestRecovery_PreservesContentType(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "test-value", rr.Header().Get("X-Custom-Header"))
	assert.Equal(t, "ok", rr.Body.String())
}
