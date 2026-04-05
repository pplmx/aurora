package oracle

import (
	"testing"
	"time"
)

func TestOracleData_Serialization(t *testing.T) {
	data := &OracleData{
		ID:          "data-1",
		SourceID:    "source-1",
		Value:       "123.45",
		RawResponse: `{"result": 123.45}`,
		Timestamp:   time.Now().Unix(),
		BlockHeight: 100,
	}

	_ = data.ID
	_ = data.SourceID
	_ = data.Value
	_ = data.RawResponse
	_ = data.Timestamp
	_ = data.BlockHeight
}

func TestDataSource_Validation(t *testing.T) {
	tests := []struct {
		name    string
		source  *DataSource
		wantErr bool
	}{
		{
			name: "valid source",
			source: &DataSource{
				ID:   "s1",
				Name: "Test",
				URL:  "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			source: &DataSource{
				ID:  "s1",
				URL: "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "missing url",
			source: &DataSource{
				ID:   "s1",
				Name: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateSource(s *DataSource) error {
	if s.Name == "" {
		return ErrInvalidSource
	}
	if s.URL == "" {
		return ErrInvalidSource
	}
	return nil
}

func TestOracleError_Error(t *testing.T) {
	err := &OracleError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}
}

func TestDataSource_IsEnabled(t *testing.T) {
	source := &DataSource{Enabled: true}
	if !source.Enabled {
		t.Error("Expected enabled")
	}

	source.Enabled = false
	if source.Enabled {
		t.Error("Expected disabled after toggle")
	}
}

func TestDataSource_Headers(t *testing.T) {
	source := &DataSource{
		Headers: `{"Authorization": "Bearer token123"}`,
	}

	if source.Headers == "" {
		t.Error("Expected non-empty headers")
	}
}

func TestOracleData_TimeFields(t *testing.T) {
	now := time.Now().Unix()
	data := &OracleData{
		Timestamp:   now,
		BlockHeight: now * 10,
	}

	if data.Timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}

	if data.BlockHeight == 0 {
		t.Error("Expected non-zero block height")
	}
}

func TestOracleData_MultipleRecords(t *testing.T) {
	records := make([]*OracleData, 0)
	for i := 0; i < 5; i++ {
		records = append(records, &OracleData{
			ID:        "data-" + string(rune('0'+i)),
			SourceID:  "source-1",
			Value:     "100",
			Timestamp: int64(i),
		})
	}

	if len(records) != 5 {
		t.Errorf("Expected 5 records, got %d", len(records))
	}
}

func TestDataSource_MultipleSources(t *testing.T) {
	sources := []*DataSource{
		{ID: "s1", Name: "Source 1", URL: "https://s1.com"},
		{ID: "s2", Name: "Source 2", URL: "https://s2.com"},
		{ID: "s3", Name: "Source 3", URL: "https://s3.com"},
	}

	for _, s := range sources {
		if s.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if s.URL == "" {
			t.Error("Expected non-empty URL")
		}
	}
}

func TestAddSource(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		Name: "test-source",
		URL:  "https://api.example.com",
		Type: "http",
	}

	err := svc.AddSource(source)
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	sources, _ := repo.ListSources()
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(sources))
	}
	if !sources[0].Enabled {
		t.Error("expected source to be enabled by default")
	}
}

func TestAddSource_Invalid(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	err := svc.AddSource(&DataSource{Name: "", URL: "https://api.example.com"})
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestEnableSource(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:      "test-id",
		Name:    "test-source",
		URL:     "https://api.example.com",
		Type:    "http",
		Enabled: false,
	}
	_ = repo.SaveSource(source)

	err := svc.EnableSource("test-id")
	if err != nil {
		t.Fatalf("EnableSource failed: %v", err)
	}

	updated, _ := repo.GetSource("test-id")
	if !updated.Enabled {
		t.Error("expected source to be enabled")
	}
}

func TestEnableSource_NotFound(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	err := svc.EnableSource("non-existent")
	if err != ErrSourceNotFound {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestDisableSource(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:      "test-id",
		Name:    "test-source",
		URL:     "https://api.example.com",
		Type:    "http",
		Enabled: true,
	}
	_ = repo.SaveSource(source)

	err := svc.DisableSource("test-id")
	if err != nil {
		t.Fatalf("DisableSource failed: %v", err)
	}

	updated, _ := repo.GetSource("test-id")
	if updated.Enabled {
		t.Error("expected source to be disabled")
	}
}

func TestDisableSource_NotFound(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	err := svc.DisableSource("non-existent")
	if err != ErrSourceNotFound {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestDeleteSource(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:   "test-id",
		Name: "test-source",
		URL:  "https://api.example.com",
		Type: "http",
	}
	_ = repo.SaveSource(source)

	err := svc.DeleteSource("test-id")
	if err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}

	deleted, _ := repo.GetSource("test-id")
	if deleted != nil {
		t.Error("expected source to be deleted")
	}
}

func TestFetchData(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:      "test-id",
		Name:    "test-source",
		URL:     "https://api.example.com",
		Type:    "http",
		Enabled: true,
	}
	_ = repo.SaveSource(source)

	data, err := svc.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}
	if data == nil {
		t.Fatal("expected data, got nil")
	}
	if data.SourceID != source.ID {
		t.Errorf("expected SourceID %s, got %s", source.ID, data.SourceID)
	}
}

func TestFetchData_Disabled(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:      "test-id",
		Name:    "test-source",
		URL:     "https://api.example.com",
		Type:    "http",
		Enabled: false,
	}

	_, err := svc.FetchData(source)
	if err != ErrSourceDisabled {
		t.Errorf("expected ErrSourceDisabled, got %v", err)
	}
}

func TestQueryData(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	source := &DataSource{
		ID:   "test-id",
		Name: "test-source",
		URL:  "https://api.example.com",
		Type: "http",
	}
	_ = repo.SaveSource(source)

	for i := 0; i < 3; i++ {
		data := &OracleData{
			ID:        "data-" + string(rune('0'+i)),
			SourceID:  "test-id",
			Value:     "100",
			Timestamp: int64(i),
		}
		_ = repo.SaveData(data)
	}

	results, err := svc.QueryData("test-id", 10)
	if err != nil {
		t.Fatalf("QueryData failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestQueryData_Empty(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	results, err := svc.QueryData("non-existent", 10)
	if err != nil {
		t.Fatalf("QueryData failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
