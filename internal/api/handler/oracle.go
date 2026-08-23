package handler

import (
	"encoding/json"
	"errors"
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
	r.Post("/sources", h.CreateSource)
	r.Delete("/sources/{id}", h.DeleteSource)
	r.Patch("/sources/{id}", h.SetSourceEnabled)
	r.Post("/fetch", h.Fetch)
	r.Get("/query", h.Query)
	r.Get("/latest", h.Latest)
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

// CreateSource adds a new data source (v1.40). Previously adding a source was
// CLI-only; the REST API and web UI could list and fetch but never bootstrap a
// feed. Validation lives in the domain (URL-scheme SSRF guard, name checks).
func (h *OracleHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Type     string `json:"type"`
		Method   string `json:"method"`
		Path     string `json:"path"`
		Interval int    `json:"interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}

	uc := oracleapp.NewAddSourceUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.AddSourceRequest{
		Name:     req.Name,
		URL:      req.URL,
		Type:     req.Type,
		Method:   req.Method,
		Path:     req.Path,
		Interval: req.Interval,
	})
	if err != nil {
		if errors.Is(err, oracle.ErrInvalidSource) {
			writeBadRequest(w, err.Error())
			return
		}
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// DeleteSource removes a data source (v1.41).
func (h *OracleHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := oracleapp.NewDeleteSourceUseCase(h.repo)
	if err := uc.Execute(id); err != nil {
		if errors.Is(err, oracle.ErrSourceNotFound) {
			writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		}
		writeUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// SetSourceEnabled enables or disables a source based on the PATCH body's
// `enabled` boolean (v1.41).
func (h *OracleHandler) SetSourceEnabled(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}
	if req.Enabled == nil {
		writeBadRequest(w, "enabled field is required")
		return
	}

	var err error
	if *req.Enabled {
		err = oracleapp.NewEnableSourceUseCase(h.repo).Execute(id)
	} else {
		err = oracleapp.NewDisableSourceUseCase(h.repo).Execute(id)
	}
	if err != nil {
		if errors.Is(err, oracle.ErrSourceNotFound) {
			writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		}
		writeUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": *req.Enabled})
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

// Latest returns the single most recent data point for a source (v1.45). The
// CLI exposed this as `oracle latest` but the REST API and web UI had no way to
// read a source's current value.
func (h *OracleHandler) Latest(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		writeBadRequest(w, "source parameter is required")
		return
	}

	uc := oracleapp.NewGetLatestDataUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.GetLatestDataRequest{SourceID: source})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
