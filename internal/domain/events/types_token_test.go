package events

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestTokenTransferEvent(t *testing.T) {
	from := base64.StdEncoding.EncodeToString([]byte("sender_pubkey"))
	to := base64.StdEncoding.EncodeToString([]byte("receiver_pubkey"))
	sig := base64.StdEncoding.EncodeToString([]byte("test_signature"))

	payload, err := json.Marshal(tokenTransferPayload{
		From:      from,
		To:        to,
		Amount:    "100",
		Nonce:     42,
		Signature: sig,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &TokenTransferEvent{
		BaseEvent: NewBaseEvent("token.transfer", "token123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.ID() == "" {
			t.Error("expected non-empty ID")
		}
		if event.EventType() != "token.transfer" {
			t.Errorf("expected token.transfer, got %s", event.EventType())
		}
		if event.Module() != "token" {
			t.Errorf("expected token, got %s", event.Module())
		}
		if event.AggregateID() != "token123" {
			t.Errorf("expected token123, got %s", event.AggregateID())
		}
	})

	t.Run("From accessor", func(t *testing.T) {
		got, err := event.From()
		if err != nil {
			t.Fatalf("From() error = %v", err)
		}
		if string(got) != "sender_pubkey" {
			t.Errorf("From() = %s, want sender_pubkey", string(got))
		}
	})

	t.Run("To accessor", func(t *testing.T) {
		got, err := event.To()
		if err != nil {
			t.Fatalf("To() error = %v", err)
		}
		if string(got) != "receiver_pubkey" {
			t.Errorf("To() = %s, want receiver_pubkey", string(got))
		}
	})

	t.Run("Amount accessor", func(t *testing.T) {
		got, err := event.Amount()
		if err != nil {
			t.Fatalf("Amount() error = %v", err)
		}
		if got != "100" {
			t.Errorf("Amount() = %s, want 100", got)
		}
	})

	t.Run("Nonce accessor", func(t *testing.T) {
		got, err := event.Nonce()
		if err != nil {
			t.Fatalf("Nonce() error = %v", err)
		}
		if got != 42 {
			t.Errorf("Nonce() = %d, want 42", got)
		}
	})

	t.Run("Signature accessor", func(t *testing.T) {
		got, err := event.Signature()
		if err != nil {
			t.Fatalf("Signature() error = %v", err)
		}
		if string(got) != "test_signature" {
			t.Errorf("Signature() = %s, want test_signature", string(got))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &TokenTransferEvent{
			BaseEvent: NewBaseEvent("token.transfer", "token123", []byte("not json")),
		}
		_, err := badEvent.From()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &TokenTransferEvent{
			BaseEvent: NewBaseEvent("token.transfer", "token123", []byte("not json")),
		}
		_, err := badEvent.From()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		badPayload, _ := json.Marshal(tokenTransferPayload{
			From:      "!!!not-base64!!!",
			To:        to,
			Amount:    "100",
			Nonce:     42,
			Signature: sig,
		})
		badEvent := &TokenTransferEvent{
			BaseEvent: NewBaseEvent("token.transfer", "token123", badPayload),
		}
		_, err := badEvent.From()
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("invalid base64 for Signature", func(t *testing.T) {
		badPayload, _ := json.Marshal(tokenTransferPayload{
			From:      from,
			To:        to,
			Amount:    "100",
			Nonce:     42,
			Signature: "!!!invalid!!!",
		})
		badEvent := &TokenTransferEvent{
			BaseEvent: NewBaseEvent("token.transfer", "token123", badPayload),
		}
		_, err := badEvent.Signature()
		if err == nil {
			t.Error("expected error for invalid base64 in Signature")
		}
	})
}

func TestTokenMintEvent(t *testing.T) {
	to := base64.StdEncoding.EncodeToString([]byte("minter_pubkey"))

	payload, err := json.Marshal(tokenMintPayload{
		To:     to,
		Amount: "500",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &TokenMintEvent{
		BaseEvent: NewBaseEvent("token.mint", "token123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.ID() == "" {
			t.Error("expected non-empty ID")
		}
		if event.EventType() != "token.mint" {
			t.Errorf("expected token.mint, got %s", event.EventType())
		}
	})

	t.Run("To accessor", func(t *testing.T) {
		got, err := event.To()
		if err != nil {
			t.Fatalf("To() error = %v", err)
		}
		if string(got) != "minter_pubkey" {
			t.Errorf("To() = %s, want minter_pubkey", string(got))
		}
	})

	t.Run("Amount accessor", func(t *testing.T) {
		got, err := event.Amount()
		if err != nil {
			t.Fatalf("Amount() error = %v", err)
		}
		if got != "500" {
			t.Errorf("Amount() = %s, want 500", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &TokenMintEvent{
			BaseEvent: NewBaseEvent("token.mint", "token123", []byte("invalid")),
		}
		_, err := badEvent.To()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestTokenBurnEvent(t *testing.T) {
	from := base64.StdEncoding.EncodeToString([]byte("burner_pubkey"))

	payload, err := json.Marshal(tokenBurnPayload{
		From:   from,
		Amount: "250",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &TokenBurnEvent{
		BaseEvent: NewBaseEvent("token.burn", "token123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.ID() == "" {
			t.Error("expected non-empty ID")
		}
		if event.EventType() != "token.burn" {
			t.Errorf("expected token.burn, got %s", event.EventType())
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

	t.Run("Amount accessor", func(t *testing.T) {
		got, err := event.Amount()
		if err != nil {
			t.Fatalf("Amount() error = %v", err)
		}
		if got != "250" {
			t.Errorf("Amount() = %s, want 250", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &TokenBurnEvent{
			BaseEvent: NewBaseEvent("token.burn", "token123", []byte("invalid")),
		}
		_, err := badEvent.From()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestTokenApproveEvent(t *testing.T) {
	owner := base64.StdEncoding.EncodeToString([]byte("owner_pubkey"))
	spender := base64.StdEncoding.EncodeToString([]byte("spender_pubkey"))

	payload, err := json.Marshal(tokenApprovePayload{
		Owner:   owner,
		Spender: spender,
		Amount:  "1000",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	event := &TokenApproveEvent{
		BaseEvent: NewBaseEvent("token.approve", "token123", payload),
	}

	t.Run("Event interface", func(t *testing.T) {
		if event.ID() == "" {
			t.Error("expected non-empty ID")
		}
		if event.EventType() != "token.approve" {
			t.Errorf("expected token.approve, got %s", event.EventType())
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

	t.Run("Spender accessor", func(t *testing.T) {
		got, err := event.Spender()
		if err != nil {
			t.Fatalf("Spender() error = %v", err)
		}
		if string(got) != "spender_pubkey" {
			t.Errorf("Spender() = %s, want spender_pubkey", string(got))
		}
	})

	t.Run("Amount accessor", func(t *testing.T) {
		got, err := event.Amount()
		if err != nil {
			t.Fatalf("Amount() error = %v", err)
		}
		if got != "1000" {
			t.Errorf("Amount() = %s, want 1000", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badEvent := &TokenApproveEvent{
			BaseEvent: NewBaseEvent("token.approve", "token123", []byte("invalid")),
		}
		_, err := badEvent.Owner()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("invalid base64 for Owner", func(t *testing.T) {
		badPayload, _ := json.Marshal(tokenApprovePayload{
			Owner:   "!!!invalid!!!",
			Spender: spender,
			Amount:  "1000",
		})
		badEvent := &TokenApproveEvent{
			BaseEvent: NewBaseEvent("token.approve", "token123", badPayload),
		}
		_, err := badEvent.Owner()
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("invalid base64 for Spender", func(t *testing.T) {
		badPayload, _ := json.Marshal(tokenApprovePayload{
			Owner:   owner,
			Spender: "!!!invalid!!!",
			Amount:  "1000",
		})
		badEvent := &TokenApproveEvent{
			BaseEvent: NewBaseEvent("token.approve", "token123", badPayload),
		}
		_, err := badEvent.Spender()
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}
