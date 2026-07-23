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
// response. If the handler already wrote, we just log and let the
// connection end.
type writeTracker struct {
	http.ResponseWriter
	wroteHeader bool
	wroteBody   bool
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
		f.Flush()
	}
}

// Hijack supports websocket-style upgrades through the wrapper.
func (w *writeTracker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijacker not supported")
}

// alreadyWritten reports whether the wrapped handler has sent
// anything to the client.
func (w *writeTracker) alreadyWritten() bool {
	return w.wroteHeader || w.wroteBody
}

func Recovery(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tracker := &writeTracker{ResponseWriter: w}
		defer func() {
			if err := recover(); err != nil {
				logger.Error().Interface("error", err).Msg("panic recovered")
				if tracker.alreadyWritten() {
					// The handler already sent headers or a body
					// before panicking. We can't send a clean 500
					// response now — the response is on the wire
					// (potentially partially). Just let the
					// connection close; the client will see an
					// incomplete response and can retry.
					logger.Warn().Msg("handler had already written; skipping recovery body to avoid corrupting response")
					return
				}
				tracker.Header().Set("Content-Type", "application/json")
				tracker.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(tracker).Encode(map[string]string{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()
		next.ServeHTTP(tracker, r)
	}
	return http.HandlerFunc(fn)
}
