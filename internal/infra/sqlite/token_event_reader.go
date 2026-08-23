package sqlite

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

type TokenEventReader struct {
	store infraevents.EventRepository
}

func NewTokenEventReader(store infraevents.EventRepository) *TokenEventReader {
	return &TokenEventReader{store: store}
}

func (r *TokenEventReader) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey, limit, offset int) ([]*token.TransferEvent, error) {
	// Page over TRANSFER events only (TASK-074, ISS-066). Passing the limit
	// straight through GetByAggregate applies the SQL LIMIT/OFFSET over ALL
	// events in the token aggregate (transfers + mints + burns) before the
	// owner/type filter below, so interleaved mint/burn events would crowd out
	// transfers and the page would under-fill (verified: limit=5 -> 2). By
	// restricting the query to token.transfer first, the SQL page is composed
	// entirely of transfers and pagination is correct. (The owner filter then
	// narrows that transfer page in memory.)
	events, err := r.store.GetByAggregateAndType(string(tokenID), "token.transfer", limit, offset)
	if err != nil {
		return nil, err
	}

	var result []*token.TransferEvent
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	for _, e := range events {
		var payload struct {
			From        string `json:"from"`
			To          string `json:"to"`
			Amount      string `json:"amount"`
			Nonce       uint64 `json:"nonce"`
			Sig         string `json:"sig"`
			BlockHeight int64  `json:"block_height"`
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

		evt := token.NewTransferEventFromData(e.ID(), tokenID, from, to, amount, payload.Nonce, sig, payload.BlockHeight, e.Timestamp())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) GetMintEventsByToken(tokenID token.TokenID) ([]*token.MintEvent, error) {
	events, err := r.store.GetByAggregate(string(tokenID), 0, 0)
	if err != nil {
		return nil, err
	}

	var result []*token.MintEvent

	for _, e := range events {
		if e.EventType() != "token.mint" {
			continue
		}

		var payload struct {
			To          string `json:"to"`
			Amount      string `json:"amount"`
			BlockHeight int64  `json:"block_height"`
		}
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		to, _ := base64.StdEncoding.DecodeString(payload.To)

		amount, err := token.NewAmountFromString(payload.Amount)
		if err != nil {
			continue
		}

		evt := token.NewMintEventFromData(e.ID(), tokenID, to, amount, payload.BlockHeight, e.Timestamp())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) GetBurnEventsByToken(tokenID token.TokenID) ([]*token.BurnEvent, error) {
	events, err := r.store.GetByAggregate(string(tokenID), 0, 0)
	if err != nil {
		return nil, err
	}

	var result []*token.BurnEvent

	for _, e := range events {
		if e.EventType() != "token.burn" {
			continue
		}

		var payload struct {
			From        string `json:"from"`
			Amount      string `json:"amount"`
			BlockHeight int64  `json:"block_height"`
		}
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		from, _ := base64.StdEncoding.DecodeString(payload.From)

		amount, err := token.NewAmountFromString(payload.Amount)
		if err != nil {
			continue
		}

		evt := token.NewBurnEventFromData(e.ID(), tokenID, from, amount, payload.BlockHeight, e.Timestamp())
		result = append(result, evt)
	}

	return result, nil
}

func (r *TokenEventReader) Close() error {
	return nil
}

var _ token.EventReader = (*TokenEventReader)(nil)
