package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

// defaultQueryLimit is the default limit for oracle data query API responses.
const defaultQueryLimit = 10

// maxQueryLimit caps the user-supplied ?limit= so a key-holding caller cannot
// force an arbitrarily large DB scan by passing e.g. ?limit=999999999.
const maxQueryLimit = 100

const (
	defaultOracleSourceLimit = 20
	maxOracleSourceLimit     = 100
	maxOracleSourceOffset    = 10000
)

type OracleHandler struct {
	repo  oracle.Repository
	chain oracleapp.ChainInterface
	stats oracleHealthStats
	// fetcher is shared across requests so the outbound per-source
	// http.rateLimit budget actually throttles REST fetches. A fetcher built
	// per request starts a fresh limiter every time, making the documented
	// rate limit a no-op at the endpoint that advertises it (TASK-241,
	// ISS-239). The scheduler already shares one fetcher the same way.
	fetcher oracleapp.FetcherInterface
}

// oracleHealthStats is the scheduler's fetch-health surface the health
// endpoint needs. Returning the interface (rather than the concrete
// *oracleapp.Scheduler) keeps the handler fakes easy and the dependency a
// single method.
type oracleHealthStats interface {
	Stats() []oracleapp.SourceStat
}

// NewOracleHandler builds a handler without a fetcher; Fetch on a source that
// reaches the network then reports a configuration error. Production uses
// NewOracleHandlerWithFetcher and the container's shared client.
func NewOracleHandler(repo oracle.Repository) *OracleHandler {
	return &OracleHandler{repo: repo}
}

func NewOracleHandlerWithFetcher(repo oracle.Repository, fetcher oracleapp.FetcherInterface) *OracleHandler {
	return &OracleHandler{repo: repo, fetcher: fetcher}
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
	r.Get("/templates", h.Templates)
	r.Post("/fetch", h.Fetch)
	r.Get("/query", h.Query)
	r.Get("/latest", h.Latest)
	r.Get("/health", h.Health)
}

func (h *OracleHandler) Sources(w http.ResponseWriter, r *http.Request) {
	limit, offset := oracleSourcePage(r)
	uc := oracleapp.NewListSourcesUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.ListSourcesRequest{Limit: limit, Offset: offset})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func oracleSourcePage(r *http.Request) (limit, offset int) {
	return pageParams(r, defaultOracleSourceLimit, maxOracleSourceLimit, maxOracleSourceOffset)
}

// Templates lists the built-in oracle source presets (v1.49). Previously the
// catalog lived only in the CLI; exposing it over REST + the shared app-layer
// catalog lets a client offer one-click source presets.
func (h *OracleHandler) Templates(w http.ResponseWriter, r *http.Request) {
	writeList(w, oracleapp.ListTemplates())
}

// CreateSource adds a new data source (v1.40). Previously adding a source was
// CLI-only; the REST API could list and fetch but never bootstrap a
// feed. Validation lives in the domain (URL-scheme SSRF guard, name checks).
func (h *OracleHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string            `json:"name"`
		URL      string            `json:"url"`
		Type     string            `json:"type"`
		Method   string            `json:"method"`
		Headers  map[string]string `json:"headers"`
		Path     string            `json:"path"`
		Interval int               `json:"interval"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	headers := ""
	if len(req.Headers) > 0 {
		encoded, err := json.Marshal(req.Headers)
		if err != nil {
			writeUseCaseError(w, fmt.Errorf("%w: invalid headers json", oracle.ErrInvalidSource))
			return
		}
		headers = string(encoded)
	}

	uc := oracleapp.NewAddSourceUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.AddSourceRequest{
		Name:     req.Name,
		URL:      req.URL,
		Type:     req.Type,
		Method:   req.Method,
		Headers:  headers,
		Path:     req.Path,
		Interval: req.Interval,
	})
	if err != nil {
		// The generic path maps ErrInvalidSource to 400 INVALID_SOURCE, the
		// same code every other oracle endpoint uses for it — a per-endpoint
		// special case here returned INVALID_REQUEST for the identical error.
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SetSourceEnabled enables or disables a source based on the PATCH body's
// `enabled` boolean (v1.41).
func (h *OracleHandler) SetSourceEnabled(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &req) {
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
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": *req.Enabled})
}

// Health returns the scheduler's per-source fetch-health statistics as JSON
// (v1.33). The same data is exposed as Prometheus text on /metrics/oracle, but
// this protected endpoint gives operators a readable, key-authenticated JSON
// view that a client consumes. It is nil-safe: before the scheduler starts
// it returns an empty list.
func (h *OracleHandler) Health(w http.ResponseWriter, r *http.Request) {
	stats := []oracleapp.SourceStat{}
	if h.stats != nil {
		stats = h.stats.Stats()
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *OracleHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := oracleapp.NewFetchDataUseCaseWithDeps(h.repo, h.fetcher)
	uc.SetChain(h.chain)
	result, err := uc.ExecuteContext(r.Context(), &oracleapp.FetchDataRequest{SourceID: req.Source})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *OracleHandler) Query(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		// Parity with /latest: an explicit source is required. Previously a
		// missing source fell through to 200 [] (ISS-130).
		writeBadRequest(w, "source parameter is required")
		return
	}

	limit := pageLimit(r, defaultQueryLimit, maxQueryLimit)

	uc := oracleapp.NewGetDataUseCase(h.repo)
	result, err := uc.Execute(&oracleapp.GetDataRequest{
		SourceID: source,
		Limit:    limit,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Latest returns the single most recent data point for a source (v1.45). The
// CLI exposed this as `oracle latest` but the REST API had no way to
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

	writeJSON(w, http.StatusOK, result)
}
