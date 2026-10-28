package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/pplmx/aurora/internal/logger"
)

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
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("latency", time.Since(start)).
				Msg("request")
		}()
		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
