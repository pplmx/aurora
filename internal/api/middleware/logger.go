package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/pplmx/aurora/internal/logger"
)

// maxLoggedPathLength bounds the request path copied into the access log.
// Logger runs before authentication, while net/http accepts request lines
// within its roughly 1 MiB header limit; logging the full path would let one
// unauthenticated request amplify into a near-1 MiB log entry.
const maxLoggedPathLength = 1024

func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		defer func() {
			// Correlate this log line with the request ID that chi's RequestID
			// middleware (registered before Logger) stored in the context.
			// GetReqID returns "" when no middleware populated it (e.g. the
			// middleware used standalone in tests), which zerolog drops
			// silently. Deferred logging preserves the record when Recovery
			// aborts a partially written response.
			logger.Info().
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("method", r.Method).
				Str("path", boundedLogValue(r.URL.Path, maxLoggedPathLength)).
				Int("status", ww.Status()).
				Dur("latency", time.Since(start)).
				Msg("request")
		}()
		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

func boundedLogValue(value string, max int) string {
	if max < 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
