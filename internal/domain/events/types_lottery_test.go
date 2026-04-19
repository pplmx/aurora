package events

import (
	"encoding/json"
	"testing"
)

func TestLotteryCreatedEvent(t *testing.T) {
	payload, err := json.Marshal(lotteryCreatedPayload{
		Participants: "A,B,C,D,E",
		WinnerCount:  2,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &LotteryCreatedEvent{
		BaseEvent: NewBaseEvent("lottery.created", "lottery123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "lottery.created" {
			t.Errorf("expected lottery.created, got %s", event.EventType())
		}
		if event.Module() != "lottery" {
			t.Errorf("expected lottery, got %s", event.Module())
		}
	})

	t.Run("Participants accessor", func(t *testing.T) {
		got, err := event.Participants()
		if err != nil {
			t.Fatalf("Participants() error = %v", err)
		}
		if got != "A,B,C,D,E" {
			t.Errorf("Participants() = %s, want A,B,C,D,E", got)
		}
	})

	t.Run("WinnerCount accessor", func(t *testing.T) {
		got, err := event.WinnerCount()
		if err != nil {
			t.Fatalf("WinnerCount() error = %v", err)
		}
		if got != 2 {
			t.Errorf("WinnerCount() = %d, want 2", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &LotteryCreatedEvent{
			BaseEvent: NewBaseEvent("lottery.created", "lottery123", []byte("invalid")),
		}
		_, err := badEvent.Participants()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestLotteryDrawnEvent(t *testing.T) {
	payload, err := json.Marshal(lotteryDrawnPayload{
		Winners: "B,D",
		Proof:   "vrf_proof_base64",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &LotteryDrawnEvent{
		BaseEvent: NewBaseEvent("lottery.drawn", "lottery123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "lottery.drawn" {
			t.Errorf("expected lottery.drawn, got %s", event.EventType())
		}
	})

	t.Run("Winners accessor", func(t *testing.T) {
		got, err := event.Winners()
		if err != nil {
			t.Fatalf("Winners() error = %v", err)
		}
		if got != "B,D" {
			t.Errorf("Winners() = %s, want B,D", got)
		}
	})

	t.Run("Proof accessor", func(t *testing.T) {
		got, err := event.Proof()
		if err != nil {
			t.Fatalf("Proof() error = %v", err)
		}
		if got != "vrf_proof_base64" {
			t.Errorf("Proof() = %s, want vrf_proof_base64", got)
		}
	})
}
