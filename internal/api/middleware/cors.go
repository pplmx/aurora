package middleware

import (
	"net/http"
)

const (
	corsAllowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders = "Content-Type, X-API-Key, X-Request-ID"
)

// CORS restricts cross-origin browser reads of API responses to an explicit
// allow-list of origins. It NEVER emits a wildcard Access-Control-Allow-Origin:
// every listed origin is echoed exactly, so an unconfigured deployment grants
// no browser-readable cross-origin response by default.
//
// Same-origin requests can read responses without CORS headers, so the empty
// default does not affect a browser app served from the API's own origin.
// Operators who intentionally host a browser client elsewhere add that exact
// origin to api.cors.allowedOrigins.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				// Browsers send Origin on every cross-origin request; agents
				// may cache the response only for the same origin, and a
				// disallowed origin must never be answered with ACAO at all.
				w.Header().Add("Vary", "Origin")
				if originAllowed(origin, allowedOrigins) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed reports whether origin appears in the configured allow-list.
// It is a small explicit loop (no `slices` dependency) over what is expected
// to be a short operator-supplied list.
func originAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if origin == o {
			return true
		}
	}
	return false
}
