package oracle

import (
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/infra/http"
)

type FetcherInterface interface {
	FetchData(source *oracle.DataSource) (*oracle.OracleData, error)
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

func (uc *FetchDataUseCase) Execute(req *FetchDataRequest) (*FetchDataResponse, error) {
	return uc.executeWithChain(req, uc.chain)
}

func (uc *FetchDataUseCase) executeWithChain(req *FetchDataRequest, chain ChainInterface) (*FetchDataResponse, error) {
	source, err := uc.repo.GetSource(req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	if source == nil {
		return nil, fmt.Errorf("data source not found")
	}
	if !source.Enabled {
		return nil, fmt.Errorf("data source is disabled")
	}

	data, err := uc.fetcher.FetchData(source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if chain != nil {
		jsonData, _ := json.Marshal(data)
		height, _ := chain.AddLotteryRecord(string(jsonData))
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
		Enabled:  true,
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

	if err := uc.repo.SaveSource(source); err != nil {
		return nil, fmt.Errorf("failed to save source: %w", err)
	}

	return &SourceResponse{
		ID:        source.ID,
		Name:      source.Name,
		URL:       source.URL,
		Type:      source.Type,
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
	ds, err := uc.repo.GetSource(id)
	if err != nil {
		return err
	}
	if ds == nil {
		return fmt.Errorf("source not found")
	}
	ds.Enabled = true
	return uc.repo.UpdateSource(ds)
}

type DisableSourceUseCase struct {
	repo oracle.Repository
}

func NewDisableSourceUseCase(repo oracle.Repository) *DisableSourceUseCase {
	return &DisableSourceUseCase{repo: repo}
}

func (uc *DisableSourceUseCase) Execute(id string) error {
	ds, err := uc.repo.GetSource(id)
	if err != nil {
		return err
	}
	if ds == nil {
		return fmt.Errorf("source not found")
	}
	ds.Enabled = false
	return uc.repo.UpdateSource(ds)
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
