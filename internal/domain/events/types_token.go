package events

import (
	"encoding/json"
	"fmt"
)

type TokenTransferEvent struct {
	*BaseEvent
}

type tokenTransferPayload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
}

func (e *TokenTransferEvent) From() ([]byte, error) {
	return base64DecodeField(e.Payload(), "from")
}

func (e *TokenTransferEvent) To() ([]byte, error) {
	return base64DecodeField(e.Payload(), "to")
}

func (e *TokenTransferEvent) Amount() (string, error) {
	var p tokenTransferPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Amount, nil
}

func (e *TokenTransferEvent) Nonce() (uint64, error) {
	var p tokenTransferPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Nonce, nil
}

func (e *TokenTransferEvent) Signature() ([]byte, error) {
	return base64DecodeField(e.Payload(), "signature")
}

type TokenMintEvent struct {
	*BaseEvent
}

type tokenMintPayload struct {
	To     string `json:"to"`
	Amount string `json:"amount"`
}

func (e *TokenMintEvent) To() ([]byte, error) {
	return base64DecodeField(e.Payload(), "to")
}

func (e *TokenMintEvent) Amount() (string, error) {
	var p tokenMintPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Amount, nil
}

type TokenBurnEvent struct {
	*BaseEvent
}

type tokenBurnPayload struct {
	From   string `json:"from"`
	Amount string `json:"amount"`
}

func (e *TokenBurnEvent) From() ([]byte, error) {
	return base64DecodeField(e.Payload(), "from")
}

func (e *TokenBurnEvent) Amount() (string, error) {
	var p tokenBurnPayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Amount, nil
}

type TokenApproveEvent struct {
	*BaseEvent
}

type tokenApprovePayload struct {
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
	Amount  string `json:"amount"`
}

func (e *TokenApproveEvent) Owner() ([]byte, error) {
	return base64DecodeField(e.Payload(), "owner")
}

func (e *TokenApproveEvent) Spender() ([]byte, error) {
	return base64DecodeField(e.Payload(), "spender")
}

func (e *TokenApproveEvent) Amount() (string, error) {
	var p tokenApprovePayload
	if err := json.Unmarshal(e.Payload(), &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return p.Amount, nil
}
