package oracle

import (
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
)

func TestFetchAndSave(t *testing.T) {
	storage := NewInMemoryStorage()
	InitOracle(storage)

	ds, err := RegisterDataSource("Test Source", "https://httpbin.org/json", "test", 60)
	if err != nil {
		t.Fatal(err)
	}

	chain := blockchain.InitBlockChain()

	data, err := FetchAndSave(ds.ID, chain)
	if err != nil {
		t.Logf("Expected network error: %v", err)
		return
	}

	if data != nil {
		t.Logf("Fetched data: %s", data.Value)
	}
}

func TestGetOracleData(t *testing.T) {
	storage := NewInMemoryStorage()
	InitOracle(storage)

	ds, _ := RegisterDataSource("Test", "https://example.com", "test", 60)

	data := &OracleData{
		ID:        "test-1",
		SourceID:  ds.ID,
		Value:     "test-value",
		Timestamp: 1234567890,
	}
	storage.SaveOracleData(data)

	latest, err := GetLatestOracleData(ds.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Value != "test-value" {
		t.Errorf("Value = %v, want test-value", latest.Value)
	}

	list, err := GetOracleData(ds.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %v, want 1", len(list))
	}
}
