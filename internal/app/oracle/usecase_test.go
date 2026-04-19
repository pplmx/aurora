package oracle

import (
	"errors"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

type mockOracleRepo struct {
	sources   []*oracle.DataSource
	data      []*oracle.OracleData
	dataErr   error
	sourceErr error
}

func (m *mockOracleRepo) SaveData(d *oracle.OracleData) error {
	m.data = append(m.data, d)
	return m.dataErr
}

func (m *mockOracleRepo) GetData(id string) (*oracle.OracleData, error) {
	for _, d := range m.data {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

func (m *mockOracleRepo) GetDataBySource(sourceID string, limit int) ([]*oracle.OracleData, error) {
	var result []*oracle.OracleData
	for _, d := range m.data {
		if d.SourceID == sourceID {
			result = append(result, d)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, m.dataErr
}

func (m *mockOracleRepo) GetLatestData(sourceID string) (*oracle.OracleData, error) {
	if m.dataErr != nil {
		return nil, m.dataErr
	}
	var latest *oracle.OracleData
	for _, d := range m.data {
		if d.SourceID == sourceID {
			if latest == nil || d.Timestamp > latest.Timestamp {
				latest = d
			}
		}
	}
	return latest, nil
}

func (m *mockOracleRepo) GetDataByTimeRange(sourceID string, start, end int64) ([]*oracle.OracleData, error) {
	var result []*oracle.OracleData
	for _, d := range m.data {
		if d.SourceID == sourceID && d.Timestamp >= start && d.Timestamp <= end {
			result = append(result, d)
		}
	}
	return result, m.dataErr
}

func (m *mockOracleRepo) SaveSource(s *oracle.DataSource) error {
	s.ID = "test-id-" + s.Name
	s.CreatedAt = time.Now().Unix()
	m.sources = append(m.sources, s)
	return m.sourceErr
}

func (m *mockOracleRepo) GetSource(id string) (*oracle.DataSource, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, m.sourceErr
}

func (m *mockOracleRepo) ListSources() ([]*oracle.DataSource, error) {
	return m.sources, m.sourceErr
}

func (m *mockOracleRepo) UpdateSource(s *oracle.DataSource) error {
	return m.sourceErr
}

func (m *mockOracleRepo) DeleteSource(id string) error {
	return m.sourceErr
}

func TestAddSourceUseCase(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewAddSourceUseCase(repo)

	req := &AddSourceRequest{
		Name:     "Test Source",
		URL:      "https://api.example.com/data",
		Type:     "json",
		Method:   "GET",
		Path:     "data.value",
		Interval: 60,
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Name != "Test Source" {
		t.Errorf("Expected name 'Test Source', got '%s'", resp.Name)
	}

	if !resp.Enabled {
		t.Error("Expected enabled to be true")
	}
}

func TestAddSourceUseCase_DefaultValues(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewAddSourceUseCase(repo)

	req := &AddSourceRequest{
		Name: "Test Source",
		URL:  "https://api.example.com/data",
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Type != "custom" {
		t.Errorf("Expected default type 'custom', got '%s'", resp.Type)
	}

	if resp.Enabled != true {
		t.Error("Expected enabled to be true by default")
	}
}

func TestListSourcesUseCase(t *testing.T) {
	repo := &mockOracleRepo{
		sources: []*oracle.DataSource{
			{ID: "1", Name: "Source 1", Enabled: true},
			{ID: "2", Name: "Source 2", Enabled: false},
		},
	}
	uc := NewListSourcesUseCase(repo)

	resp, err := uc.Execute(&ListSourcesRequest{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(resp.Sources))
	}
}

func TestListSourcesUseCase_Empty(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewListSourcesUseCase(repo)

	resp, err := uc.Execute(&ListSourcesRequest{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp.Sources) != 0 {
		t.Errorf("Expected 0 sources, got %d", len(resp.Sources))
	}
}

func TestDeleteSourceUseCase(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewDeleteSourceUseCase(repo)

	err := uc.Execute("test-id")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestEnableSourceUseCase(t *testing.T) {
	repo := &mockOracleRepo{
		sources: []*oracle.DataSource{
			{ID: "test-id", Name: "Test", Enabled: false},
		},
	}
	uc := NewEnableSourceUseCase(repo)

	err := uc.Execute("test-id")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !repo.sources[0].Enabled {
		t.Error("Expected source to be enabled")
	}
}

func TestDisableSourceUseCase(t *testing.T) {
	repo := &mockOracleRepo{
		sources: []*oracle.DataSource{
			{ID: "test-id", Name: "Test", Enabled: true},
		},
	}
	uc := NewDisableSourceUseCase(repo)

	err := uc.Execute("test-id")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if repo.sources[0].Enabled {
		t.Error("Expected source to be disabled")
	}
}

func TestGetDataUseCase(t *testing.T) {
	repo := &mockOracleRepo{
		data: []*oracle.OracleData{
			{ID: "1", SourceID: "source-1", Value: "100"},
			{ID: "2", SourceID: "source-1", Value: "200"},
		},
	}
	uc := NewGetDataUseCase(repo)

	resp, err := uc.Execute(&GetDataRequest{SourceID: "source-1", Limit: 10})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 data items, got %d", len(resp.Data))
	}
}

func TestGetLatestDataUseCase(t *testing.T) {
	repo := &mockOracleRepo{
		data: []*oracle.OracleData{
			{ID: "1", SourceID: "source-1", Value: "100", Timestamp: 1000},
			{ID: "2", SourceID: "source-1", Value: "200", Timestamp: 2000},
		},
	}
	uc := NewGetLatestDataUseCase(repo)

	resp, err := uc.Execute(&GetLatestDataRequest{SourceID: "source-1"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Expected data to not be nil")
	}

	if resp.Data.Value != "200" {
		t.Errorf("Expected latest value '200', got '%s'", resp.Data.Value)
	}
}

func TestGetLatestDataUseCase_Empty(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewGetLatestDataUseCase(repo)

	resp, err := uc.Execute(&GetLatestDataRequest{SourceID: "source-1"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Data != nil {
		t.Error("Expected data to be nil for empty source")
	}
}

func TestFetchDataUseCase_SourceNotFound(t *testing.T) {
	repo := &mockOracleRepo{}
	uc := NewFetchDataUseCase(repo)

	_, err := uc.Execute(&FetchDataRequest{SourceID: "nonexistent"})
	if err == nil {
		t.Fatal("Expected error for nonexistent source")
	}
}

func TestFetchDataUseCase_DisabledSource(t *testing.T) {
	repo := &mockOracleRepo{
		sources: []*oracle.DataSource{
			{ID: "test-id", Name: "Test", Enabled: false},
		},
	}
	uc := NewFetchDataUseCase(repo)

	_, err := uc.Execute(&FetchDataRequest{SourceID: "test-id"})
	if err == nil {
		t.Fatal("Expected error for disabled source")
	}
}

func TestFetchDataUseCase_RepoError(t *testing.T) {
	repo := &mockOracleRepo{
		sources: []*oracle.DataSource{
			{ID: "test-id", Name: "Test", Enabled: true},
		},
		sourceErr: errors.New("repo error"),
	}
	uc := NewFetchDataUseCase(repo)

	_, err := uc.Execute(&FetchDataRequest{SourceID: "test-id"})
	if err == nil {
		t.Fatal("Expected error for repo error")
	}
}
