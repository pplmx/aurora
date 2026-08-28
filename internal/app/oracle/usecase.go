package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/infra/http"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type FetcherInterface interface {
	FetchData(source *oracle.DataSource) (*oracle.OracleData, error)
}

// ctxFetcher is implemented by the real http.Fetcher (cancellable
// FetchDataContext); test doubles only implement the plain FetcherInterface.
// fetchData prefers the ctx-aware path so the scheduler can interrupt an
// in-flight fetch (TASK-134, ISS-127) without widening the public interface.
type ctxFetcher interface {
	FetchDataContext(ctx context.Context, source *oracle.DataSource) (*oracle.OracleData, error)
}

type FetchDataUseCase struct {
	repo    oracle.Repository
	fetcher FetcherInterface
	chain   ChainInterface
}

func NewFetchDataUseCase(repo oracle.Repository) *FetchDataUseCase {
	return &FetchDataUseCase{
		repo:    repo,
		fetcher: http.NewFetcher(),
	}
}

func NewFetchDataUseCaseWithDeps(repo oracle.Repository, fetcher FetcherInterface) *FetchDataUseCase {
	return &FetchDataUseCase{
		repo:    repo,
		fetcher: fetcher,
	}
}

func (uc *FetchDataUseCase) SetChain(chain ChainInterface) {
	uc.chain = chain
}

// fetchData runs the fetch through the ctx-aware path when the fetcher
// supports it (the scheduler depends on cancellation), otherwise falls back
// to the plain interface so test doubles keep compiling unchanged.
func (uc *FetchDataUseCase) fetchData(ctx context.Context, source *oracle.DataSource) (*oracle.OracleData, error) {
	if cf, ok := uc.fetcher.(ctxFetcher); ok {
		return cf.FetchDataContext(ctx, source)
	}
	return uc.fetcher.FetchData(source)
}

// Chain returns the on-chain recorder currently wired to the use case, or nil
// if none. Exposed for inspection: the TASK-097 regression asserts the oracle
// scheduler's fetch (unlike the REST handler's) is no longer running with a
// nil chain, which silently stored every scheduler observation at
// block_height=0.
func (uc *FetchDataUseCase) Chain() ChainInterface { return uc.chain }

func (uc *FetchDataUseCase) Execute(req *FetchDataRequest) (*FetchDataResponse, error) {
	return uc.executeWithChain(context.Background(), req, uc.chain)
}

// ExecuteContext is the cancellable form of Execute used by the oracle
// scheduler (TASK-134, ISS-127): the in-flight HTTP fetch is interrupted when
// ctx is cancelled, so a shutdown no longer stalls up to the client timeout
// with the DB pool being torn down underneath the fetch goroutine.
func (uc *FetchDataUseCase) ExecuteContext(ctx context.Context, req *FetchDataRequest) (*FetchDataResponse, error) {
	return uc.executeWithChain(ctx, req, uc.chain)
}

func (uc *FetchDataUseCase) executeWithChain(ctx context.Context, req *FetchDataRequest, chain ChainInterface) (*FetchDataResponse, error) {
	source, err := uc.repo.GetSource(req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	if source == nil {
		return nil, oracle.ErrSourceNotFound
	}
	if !source.Enabled {
		return nil, oracle.ErrSourceDisabled
	}

	// SSRF TOCTOU guard (v1.54): the source URL is validated when the source is
	// added, but the actual dial may happen much later (or after the record is
	// edited in the DB), so the add-time check alone leaves a window in which a
	// hostname that was public can be DNS-rebound to a loopback / RFC1918 /
	// cloud-metadata address. Re-apply the domain SSRF policy immediately
	// before fetching so a rebound source fails closed instead of reaching
	// internal services. This runs on every fetch path (CLI, REST, TUI,
	// scheduler) that funnels through this use case. (An empty URL is not an
	// SSRF vector and is left to the fetcher's own empty-URL handling.)
	if source.URL != "" {
		if err := oracle.ValidateSourceURL(source.URL); err != nil {
			return nil, fmt.Errorf("blocked source URL: %w", err)
		}
	}

	data, err := uc.fetchData(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if chain != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal oracle data: %w", err)
		}
		height, err := chain.AddLotteryRecord(string(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to record on chain: %w", err)
		}
		data.BlockHeight = height
	}

	if err := uc.repo.SaveData(data); err != nil {
		return nil, fmt.Errorf("failed to save data: %w", err)
	}

	return &FetchDataResponse{
		ID:          data.ID,
		SourceID:    data.SourceID,
		Value:       data.Value,
		Timestamp:   data.Timestamp,
		BlockHeight: data.BlockHeight,
	}, nil
}

type ChainInterface interface {
	AddLotteryRecord(data string) (int64, error)
}

type AddSourceUseCase struct {
	repo oracle.Repository
}

func NewAddSourceUseCase(repo oracle.Repository) *AddSourceUseCase {
	return &AddSourceUseCase{repo: repo}
}

func (uc *AddSourceUseCase) Execute(req *AddSourceRequest) (*SourceResponse, error) {
	source := &oracle.DataSource{
		Name:     req.Name,
		URL:      req.URL,
		Type:     req.Type,
		Method:   req.Method,
		Path:     req.Path,
		Interval: req.Interval,
	}

	if source.Method == "" {
		source.Method = "GET"
	}
	if source.Type == "" {
		source.Type = "custom"
	}
	if source.Interval == 0 {
		source.Interval = 60
	}

	// Delegate to the domain service's AddSource. That is where URL-scheme
	// validation (SSRF guard: only http/https allowed, host required), name
	// validation, ID generation, and consistent defaults live. Calling
	// repo.SaveSource directly here would let a caller persist a source with
	// e.g. a file:// or empty-host URL that the fetcher could never reach —
	// the validation is part of the service contract, not the repo.
	if err := oracle.NewService(uc.repo).AddSource(source); err != nil {
		// Validation errors are already descriptive; keep them untouched so
		// API/CLI classify them as client errors. Persist failures get context.
		if errors.Is(err, oracle.ErrInvalidSource) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to save source: %w", err)
	}

	return &SourceResponse{
		ID:        source.ID,
		Name:      source.Name,
		URL:       source.URL,
		Type:      source.Type,
		Method:    source.Method,
		Headers:   source.Headers,
		Path:      source.Path,
		Interval:  source.Interval,
		Enabled:   source.Enabled,
		CreatedAt: source.CreatedAt,
	}, nil
}

type ListSourcesUseCase struct {
	repo oracle.Repository
}

func NewListSourcesUseCase(repo oracle.Repository) *ListSourcesUseCase {
	return &ListSourcesUseCase{repo: repo}
}

func (uc *ListSourcesUseCase) Execute(req *ListSourcesRequest) (*ListSourcesResponse, error) {
	sources, err := uc.repo.ListSources()
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}

	result := make([]*SourceResponse, 0, len(sources))
	for _, s := range sources {
		result = append(result, &SourceResponse{
			ID:        s.ID,
			Name:      s.Name,
			URL:       s.URL,
			Type:      s.Type,
			Method:    s.Method,
			Headers:   s.Headers,
			Path:      s.Path,
			Interval:  s.Interval,
			Enabled:   s.Enabled,
			CreatedAt: s.CreatedAt,
		})
	}

	return &ListSourcesResponse{Sources: result}, nil
}

type DeleteSourceUseCase struct {
	repo oracle.Repository
}

func NewDeleteSourceUseCase(repo oracle.Repository) *DeleteSourceUseCase {
	return &DeleteSourceUseCase{repo: repo}
}

func (uc *DeleteSourceUseCase) Execute(id string) error {
	return uc.repo.DeleteSource(id)
}

type EnableSourceUseCase struct {
	repo oracle.Repository
}

func NewEnableSourceUseCase(repo oracle.Repository) *EnableSourceUseCase {
	return &EnableSourceUseCase{repo: repo}
}

func (uc *EnableSourceUseCase) Execute(id string) error {
	// Atomic primitive: closes the TOCTOU window where
	// Enable-vs-UpdateURL could clobber each other's fields.
	if err := uc.repo.SetSourceEnabled(id, true); err != nil {
		if errors.Is(err, oracle.ErrSourceNotFound) || errors.Is(err, sqlite.ErrNotFound) {
			return oracle.ErrSourceNotFound
		}
		return err
	}
	return nil
}

type DisableSourceUseCase struct {
	repo oracle.Repository
}

func NewDisableSourceUseCase(repo oracle.Repository) *DisableSourceUseCase {
	return &DisableSourceUseCase{repo: repo}
}

func (uc *DisableSourceUseCase) Execute(id string) error {
	// Atomic primitive: closes the TOCTOU window where
	// Disable-vs-UpdateURL could clobber each other's fields.
	if err := uc.repo.SetSourceEnabled(id, false); err != nil {
		if errors.Is(err, oracle.ErrSourceNotFound) || errors.Is(err, sqlite.ErrNotFound) {
			return oracle.ErrSourceNotFound
		}
		return err
	}
	return nil
}

type GetDataUseCase struct {
	repo oracle.Repository
}

func NewGetDataUseCase(repo oracle.Repository) *GetDataUseCase {
	return &GetDataUseCase{repo: repo}
}

func (uc *GetDataUseCase) Execute(req *GetDataRequest) (*GetDataResponse, error) {
	data, err := uc.repo.GetDataBySource(req.SourceID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	result := make([]*DataResponse, 0, len(data))
	for _, d := range data {
		result = append(result, &DataResponse{
			ID:          d.ID,
			SourceID:    d.SourceID,
			Value:       d.Value,
			Timestamp:   d.Timestamp,
			BlockHeight: d.BlockHeight,
		})
	}

	return &GetDataResponse{Data: result}, nil
}

type GetLatestDataUseCase struct {
	repo oracle.Repository
}

func NewGetLatestDataUseCase(repo oracle.Repository) *GetLatestDataUseCase {
	return &GetLatestDataUseCase{repo: repo}
}

func (uc *GetLatestDataUseCase) Execute(req *GetLatestDataRequest) (*GetLatestDataResponse, error) {
	data, err := uc.repo.GetLatestData(req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest data: %w", err)
	}
	if data == nil {
		return &GetLatestDataResponse{Data: nil}, nil
	}

	return &GetLatestDataResponse{
		Data: &DataResponse{
			ID:          data.ID,
			SourceID:    data.SourceID,
			Value:       data.Value,
			Timestamp:   data.Timestamp,
			BlockHeight: data.BlockHeight,
		},
	}, nil
}
