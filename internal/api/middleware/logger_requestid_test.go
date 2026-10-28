package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/logger"
	"github.com/rs/zerolog"
)

// TestLogger_IncludesRequestID proves the Logger middleware emits the request
// ID that chi's RequestID middleware stored in the context, so log lines can
// be correlated to a single request (v1.14: wire request IDs into request logs).
func TestLogger_IncludesRequestID(t *testing.T) {
	old := logger.Log
	defer func() { logger.Log = old }()

	var buf bytes.Buffer
	logger.Log = zerolog.New(&buf).With().Timestamp().Logger()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	reqID := "req-abc-123"
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", reqID)

	// Chain chi's RequestID (as the router does) so the ID lands in ctx before
	// Logger runs.
	chain := RequestID(Logger(handler))
	chain.ServeHTTP(httptest.NewRecorder(), req)

	logs := buf.String()
	if !strings.Contains(logs, `"request_id":"`+reqID+`"`) {
		t.Errorf("log did not include structured request_id=%q:\n%s", reqID, logs)
	}
	if !strings.Contains(logs, `"method":"GET"`) {
		t.Errorf("expected method field:\n%s", logs)
	}
}

func TestLogger_LogsAbortedRequest(t *testing.T) {
	old := logger.Log
	defer func() { logger.Log = old }()

	var buf bytes.Buffer
	logger.Log = zerolog.New(&buf).With().Timestamp().Logger()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
		panic(http.ErrAbortHandler)
	})

	reqID := "req-aborted"
	req := httptest.NewRequest(http.MethodGet, "/aborted", nil)
	req.Header.Set("X-Request-ID", reqID)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		RequestID(Logger(handler)).ServeHTTP(httptest.NewRecorder(), req)
	}()
	if recovered != http.ErrAbortHandler {
		t.Fatalf("recovered panic = %v, want http.ErrAbortHandler", recovered)
	}

	logs := buf.String()
	if !strings.Contains(logs, `"request_id":"`+reqID+`"`) {
		t.Errorf("aborted request log did not include request_id=%q:\n%s", reqID, logs)
	}
	if !strings.Contains(logs, `"path":"/aborted"`) {
		t.Errorf("aborted request log did not include the request path:\n%s", logs)
	}
	if !strings.Contains(logs, `"status":206`) {
		t.Errorf("aborted request log did not include the committed status:\n%s", logs)
	}
}

func TestLogger_DoesNotAmplifyOversizedClientRequestID(t *testing.T) {
	old := logger.Log
	defer func() { logger.Log = old }()

	var buf bytes.Buffer
	logger.Log = zerolog.New(&buf).With().Timestamp().Logger()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, strings.Repeat("x", 64<<10))

	RequestID(Logger(handler)).ServeHTTP(httptest.NewRecorder(), req)

	logs := buf.String()
	if strings.Contains(logs, strings.Repeat("x", 1024)) {
		t.Fatal("the access log echoed an oversized client request ID")
	}
	if !strings.Contains(logs, `"request_id":"`) {
		t.Fatalf("the access log did not use a sanitized request ID:\n%s", logs)
	}
	if len(logs) > 2048 {
		t.Fatalf("access log grew to %d bytes from one request", len(logs))
	}
}

func TestLogger_DoesNotAmplifyOversizedPath(t *testing.T) {
	old := logger.Log
	defer func() { logger.Log = old }()

	var buf bytes.Buffer
	logger.Log = zerolog.New(&buf).With().Timestamp().Logger()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.Path = "/" + strings.Repeat("p", 64<<10)

	RequestID(Logger(handler)).ServeHTTP(httptest.NewRecorder(), req)

	logs := buf.String()
	if strings.Contains(logs, strings.Repeat("p", 2048)) {
		t.Fatal("the access log echoed an oversized URL path")
	}
	if !strings.Contains(logs, `"path":"`) {
		t.Fatalf("the access log did not include a bounded path:\n%s", logs)
	}
	if len(logs) > 2048 {
		t.Fatalf("access log grew to %d bytes from one request", len(logs))
	}
}
