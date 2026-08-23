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

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)
	r.Use(apimw.CORS)

	// Request observability: a stdlib Prometheus-text /metrics endpoint
	// (v1.13) recording per-request status + latency distribution. The
	// registry is owned by the Server (see Server.MetricsRegistry /
	// Server.MetricsHandler) so the same live counters are also exportable
	// on an external metrics mount (v1.14).
	reg := s.MetricsRegistry()
	r.Use(metrics.Middleware(reg))

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
