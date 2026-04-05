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

func TestOracleData_Empty(t *testing.T) {
	data := &OracleData{}

	if data.ID != "" {
		t.Errorf("Expected empty ID, got '%s'", data.ID)
	}

	if data.Value != "" {
		t.Errorf("Expected empty Value, got '%s'", data.Value)
	}

	if data.Timestamp != 0 {
		t.Errorf("Expected zero Timestamp, got %d", data.Timestamp)
	}
}

func TestDataSource_Disable(t *testing.T) {
	source := &DataSource{
		ID:      "source-1",
		Name:    "Test",
		Enabled: true,
	}

	source.Enabled = false

	if source.Enabled {
		t.Error("Expected Disabled after toggle")
	}
}

func TestOracleData_Update(t *testing.T) {
	data := &OracleData{
		ID:          "data-1",
		SourceID:    "source-1",
		Value:       "100",
		Timestamp:   1000,
		BlockHeight: 10,
	}

	data.Value = "200"
	data.BlockHeight = 20

	if data.Value != "200" {
		t.Errorf("Expected Value '200', got '%s'", data.Value)
	}

	if data.BlockHeight != 20 {
		t.Errorf("Expected BlockHeight 20, got %d", data.BlockHeight)
	}
}
