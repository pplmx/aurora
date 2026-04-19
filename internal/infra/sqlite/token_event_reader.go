package sqlite

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

type TokenEventReader struct {
	store infraevents.EventRepository
}

func NewTokenEventReader(store infraevents.EventRepository) *TokenEventReader {
	return &TokenEventReader{store: store}
}

func (r *TokenEventReader) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.TransferEvent, error) {
	events, err := r.store.GetByAggregate(string(tokenID))
	if err != nil {
		return nil, err
	}

	var result []*token.TransferEvent
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	for _, e := range events {
		if e.EventType() != "token.transfer" {
			continue
		}

		var payload struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Amount string `json:"amount"`
			Nonce  uint64 `json:"nonce"`
			Sig    string `json:"sig"`
		}
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		if payload.From != ownerB64 {
			continue
		}

		from, _ := base64.StdEncoding.DecodeString(payload.From)
		to, _ := base64.StdEncoding.DecodeString(payload.To)
		sig, _ := base64.StdEncoding.DecodeString(payload.Sig)

		amount, err := token.NewAmountFromString(payload.Amount)
		if err != nil {
			continue
		}

		evt := token.NewTransferEventFromData(e.ID(), tokenID, from, to, amount, payload.Nonce, sig, 0, time.Now())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) GetMintEventsByToken(tokenID token.TokenID) ([]*token.MintEvent, error) {
	events, err := r.store.GetByAggregate(string(tokenID))
	if err != nil {
		return nil, err
	}

	var result []*token.MintEvent

	for _, e := range events {
		if e.EventType() != "token.mint" {
			continue
		}

		var payload struct {
			To     string `json:"to"`
			Amount string `json:"amount"`
		}
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		to, _ := base64.StdEncoding.DecodeString(payload.To)

		amount, err := token.NewAmountFromString(payload.Amount)
		if err != nil {
			continue
		}

		evt := token.NewMintEventFromData(e.ID(), tokenID, to, amount, 0, time.Now())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) GetBurnEventsByToken(tokenID token.TokenID) ([]*token.BurnEvent, error) {
	events, err := r.store.GetByAggregate(string(tokenID))
	if err != nil {
		return nil, err
	}

	var result []*token.BurnEvent

	for _, e := range events {
		if e.EventType() != "token.burn" {
			continue
		}

		var payload struct {
			From   string `json:"from"`
			Amount string `json:"amount"`
		}
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		from, _ := base64.StdEncoding.DecodeString(payload.From)

		amount, err := token.NewAmountFromString(payload.Amount)
		if err != nil {
			continue
		}

		evt := token.NewBurnEventFromData(e.ID(), tokenID, from, amount, 0, time.Now())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) Close() error {
	return nil
}

var _ token.EventReader = (*TokenEventReader)(nil)
