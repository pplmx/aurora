package oracle

import (
	"testing"
	"time"
)

func TestOracleData_Fields(t *testing.T) {
	data := &OracleData{
		ID:          "data-1",
		SourceID:    "source-1",
		Value:       "100.50",
		RawResponse: `{"price": 100.50}`,
		Timestamp:   time.Now().Unix(),
		BlockHeight: 42,
	}

	if data.ID != "data-1" {
		t.Errorf("Expected ID 'data-1', got '%s'", data.ID)
	}

	if data.SourceID != "source-1" {
		t.Errorf("Expected SourceID 'source-1', got '%s'", data.SourceID)
	}

	if data.Value != "100.50" {
		t.Errorf("Expected Value '100.50', got '%s'", data.Value)
	}

	if data.BlockHeight != 42 {
		t.Errorf("Expected BlockHeight 42, got %d", data.BlockHeight)
	}
}

func TestDataSource_Fields(t *testing.T) {
	now := time.Now().Unix()
	source := &DataSource{
		ID:        "source-1",
		Name:      "BTC Price",
		URL:       "https://api.coindesk.com/v1/bpi/currentprice.json",
		Type:      "json",
		Method:    "GET",
		Headers:   "",
		Path:      "bpi.USD.rate_float",
		Interval:  60,
		Enabled:   true,
		CreatedAt: now,
	}

	if source.ID != "source-1" {
		t.Errorf("Expected ID 'source-1', got '%s'", source.ID)
	}

	if source.Name != "BTC Price" {
		t.Errorf("Expected Name 'BTC Price', got '%s'", source.Name)
	}

	if source.Type != "json" {
		t.Errorf("Expected Type 'json', got '%s'", source.Type)
	}

	if source.Method != "GET" {
		t.Errorf("Expected Method 'GET', got '%s'", source.Method)
	}

	if source.Interval != 60 {
		t.Errorf("Expected Interval 60, got %d", source.Interval)
	}

	if !source.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestDataSource_DefaultValues(t *testing.T) {
	source := &DataSource{
		ID:   "source-1",
		Name: "Test",
		URL:  "https://example.com",
	}

	if source.Method == "" {
		source.Method = "GET"
	}
	if source.Method != "GET" {
		t.Errorf("Expected default Method 'GET', got '%s'", source.Method)
	}

	if source.Type == "" {
		source.Type = "custom"
	}
	if source.Type != "custom" {
		t.Errorf("Expected default Type 'custom', got '%s'", source.Type)
	}

	if source.Interval == 0 {
		source.Interval = 60
	}
	if source.Interval != 60 {
		t.Errorf("Expected default Interval 60, got %d", source.Interval)
	}
}
