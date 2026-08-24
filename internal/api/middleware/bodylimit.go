package middleware

import (
	"net/http"
)

// MaxRequestBody caps the size of any JSON request body the API handlers may
// read. With rate limiting off by default (api.rateLimit.enabled=false), a
// client could otherwise stream an arbitrarily large body — e.g. a fake
// lottery `participants` array — and the json.Decoder would materialize every
// element, letting a few connections exhaust server memory (v1.71, ISS-077).
//
// The cap must comfortably exceed the largest legitimate payload: the lottery
// domain allows up to 10k participants (~1MB as JSON), so 4 MiB leaves ample
// headroom for that and any voting/NFT/oracle payload while bounding the
// memory a single request can force into the process.
const MaxRequestBody = 4 << 20 // 4 MiB

// BodyLimit wraps r.Body in http.MaxBytesReader so the first handler that
// reads it is stopped once the cap is exceeded (json decode then returns
// *http.MaxBytesError, surfaced as HTTP 413 by handler.decodeJSON). Registered
// outer to APIKeyAuth so even unauthenticated bodies are capped before auth
// runs, and outer to the rate limiter so the cap applies on both sides of it.
func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}
