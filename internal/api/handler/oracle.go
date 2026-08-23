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

// maxQueryLimit caps the user-supplied ?limit= so a key-holding caller cannot
// force an arbitrarily large DB scan by passing e.g. ?limit=999999999.
const maxQueryLimit = 100

type OracleHandler struct {
	repo  oracle.Repository
	chain oracleapp.ChainInterface
	stats oracleHealthStats
}

// oracleHealthStats is the scheduler's fetch-health surface the health
// endpoint needs. Returning the interface (rather than the concrete
// *oracleapp.Scheduler) keeps the handler fakes easy and the dependency a
// single method.
type oracleHealthStats interface {
	Stats() []oracleapp.SourceStat
}

func NewOracleHandler(repo oracle.Repository) *OracleHandler {
	return &OracleHandler{repo: repo}
}

// SetChain wires the blockchain writer used to record fetched data on-chain.
// It is optional and nil-safe: without it fetched data keeps BlockHeight 0
// (evaluable tests / in-memory setups), matching the documented intent only
// when the caller (the API server) supplies the chain.
func (h *OracleHandler) SetChain(chain oracleapp.ChainInterface) {
	h.chain = chain
}

// SetStats wires the oracle fetch-health statistics source (the scheduler).
// It is optional and nil-safe: without it the health endpoint returns an
// empty list rather than panicking.
func (h *OracleHandler) SetStats(stats oracleHealthStats) {
	h.stats = stats
}

func (h *OracleHandler) Routes(r chi.Router) {
	r.Get("/sources", h.Sources)
	r.Post("/fetch", h.Fetch)
	r.Get("/query", h.Query)
	r.Get("/health", h.Health)
}

func (h *OracleHandler) Sources(w http.ResponseWriter, r *http.Request) {
	uc := oracleapp.NewListSourcesUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.ListSourcesRequest{})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Health returns the scheduler's per-source fetch-health statistics as JSON
// (v1.33). The same data is exposed as Prometheus text on /metrics/oracle, but
// this protected endpoint gives operators a readable, key-authenticated JSON
// view that the web UI consumes. It is nil-safe: before the scheduler starts
// it returns an empty list.
func (h *OracleHandler) Health(w http.ResponseWriter, r *http.Request) {
	stats := []oracleapp.SourceStat{}
	if h.stats != nil {
		stats = h.stats.Stats()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
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
	uc.SetChain(h.chain)
	result, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: req.Source})
	if err != nil {
		writeUseCaseError(w, err)
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
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxQueryLimit {
				limit = maxQueryLimit
			}
		}
	}

	uc := oracleapp.NewGetDataUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.GetDataRequest{
		SourceID: source,
		Limit:    limit,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
