package middleware

import (
	"net/http"
)

// CORS restricts cross-origin reads of API and Web UI responses to an explicit
// allow-list of origins. It NEVER emits a wildcard "Access-Control-Allow-
// Origin": the gateway embeds the API key into every served HTML page (see
// api.injectAPIKey), so a wildcard would let any page the operator visits read
// window.AURORA_API_KEY and then drive every /api/v1 mutation (mint/transfer/
// burn, lottery/voting state, oracle sources) with the operator's authority.
//
// The Web UI is served by the gateway itself, so same-origin requests — which
// browsers can read without any CORS header — continue to work with an empty
// allow-list (the secure default). Operators who intentionally serve the UI or
// API from another origin add that origin to api.cors.allowedOrigins.
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

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

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
