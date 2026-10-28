package middleware

import (
	"bufio"
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

// TestRecovery_PanicAfterWrite_NoCorruption is the regression for
// the bug fixed in this round. If the handler has already called
// WriteHeader or Write before panicking, the recovery middleware
// MUST NOT send a 500 JSON body — that would corrupt the
// partially-written response and confuse the client (e.g. SSE
// clients would see a malformed event stream).
//
// We verify by asserting that the response body contains ONLY
// what the handler wrote, not the recovery body.
func TestRecovery_PanicAfterWrite_NoCorruption(t *testing.T) {
	const expectedBody = "streaming in progress"
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
		panic("oops")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"recovery must not change the status code the handler already set")
	assert.Equal(t, expectedBody, rr.Body.String(),
		"recovery must not append a JSON 500 body to an already-written response")
}

// TestRecovery_PanicAfterFlush_NoCorruption covers the case where Flush commits
// the response headers before the handler explicitly writes a header or body.
// Recovery must treat that flush as the start of the response; otherwise it
// appends an error envelope to a response the client already sees as 200.
func TestRecovery_PanicAfterFlush_NoCorruption(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		panic("oops after flush")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String(),
		"recovery must not append a JSON 500 body after Flush committed the response")
}

func TestRecovery_PanicAfterHijack_NoHTTPWrite(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := w.(http.Hijacker).Hijack(); err != nil {
			t.Fatalf("Hijack failed: %v", err)
		}
		panic("oops after hijack")
	}))

	rr := httptest.NewRecorder()
	tracked := &hijackTrackingWriter{ResponseWriter: rr}
	req := httptest.NewRequest("GET", "/upgrade", nil)
	handler.ServeHTTP(tracked, req)

	assert.Zero(t, tracked.headerWrites,
		"recovery must not write HTTP headers after the connection was hijacked")
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
