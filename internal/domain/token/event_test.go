package token

import (
	"encoding/json"
	"testing"
	"time"

	"crypto/ed25519"
)

func TestNewTransferEvent(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)
	signature := Signature(make([]byte, ed25519.SignatureSize))

	event := NewTransferEvent("TEST", from, to, amount, 1, signature)

	if event.ID() == "" {
		t.Error("TransferEvent.ID() should not be empty")
	}
	if event.TokenID() != "TEST" {
		t.Errorf("TransferEvent.TokenID() = %s, want TEST", event.TokenID())
	}
	if event.From() == nil {
		t.Error("TransferEvent.From() should not be nil")
	}
	if event.To() == nil {
		t.Error("TransferEvent.To() should not be nil")
	}
	if event.Amount() == nil {
		t.Error("TransferEvent.Amount() should not be nil")
	}
	if event.Nonce() != 1 {
		t.Errorf("TransferEvent.Nonce() = %d, want 1", event.Nonce())
	}
	if event.BlockHeight() != 0 {
		t.Errorf("TransferEvent.BlockHeight() = %d, want 0", event.BlockHeight())
	}
	if event.Timestamp().IsZero() {
		t.Error("TransferEvent.Timestamp() should not be zero")
	}
}

func TestNewTransferEventFromData(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)
	signature := Signature(make([]byte, ed25519.SignatureSize))
	timestamp := time.Now()

	event := NewTransferEventFromData("custom-id", "TEST", from, to, amount, 1, signature, 42, timestamp)

	if event.ID() != "custom-id" {
		t.Errorf("TransferEvent.ID() = %s, want custom-id", event.ID())
	}
	if event.BlockHeight() != 42 {
		t.Errorf("TransferEvent.BlockHeight() = %d, want 42", event.BlockHeight())
	}
	if event.Timestamp() != timestamp {
		t.Errorf("TransferEvent.Timestamp() = %v, want %v", event.Timestamp(), timestamp)
	}
}

func TestTransferEvent_SetBlockHeight(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewTransferEvent("TEST", from, to, amount, 1, nil)
	event.SetBlockHeight(100)

	if event.BlockHeight() != 100 {
		t.Errorf("TransferEvent.BlockHeight() = %d, want 100", event.BlockHeight())
	}
}

func TestTransferEvent_EventMethods(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewTransferEvent("TEST", from, to, amount, 1, nil)

	if event.EventType() != "token.transfer" {
		t.Errorf("EventType() = %s, want token.transfer", event.EventType())
	}
	if event.Module() != "token" {
		t.Errorf("Module() = %s, want token", event.Module())
	}
	if event.AggregateID() != "TEST" {
		t.Errorf("AggregateID() = %s, want TEST", event.AggregateID())
	}

	payload := event.Payload()
	if len(payload) == 0 {
		t.Error("Payload() should not be empty")
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		t.Errorf("Payload() is not valid JSON: %v", err)
	}
}

func TestNewMintEvent(t *testing.T) {
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewMintEvent("TEST", to, amount)

	if event.ID() == "" {
		t.Error("MintEvent.ID() should not be empty")
	}
	if event.TokenID() != "TEST" {
		t.Errorf("MintEvent.TokenID() = %s, want TEST", event.TokenID())
	}
	if event.To() == nil {
		t.Error("MintEvent.To() should not be nil")
	}
	if event.Amount() == nil {
		t.Error("MintEvent.Amount() should not be nil")
	}
	if event.BlockHeight() != 0 {
		t.Errorf("MintEvent.BlockHeight() = %d, want 0", event.BlockHeight())
	}
	if event.Timestamp().IsZero() {
		t.Error("MintEvent.Timestamp() should not be zero")
	}
}

func TestNewMintEventFromData(t *testing.T) {
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)
	timestamp := time.Now()

	event := NewMintEventFromData("custom-id", "TEST", to, amount, 42, timestamp)

	if event.ID() != "custom-id" {
		t.Errorf("MintEvent.ID() = %s, want custom-id", event.ID())
	}
	if event.BlockHeight() != 42 {
		t.Errorf("MintEvent.BlockHeight() = %d, want 42", event.BlockHeight())
	}
}

func TestMintEvent_SetBlockHeight(t *testing.T) {
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewMintEvent("TEST", to, amount)
	event.SetBlockHeight(100)

	if event.BlockHeight() != 100 {
		t.Errorf("MintEvent.BlockHeight() = %d, want 100", event.BlockHeight())
	}
}

func TestMintEvent_EventMethods(t *testing.T) {
	to := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewMintEvent("TEST", to, amount)

	if event.EventType() != "token.mint" {
		t.Errorf("EventType() = %s, want token.mint", event.EventType())
	}
	if event.Module() != "token" {
		t.Errorf("Module() = %s, want token", event.Module())
	}
	if event.AggregateID() != "TEST" {
		t.Errorf("AggregateID() = %s, want TEST", event.AggregateID())
	}

	payload := event.Payload()
	if len(payload) == 0 {
		t.Error("Payload() should not be empty")
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		t.Errorf("Payload() is not valid JSON: %v", err)
	}
}

func TestNewBurnEvent(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewBurnEvent("TEST", from, amount)

	if event.ID() == "" {
		t.Error("BurnEvent.ID() should not be empty")
	}
	if event.TokenID() != "TEST" {
		t.Errorf("BurnEvent.TokenID() = %s, want TEST", event.TokenID())
	}
	if event.From() == nil {
		t.Error("BurnEvent.From() should not be nil")
	}
	if event.Amount() == nil {
		t.Error("BurnEvent.Amount() should not be nil")
	}
	if event.BlockHeight() != 0 {
		t.Errorf("BurnEvent.BlockHeight() = %d, want 0", event.BlockHeight())
	}
	if event.Timestamp().IsZero() {
		t.Error("BurnEvent.Timestamp() should not be zero")
	}
}

func TestNewBurnEventFromData(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	amount := NewAmount(100)
	timestamp := time.Now()

	event := NewBurnEventFromData("custom-id", "TEST", from, amount, 42, timestamp)

	if event.ID() != "custom-id" {
		t.Errorf("BurnEvent.ID() = %s, want custom-id", event.ID())
	}
	if event.BlockHeight() != 42 {
		t.Errorf("BurnEvent.BlockHeight() = %d, want 42", event.BlockHeight())
	}
}

func TestBurnEvent_SetBlockHeight(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewBurnEvent("TEST", from, amount)
	event.SetBlockHeight(100)

	if event.BlockHeight() != 100 {
		t.Errorf("BurnEvent.BlockHeight() = %d, want 100", event.BlockHeight())
	}
}

func TestBurnEvent_EventMethods(t *testing.T) {
	from := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewBurnEvent("TEST", from, amount)

	if event.EventType() != "token.burn" {
		t.Errorf("EventType() = %s, want token.burn", event.EventType())
	}
	if event.Module() != "token" {
		t.Errorf("Module() = %s, want token", event.Module())
	}
	if event.AggregateID() != "TEST" {
		t.Errorf("AggregateID() = %s, want TEST", event.AggregateID())
	}

	payload := event.Payload()
	if len(payload) == 0 {
		t.Error("Payload() should not be empty")
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		t.Errorf("Payload() is not valid JSON: %v", err)
	}
}

func TestNewApproveEvent(t *testing.T) {
	owner := PublicKey(make([]byte, 32))
	spender := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewApproveEvent("TEST", owner, spender, amount)

	if event.ID() == "" {
		t.Error("ApproveEvent.ID() should not be empty")
	}
	if event.TokenID() != "TEST" {
		t.Errorf("ApproveEvent.TokenID() = %s, want TEST", event.TokenID())
	}
	if event.Owner() == nil {
		t.Error("ApproveEvent.Owner() should not be nil")
	}
	if event.Spender() == nil {
		t.Error("ApproveEvent.Spender() should not be nil")
	}
	if event.Amount() == nil {
		t.Error("ApproveEvent.Amount() should not be nil")
	}
	if !event.ExpiresAt().IsZero() {
		t.Errorf("ApproveEvent.ExpiresAt() = %v, want zero", event.ExpiresAt())
	}
	if event.Timestamp().IsZero() {
		t.Error("ApproveEvent.Timestamp() should not be zero")
	}
}

func TestApproveEvent_SetExpiresAt(t *testing.T) {
	owner := PublicKey(make([]byte, 32))
	spender := PublicKey(make([]byte, 32))
	amount := NewAmount(100)
	expiry := time.Now().Add(24 * time.Hour)

	event := NewApproveEvent("TEST", owner, spender, amount)
	event.SetExpiresAt(expiry)

	if event.ExpiresAt() != expiry {
		t.Errorf("ApproveEvent.ExpiresAt() = %v, want %v", event.ExpiresAt(), expiry)
	}
}

func TestApproveEvent_EventMethods(t *testing.T) {
	owner := PublicKey(make([]byte, 32))
	spender := PublicKey(make([]byte, 32))
	amount := NewAmount(100)

	event := NewApproveEvent("TEST", owner, spender, amount)

	if event.EventType() != "token.approve" {
		t.Errorf("EventType() = %s, want token.approve", event.EventType())
	}
	if event.Module() != "token" {
		t.Errorf("Module() = %s, want token", event.Module())
	}
	if event.AggregateID() != "TEST" {
		t.Errorf("AggregateID() = %s, want TEST", event.AggregateID())
	}

	payload := event.Payload()
	if len(payload) == 0 {
		t.Error("Payload() should not be empty")
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		t.Errorf("Payload() is not valid JSON: %v", err)
	}
}
