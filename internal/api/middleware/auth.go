// Package middleware holds the HTTP wrappers every route passes through: the
// API key check, the request body limit, the client-address policy, CORS, the
// access log, the rate limiter, panic recovery, request IDs and the security
// headers.
//
// Order matters and lives in one place, the router, because a limit applied
// after authentication protects nothing.
package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if !secureCompare(key, apiKey) {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func secureCompare(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	// Flat {"error","code"} envelope, matching every other error surface
	// (handler helpers, rate limiter, recovery); the previous nested
	// {"error":{"code","message"}} required clients to special-case 401
	// (TASK-114).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "authentication required",
		"code":  "UNAUTHORIZED",
	})
}
