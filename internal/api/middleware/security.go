package middleware

import "net/http"

// SecurityHeaders sets conservative security headers on every response
// (v1.48). Rails-adjacent hardening for the served API and web UI:
//   - X-Content-Type-Options: nosniff      prevent MIME-type sniffing
//   - X-Frame-Options: DENY                prevent clickjacking of the web UI
//   - Referrer-Policy: no-referrer         prevent referrer leakage
//
// These are safe defaults for the single-origin web UI: it does not use frames
// and all authenticating REST calls send the X-API-Key header rather than
// relying on the Referer.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
