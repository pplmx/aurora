package middleware

import "net/http"

// SecurityHeaders sets conservative security headers on every response
// (v1.48). Rails-adjacent hardening for the served API:
//   - X-Content-Type-Options: nosniff      prevent MIME-type sniffing
//   - X-Frame-Options: DENY                prevent clickjacking of the API's responses
//   - Referrer-Policy: no-referrer         prevent referrer leakage
//
// These are safe defaults for a single-origin API client: it does not use frames
// and all authenticating REST calls send the X-API-Key header rather than
// relying on the Referer.
//
// Content-Security-Policy is deliberately NOT set here: it must carry a
// per-response nonce for the API-key bootstrap script the webauth injector
// writes into served HTML, so it lives in api.injectAPIKey
// (internal/api/webauth.go, round-151 ISS-268).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
