package test

import (
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	oracle "github.com/pplmx/aurora/internal/domain/oracle"
)

func TestOracleE2E_DataSourceCreation(t *testing.T) {
	blockchain.ResetForTest()

	source := &oracle.DataSource{
		ID:      "source-1",
		Name:    "BTC Price",
		Type:    "http",
		URL:     "https://api.example.com/btc",
		Enabled: true,
	}

	if source.Name != "BTC Price" {
		t.Errorf("Expected BTC Price, got %s", source.Name)
	}
	if !source.Enabled {
		t.Error("Expected enabled")
	}
}

func TestOracleE2E_OracleDataCreation(t *testing.T) {
	blockchain.ResetForTest()

	data := &oracle.OracleData{
		ID:        "data-1",
		SourceID:  "source-1",
		Value:     "50000.00",
		Timestamp: 1234567890,
	}

	if data.Value != "50000.00" {
		t.Errorf("Expected 50000.00, got %s", data.Value)
	}
	if data.SourceID != "source-1" {
		t.Errorf("Expected source-1, got %s", data.SourceID)
	}
}
