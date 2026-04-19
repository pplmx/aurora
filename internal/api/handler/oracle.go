package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

// defaultQueryLimit is the default limit for oracle data query API responses.
const defaultQueryLimit = 10

type OracleHandler struct {
	repo oracle.Repository
}

func NewOracleHandler(repo oracle.Repository) *OracleHandler {
	return &OracleHandler{repo: repo}
}

func (h *OracleHandler) Routes(r chi.Router) {
	r.Get("/sources", h.Sources)
	r.Post("/fetch", h.Fetch)
	r.Get("/query", h.Query)
}

func (h *OracleHandler) Sources(w http.ResponseWriter, r *http.Request) {
	uc := oracleapp.NewListSourcesUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.ListSourcesRequest{})
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *OracleHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}

	uc := oracleapp.NewFetchDataUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: req.Source})
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *OracleHandler) Query(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	limitStr := r.URL.Query().Get("limit")
	limit := defaultQueryLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	uc := oracleapp.NewGetDataUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.GetDataRequest{
		SourceID: source,
		Limit:    limit,
	})
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
