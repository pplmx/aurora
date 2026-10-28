package sqlite

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pplmx/aurora/internal/domain/events"
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
	// Page over the OWNER's transfer events at the SQL layer (TASK-093,
	// ISS-086) on top of the type-restricted pagination (TASK-074, ISS-066).
	// LIMIT/OFFSET must count only transfers for this owner; filtering the
	// requested owner in memory after paging over all of the token's
	// transfers under-fills each page on a multi-owner token and steps offset
	// through the whole token stream instead of the owner's. json_extract is
	// evaluated in SQL (payload is stored as BLOB, cast to TEXT), so tx.from
	// and no other owner can crowd a page (verified: limit=10 -> 5 pre-fix).
	// The Either predicate ALSO matches $.to == owner, so transfers the account
	// RECEIVED show up next to the ones it sent instead of a one-sided history
	// (ISS-162); the post-scan re-check below keeps the memory path honest.
	//
	// That re-check can shrink a page: a stored event whose JSON does not
	// decode, or whose amount is unreadable, is skipped after the query, so a
	// page of 3 can come back with 2 rows while a third matching row exists.
	// Measured with one malformed transfer among three.
	//
	// CONTRACT: a short page does not mean the end of the data. A client paging
	// through this MUST advance by the offset it asked for and stop on an empty
	// page; treating `len(page) < limit` as the last page silently skips the
	// remainder. The CLI and the REST handler both read one page per request,
	// so neither trips over it today.
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	events, err := r.store.GetByAggregateAndTypePayloadEither(string(tokenID), "token.transfer", "$.from", ownerB64, "$.to", ownerB64, limit, offset)
	if err != nil {
		return nil, err
	}

	return decodeTransferEvents(tokenID, events, func(payload transferPayload) bool {
		return payload.From == ownerB64 || payload.To == ownerB64
	}), nil
}

// GetTransferEventsByToken pages over every transfer in a token's audit
// stream. The query restricts both aggregate and event type before applying
// LIMIT/OFFSET, so interleaved mint, approve and burn events cannot consume a
// history page.
func (r *TokenEventReader) GetTransferEventsByToken(tokenID token.TokenID, limit, offset int) ([]*token.TransferEvent, error) {
	stored, err := r.store.GetByAggregateAndType(string(tokenID), "token.transfer", limit, offset)
	if err != nil {
		return nil, err
	}
	return decodeTransferEvents(tokenID, stored, nil), nil
}

type transferPayload struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Nonce       uint64 `json:"nonce"`
	Sig         string `json:"sig"`
	BlockHeight int64  `json:"block_height"`
}

// decodeTransferEvents maps stored audit payloads into domain events. An
// optional keep predicate applies the owner-side post-scan check that protects
// the memory path even if the SQL predicate and payload disagree.
func decodeTransferEvents(tokenID token.TokenID, stored []events.Event, keep func(transferPayload) bool) []*token.TransferEvent {
	var result []*token.TransferEvent

	for _, e := range stored {
		var payload transferPayload
		if err := json.Unmarshal(e.Payload(), &payload); err != nil {
			continue
		}

		if keep != nil && !keep(payload) {
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

	return result
}

func (r *TokenEventReader) Close() error {
	return nil
}

var _ token.EventReader = (*TokenEventReader)(nil)
