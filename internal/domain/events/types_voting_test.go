package events

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestVotingCreatedEvent(t *testing.T) {
	proposer := base64.StdEncoding.EncodeToString([]byte("proposer_pubkey"))

	payload, err := json.Marshal(map[string]interface{}{
		"proposer": proposer,
		"proposal": "Increase block reward",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &VotingCreatedEvent{
		BaseEvent: NewBaseEvent("voting.created", "vote123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "voting.created" {
			t.Errorf("expected voting.created, got %s", event.EventType())
		}
		if event.Module() != "voting" {
			t.Errorf("expected voting, got %s", event.Module())
		}
	})

	t.Run("Proposer accessor", func(t *testing.T) {
		got, err := event.Proposer()
		if err != nil {
			t.Fatalf("Proposer() error = %v", err)
		}
		if string(got) != "proposer_pubkey" {
			t.Errorf("Proposer() = %s, want proposer_pubkey", string(got))
		}
	})

	t.Run("Proposal accessor", func(t *testing.T) {
		got, err := event.Proposal()
		if err != nil {
			t.Fatalf("Proposal() error = %v", err)
		}
		if got != "Increase block reward" {
			t.Errorf("Proposal() = %s, want Increase block reward", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &VotingCreatedEvent{
			BaseEvent: NewBaseEvent("voting.created", "vote123", []byte("invalid")),
		}
		_, err := badEvent.Proposer()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		badPayload, _ := json.Marshal(map[string]interface{}{
			"proposer": "!!!invalid!!!",
			"proposal": "test",
		})
		badEvent := &VotingCreatedEvent{
			BaseEvent: NewBaseEvent("voting.created", "vote123", badPayload),
		}
		_, err := badEvent.Proposer()
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestVotingVoteEvent(t *testing.T) {
	voter := base64.StdEncoding.EncodeToString([]byte("voter_pubkey"))

	payload, err := json.Marshal(map[string]interface{}{
		"voter":  voter,
		"choice": "yes",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &VotingVoteEvent{
		BaseEvent: NewBaseEvent("voting.vote", "vote123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "voting.vote" {
			t.Errorf("expected voting.vote, got %s", event.EventType())
		}
	})

	t.Run("Voter accessor", func(t *testing.T) {
		got, err := event.Voter()
		if err != nil {
			t.Fatalf("Voter() error = %v", err)
		}
		if string(got) != "voter_pubkey" {
			t.Errorf("Voter() = %s, want voter_pubkey", string(got))
		}
	})

	t.Run("Choice accessor", func(t *testing.T) {
		got, err := event.Choice()
		if err != nil {
			t.Fatalf("Choice() error = %v", err)
		}
		if got != "yes" {
			t.Errorf("Choice() = %s, want yes", got)
		}
	})
}
