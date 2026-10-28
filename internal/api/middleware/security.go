package middleware

import "net/http"

// SecurityHeaders sets conservative security headers on every response:
//   - X-Content-Type-Options: nosniff  prevent MIME-type sniffing
//   - X-Frame-Options: DENY            prevent clickjacking of API responses
//   - Referrer-Policy: no-referrer     prevent referrer leakage
//   - Cache-Control: no-store          keep authenticated data out of caches
//
// Authentication uses X-API-Key rather than Authorization, so HTTP cache
// implementations do not apply their standard protection for authenticated
// requests. Responses can contain one-time private keys and oracle request
// headers, making shared or browser caching unsafe even for successful GETs.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
