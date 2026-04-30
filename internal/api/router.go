package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apimw "github.com/pplmx/aurora/internal/api/middleware"
	"github.com/pplmx/aurora/internal/config"
)

func newRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)
	r.Use(apimw.CORS)

	r.Get("/healthz", LivenessHandler)
	r.Get("/readyz", ReadinessHandler(s.db))
	r.Get("/health", LivenessHandler)

	apiKey := config.GetAPIKey()

	r.Group(func(api chi.Router) {
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
	})

	r.Handle("/*", http.FileServer(http.Dir("web")))

	return r
}
