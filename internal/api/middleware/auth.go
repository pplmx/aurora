package middleware

import (
	"encoding/json"
	"net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key != apiKey {
				writeUnauthorized(w, "invalid api key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}
