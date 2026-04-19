package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

func setupOracleTestDB(t *testing.T) (*OracleRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_oracle.db")

	repo, err := NewOracleRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create oracle repository: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewOracleRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewOracleRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.db.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
}

func TestOracleRepository_SaveSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{
		ID:        "btc-price",
		Name:      "BTC Price",
		URL:       "https://api.coindesk.com/v1/bpi/currentprice.json",
		Type:      "json",
		Method:    "GET",
		Path:      "bpi.USD.rate_float",
		Interval:  60,
		Enabled:   true,
		CreatedAt: 1234567890,
	}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}
}

func TestOracleRepository_GetSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{
		ID:        "btc-price",
		Name:      "BTC Price",
		URL:       "https://api.coindesk.com/v1/bpi/currentprice.json",
		Type:      "json",
		Method:    "GET",
		Path:      "bpi.USD.rate_float",
		Interval:  60,
		Enabled:   true,
		CreatedAt: 1234567890,
	}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}

	retrieved, err := repo.GetSource("btc-price")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	if retrieved.ID != "btc-price" {
		t.Errorf("Expected ID 'btc-price', got '%s'", retrieved.ID)
	}

	if retrieved.Name != "BTC Price" {
		t.Errorf("Expected name 'BTC Price', got '%s'", retrieved.Name)
	}
}

func TestOracleRepository_GetSource_NotFound(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	_, err := repo.GetSource("NOTEXIST")
	if err != nil {
		t.Fatalf("Expected nil for non-existent source, got error: %v", err)
	}
}

func TestOracleRepository_ListSources(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source1 := &oracle.DataSource{ID: "source1", Name: "Source 1", URL: "http://example.com/1", Enabled: true}
	source2 := &oracle.DataSource{ID: "source2", Name: "Source 2", URL: "http://example.com/2", Enabled: true}
	source3 := &oracle.DataSource{ID: "source3", Name: "Source 3", URL: "http://example.com/3", Enabled: false}

	err := repo.SaveSource(source1)
	if err != nil {
		t.Fatalf("Failed to save source1: %v", err)
	}
	err = repo.SaveSource(source2)
	if err != nil {
		t.Fatalf("Failed to save source2: %v", err)
	}
	err = repo.SaveSource(source3)
	if err != nil {
		t.Fatalf("Failed to save source3: %v", err)
	}

	sources, err := repo.ListSources()
	if err != nil {
		t.Fatalf("Failed to list sources: %v", err)
	}

	if len(sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(sources))
	}
}

func TestOracleRepository_SaveData(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data := &oracle.OracleData{
		ID:          "data-1",
		SourceID:    "btc-price",
		Value:       "50000.00",
		RawResponse: "{}",
		Timestamp:   1234567890,
		BlockHeight: 1,
	}

	err := repo.SaveData(data)
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}
}

func TestOracleRepository_GetDataBySource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data1 := &oracle.OracleData{ID: "data-1", SourceID: "btc-price", Value: "50000", Timestamp: 1000}
	data2 := &oracle.OracleData{ID: "data-2", SourceID: "btc-price", Value: "51000", Timestamp: 2000}
	data3 := &oracle.OracleData{ID: "data-3", SourceID: "eth-price", Value: "3000", Timestamp: 1500}

	err := repo.SaveData(data1)
	if err != nil {
		t.Fatalf("Failed to save data1: %v", err)
	}
	err = repo.SaveData(data2)
	if err != nil {
		t.Fatalf("Failed to save data2: %v", err)
	}
	err = repo.SaveData(data3)
	if err != nil {
		t.Fatalf("Failed to save data3: %v", err)
	}

	results, err := repo.GetDataBySource("btc-price", 10)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestOracleRepository_GetLatestData(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data1 := &oracle.OracleData{ID: "data-1", SourceID: "btc-price", Value: "50000", Timestamp: 1000}
	data2 := &oracle.OracleData{ID: "data-2", SourceID: "btc-price", Value: "51000", Timestamp: 2000}

	err := repo.SaveData(data1)
	if err != nil {
		t.Fatalf("Failed to save data1: %v", err)
	}
	err = repo.SaveData(data2)
	if err != nil {
		t.Fatalf("Failed to save data2: %v", err)
	}

	latest, err := repo.GetLatestData("btc-price")
	if err != nil {
		t.Fatalf("Failed to get latest data: %v", err)
	}
	if latest.Value != "51000" {
		t.Errorf("Expected value '51000', got '%s'", latest.Value)
	}
}

func TestOracleRepository_DeleteSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{ID: "to-delete", Name: "Delete Me", URL: "http://example.com", Enabled: true}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}

	err = repo.DeleteSource("to-delete")
	if err != nil {
		t.Fatalf("Failed to delete source: %v", err)
	}

	_, err = repo.GetSource("to-delete")
	if err != nil {
		t.Fatalf("Source should be deleted: %v", err)
	}
}
