package middleware

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/pplmx/aurora/internal/logger"
)

// writeTracker wraps http.ResponseWriter to detect whether the
// handler already started writing the response. When a panic
// happens AFTER WriteHeader or Write has been called, the response
// is already on the wire (at least partially) — appending a 500
// JSON body to it would corrupt the protocol and the client would
// see a malformed message.
//
// Recovery uses this to decide whether it can safely write a 500
// response. If the handler already wrote, it aborts the connection
// instead of making a truncated response look complete.
type writeTracker struct {
	http.ResponseWriter
	wroteHeader bool
	wroteBody   bool
	hijacked    bool
}

func (w *writeTracker) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *writeTracker) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	w.wroteBody = true
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher so handlers that stream responses
// (e.g. SSE) keep working through the wrapper.
func (w *writeTracker) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		// Flush commits the response headers even when the handler has not
		// called WriteHeader or Write explicitly.
		w.wroteHeader = true
		f.Flush()
	}
}

// Hijack supports websocket-style upgrades through the wrapper.
func (w *writeTracker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		conn, rw, err := h.Hijack()
		if err == nil {
			w.hijacked = true
		}
		return conn, rw, err
	}
	return nil, nil, errors.New("hijacker not supported")
}

// alreadyWritten reports whether the wrapped handler has sent
// anything to the client.
func (w *writeTracker) alreadyWritten() bool {
	return w.wroteHeader || w.wroteBody || w.hijacked
}

func Recovery(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tracker := &writeTracker{ResponseWriter: w}
		defer func() {
			err := recover()
			if err == nil {
				return
			}
			// ErrAbortHandler is net/http's explicit contract for aborting a
			// response without logging a stack trace. Preserve it instead of
			// turning it into a 500 or swallowing it.
			// net/http documents this exact comparison for ErrAbortHandler
			// (httputil does the same); the value is never wrapped.
			if err == http.ErrAbortHandler { //nolint:errorlint // see above
				panic(http.ErrAbortHandler)
			}

			logger.Error().Interface("error", err).Msg("panic recovered")
			if tracker.alreadyWritten() {
				// Returning normally here would let net/http finish a chunked
				// response as if the handler had succeeded. Abort the
				// connection so the client observes an incomplete response.
				logger.Warn().Msg("handler had already written; aborting response to avoid corrupting it")
				panic(http.ErrAbortHandler)
			}
			tracker.Header().Set("Content-Type", "application/json")
			tracker.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(tracker).Encode(map[string]string{
				// lowercase to match handler-generated 500s (helpers.go)
				// so clients see one consistent message shape.
				"error": "internal server error",
				"code":  "INTERNAL_ERROR",
			})
		}()
		next.ServeHTTP(tracker, r)
	}
	return http.HandlerFunc(fn)
}
