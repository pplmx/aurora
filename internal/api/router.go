package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pplmx/aurora/internal/api/handler"
	apimw "github.com/pplmx/aurora/internal/api/middleware"
)

func NewRouter() http.Handler {
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
		h := handler.NewLotteryHandler()
		h.Routes(r)
	})

	r.Route("/api/v1/voting", func(r chi.Router) {
		h := handler.NewVotingHandler()
		h.Routes(r)
	})

	r.Route("/api/v1/nft", func(r chi.Router) {
		h := handler.NewNFTHandler()
		h.Routes(r)
	})

	return r
}
