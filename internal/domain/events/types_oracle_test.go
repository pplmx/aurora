package events

import (
	"encoding/json"
	"testing"
)

func TestOracleDataFetchedEvent(t *testing.T) {
	payload, err := json.Marshal(oracleDataFetchedPayload{
		Source: "price_feed",
		Data:   map[string]interface{}{"btc_usd": 50000.0, "eth_usd": 3000.0},
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &OracleDataFetchedEvent{
		BaseEvent: NewBaseEvent("oracle.data_fetched", "oracle123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "oracle.data_fetched" {
			t.Errorf("expected oracle.data_fetched, got %s", event.EventType())
		}
		if event.Module() != "oracle" {
			t.Errorf("expected oracle, got %s", event.Module())
		}
	})

	t.Run("Source accessor", func(t *testing.T) {
		got, err := event.Source()
		if err != nil {
			t.Fatalf("Source() error = %v", err)
		}
		if got != "price_feed" {
			t.Errorf("Source() = %s, want price_feed", got)
		}
	})

	t.Run("Data accessor", func(t *testing.T) {
		got, err := event.Data()
		if err != nil {
			t.Fatalf("Data() error = %v", err)
		}
		dataMap, ok := got.(map[string]interface{})
		if !ok {
			t.Fatalf("Data() returned non-map type: %T", got)
		}
		if dataMap["btc_usd"] != 50000.0 {
			t.Errorf("Data().btc_usd = %v, want 50000.0", dataMap["btc_usd"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &OracleDataFetchedEvent{
			BaseEvent: NewBaseEvent("oracle.data_fetched", "oracle123", []byte("invalid")),
		}
		_, err := badEvent.Source()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestBase64DecodeFieldHelper(t *testing.T) {
	t.Run("valid field", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"address": "dGVzdF9hZGRyZXNz", // base64 of "test_address"
		})
		got, err := base64DecodeField(payload, "address")
		if err != nil {
			t.Fatalf("base64DecodeField() error = %v", err)
		}
		if string(got) != "test_address" {
			t.Errorf("base64DecodeField() = %s, want test_address", string(got))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := base64DecodeField([]byte("invalid"), "field")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("field not a string", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"count": 42,
		})
		_, err := base64DecodeField(payload, "count")
		if err == nil {
			t.Error("expected error for non-string field")
		}
	})

	t.Run("field missing", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{})
		_, err := base64DecodeField(payload, "missing")
		if err == nil {
			t.Error("expected error for missing field")
		}
	})

	t.Run("invalid base64 value", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"data": "!!!invalid!!!",
		})
		_, err := base64DecodeField(payload, "data")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}
