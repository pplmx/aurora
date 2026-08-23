package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
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
	chain := middleware.RequestID(Logger(handler))
	chain.ServeHTTP(httptest.NewRecorder(), req)

	logs := buf.String()
	if !strings.Contains(logs, `"request_id":"`+reqID+`"`) {
		t.Errorf("log did not include structured request_id=%q:\n%s", reqID, logs)
	}
	if !strings.Contains(logs, `"method":"GET"`) {
		t.Errorf("expected method field:\n%s", logs)
	}
}
