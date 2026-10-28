package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const (
	// RequestIDHeader is both the inbound correlation header and the response
	// header echoed back to the caller.
	RequestIDHeader = "X-Request-ID"

	// maxRequestIDLength bounds caller-supplied IDs before they can reach the
	// access log. net/http permits headers up to roughly 1 MiB by default, and
	// Logger runs before authentication, so trusting the raw value would let an
	// unauthenticated client amplify each request into a near-1 MiB log line.
	maxRequestIDLength = 128
)

// RequestID accepts a bounded, visible-ASCII caller correlation ID or generates
// a fresh random one. The effective value is stored under chi's request-ID key,
// mirrored back into the request for downstream consumers, and returned in the
// response so clients can correlate errors with logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		r.Header.Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), chimiddleware.RequestIDKey, id)
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < 0x21 || id[i] > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	// Request IDs are correlation values, not secrets. A bounded fallback is
	// preferable to dropping the middleware or returning an empty ID if the
	// platform random source is temporarily unavailable.
	return fmt.Sprintf("req-%d", chimiddleware.NextRequestID())
}
