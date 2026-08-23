package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apimw "github.com/pplmx/aurora/internal/api/middleware"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/metrics"
)

func newRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	// Request observability: a stdlib Prometheus-text /metrics endpoint
	// (v1.13) recording per-request status + latency distribution. The
	// registry is owned by the Server (see Server.MetricsRegistry /
	// Server.MetricsHandler) so the same live counters are also exportable
	// on an external metrics mount (v1.14).
	//
	// Ordering (v1.59): this middleware MUST be registered OUTER to Recovery.
	// In chi the first-registered middleware is the outermost wrapper, and an
	// outer middleware observes the whole inner chain INCLUDING the recovered
	// response. If metrics were inner to Recovery, a panicking handler would
	// unwind past it before Recovery caught the panic, so the 500 Recovery
	// sends would never be counted — server faults would be invisible in the
	// request metrics/histograms. Placing metrics first records every request
	// (status + latency), panics included, as status 500.
	reg := s.MetricsRegistry()
	r.Use(metrics.Middleware(reg))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)
	r.Use(apimw.CORS(config.AllowedCORSOrigins()))
	r.Use(apimw.SecurityHeaders)

	r.Get("/healthz", LivenessHandler)
	r.Get("/readyz", ReadinessHandler(s.db))
	r.Get("/health", LivenessHandler)
	r.Handle("/metrics", reg.Handler())
	r.Handle("/metrics/oracle", s.OracleMetricsHandler())

	apiKey := config.GetAPIKey()

	r.Group(func(api chi.Router) {
		// Optional per-client rate limiting on the protected API (v1.19),
		// applied before auth so even unauthenticated abuse is bounded.
		// Disabled by default; enable via api.rateLimit.enabled.
		if config.RateLimitEnabled() {
			lim := apimw.NewFixedWindowLimiter(config.RateLimitRequests(), config.RateLimitWindow(), nil)
			api.Use(apimw.RateLimit(lim))
		}
		api.Use(apimw.APIKeyAuth(apiKey))

		api.Route("/api/v1/lottery", func(r chi.Router) {
			s.lotteryHandler.Routes(r)
		})

		api.Route("/api/v1/voting", func(r chi.Router) {
			s.votingHandler.Routes(r)
		})

		api.Route("/api/v1/nft", func(r chi.Router) {
			s.nftHandler.Routes(r)
		})

		api.Route("/api/v1/token", func(r chi.Router) {
			s.tokenHandler.Routes(r)
		})

		api.Route("/api/v1/oracle", func(r chi.Router) {
			s.oracleHandler.Routes(r)
		})

		api.Route("/api/v1/blockchain", func(r chi.Router) {
			s.blockchainHandler.Routes(r)
		})
	})

	r.Handle("/*", injectAPIKey(http.FileServer(http.Dir("web")), config.GetAPIKey()))

	return r
}
