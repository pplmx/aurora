package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apimw "github.com/pplmx/aurora/internal/api/middleware"
)

func newRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1/lottery", func(r chi.Router) {
		s.lotteryHandler.Routes(r)
	})

	r.Route("/api/v1/voting", func(r chi.Router) {
		s.votingHandler.Routes(r)
	})

	r.Route("/api/v1/nft", func(r chi.Router) {
		s.nftHandler.Routes(r)
	})

	r.Route("/api/v1/token", func(r chi.Router) {
		s.tokenHandler.Routes(r)
	})

	r.Route("/api/v1/oracle", func(r chi.Router) {
		s.oracleHandler.Routes(r)
	})

	return r
}
