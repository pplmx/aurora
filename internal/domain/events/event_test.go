package events

import (
	"testing"
	"time"
)

func TestBaseEvent_ImplementsEventInterface(t *testing.T) {
	payload := []byte(`{"key":"value"}`)
	e := NewBaseEvent("test.action", "agg-123", payload)

	if e.EventType() != "test.action" {
		t.Errorf("EventType() = %q, want %q", e.EventType(), "test.action")
	}
	if e.Module() != "test" {
		t.Errorf("Module() = %q, want %q", e.Module(), "test")
	}
	if e.AggregateID() != "agg-123" {
		t.Errorf("AggregateID() = %q, want %q", e.AggregateID(), "agg-123")
	}
	if time.Since(e.Timestamp()) > time.Second {
		t.Error("Timestamp() not set to current time")
	}
	if string(e.Payload()) != string(payload) {
		t.Errorf("Payload() = %q, want %q", e.Payload(), payload)
	}
}

func TestBaseEvent_ID(t *testing.T) {
	e1 := NewBaseEvent("test.a", "id1", nil)
	e2 := NewBaseEvent("test.b", "id2", nil)

	if e1.ID() == "" {
		t.Error("ID() should not be empty")
	}
	if e1.ID() == e2.ID() {
		t.Error("IDs should be unique")
	}
}

func TestNewBaseEvent_ParsesModule(t *testing.T) {
	tests := []struct {
		eventType string
		want      string
	}{
		{"token.transfer", "token"},
		{"nft.mint", "nft"},
		{"voting.vote", "voting"},
		{"lottery.created", "lottery"},
		{"oracle.data_fetched", "oracle"},
	}

	for _, tt := range tests {
		e := NewBaseEvent(tt.eventType, "agg", nil)
		if e.Module() != tt.want {
			t.Errorf("Module() for %q = %q, want %q", tt.eventType, e.Module(), tt.want)
		}
	}
}

func TestParsePayload(t *testing.T) {
	payload := []byte(`{"name":"test","value":42}`)
	e := NewBaseEvent("test.parse", "id", payload)

	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	if err := ParsePayload(e, &result); err != nil {
		t.Fatalf("ParsePayload() error = %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
	if result.Value != 42 {
		t.Errorf("Value = %d, want %d", result.Value, 42)
	}
}

func TestParsePayload_InvalidJSON(t *testing.T) {
	payload := []byte(`invalid json`)
	e := NewBaseEvent("test.parse", "id", payload)

	var result map[string]interface{}
	if err := ParsePayload(e, &result); err == nil {
		t.Error("ParsePayload() expected error for invalid JSON")
	}
}

func TestBaseEvent_Immutable(t *testing.T) {
	e := NewBaseEvent("test.immutable", "id", nil)

	originalID := e.ID()
	_ = originalID

	if e.ID() != originalID {
		t.Error("ID should be immutable")
	}
}
