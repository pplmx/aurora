package events

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestNFTMintEvent(t *testing.T) {
	owner := base64.StdEncoding.EncodeToString([]byte("owner_pubkey"))

	payload, err := json.Marshal(map[string]interface{}{
		"owner":    owner,
		"metadata": "ipfs://QmExample",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &NFTMintEvent{
		BaseEvent: NewBaseEvent("nft.mint", "nft123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.ID() == "" {
			t.Error("expected non-empty ID")
		}
		if event.EventType() != "nft.mint" {
			t.Errorf("expected nft.mint, got %s", event.EventType())
		}
		if event.Module() != "nft" {
			t.Errorf("expected nft, got %s", event.Module())
		}
	})

	t.Run("Owner accessor", func(t *testing.T) {
		got, err := event.Owner()
		if err != nil {
			t.Fatalf("Owner() error = %v", err)
		}
		if string(got) != "owner_pubkey" {
			t.Errorf("Owner() = %s, want owner_pubkey", string(got))
		}
	})

	t.Run("Metadata accessor", func(t *testing.T) {
		got, err := event.Metadata()
		if err != nil {
			t.Fatalf("Metadata() error = %v", err)
		}
		if got != "ipfs://QmExample" {
			t.Errorf("Metadata() = %s, want ipfs://QmExample", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &NFTMintEvent{
			BaseEvent: NewBaseEvent("nft.mint", "nft123", []byte("invalid")),
		}
		_, err := badEvent.Owner()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		badPayload, _ := json.Marshal(map[string]interface{}{
			"owner":    "!!!invalid!!!",
			"metadata": "test",
		})
		badEvent := &NFTMintEvent{
			BaseEvent: NewBaseEvent("nft.mint", "nft123", badPayload),
		}
		_, err := badEvent.Owner()
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestNFTTransferEvent(t *testing.T) {
	from := base64.StdEncoding.EncodeToString([]byte("from_pubkey"))
	to := base64.StdEncoding.EncodeToString([]byte("to_pubkey"))

	payload, err := json.Marshal(nftTransferPayload{
		From: from,
		To:   to,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &NFTTransferEvent{
		BaseEvent: NewBaseEvent("nft.transfer", "nft123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "nft.transfer" {
			t.Errorf("expected nft.transfer, got %s", event.EventType())
		}
		if event.Module() != "nft" {
			t.Errorf("expected nft, got %s", event.Module())
		}
	})

	t.Run("From accessor", func(t *testing.T) {
		got, err := event.From()
		if err != nil {
			t.Fatalf("From() error = %v", err)
		}
		if string(got) != "from_pubkey" {
			t.Errorf("From() = %s, want from_pubkey", string(got))
		}
	})

	t.Run("To accessor", func(t *testing.T) {
		got, err := event.To()
		if err != nil {
			t.Fatalf("To() error = %v", err)
		}
		if string(got) != "to_pubkey" {
			t.Errorf("To() = %s, want to_pubkey", string(got))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &NFTTransferEvent{
			BaseEvent: NewBaseEvent("nft.transfer", "nft123", []byte("invalid")),
		}
		_, err := badEvent.From()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestNFTBurnEvent(t *testing.T) {
	from := base64.StdEncoding.EncodeToString([]byte("burner_pubkey"))

	payload, err := json.Marshal(nftBurnPayload{
		From: from,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &NFTBurnEvent{
		BaseEvent: NewBaseEvent("nft.burn", "nft123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.EventType() != "nft.burn" {
			t.Errorf("expected nft.burn, got %s", event.EventType())
		}
	})

	t.Run("From accessor", func(t *testing.T) {
		got, err := event.From()
		if err != nil {
			t.Fatalf("From() error = %v", err)
		}
		if string(got) != "burner_pubkey" {
			t.Errorf("From() = %s, want burner_pubkey", string(got))
		}
	})
}
