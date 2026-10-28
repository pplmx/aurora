package sqlite

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

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
	// Decoding can still skip a corrupt audit row (bad JSON or amount). Scan
	// past those rows until the requested page is full or the stream ends, and
	// keep offset in terms of readable transfers so a short page reliably means
	// there is no readable row behind it.
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	return pageTransferEvents(
		limit,
		offset,
		func(pageLimit, pageOffset int) ([]events.Event, error) {
			return r.store.GetByAggregateAndTypePayloadEither(
				string(tokenID),
				"token.transfer",
				"$.from",
				ownerB64,
				"$.to",
				ownerB64,
				pageLimit,
				pageOffset,
			)
		},
		func(stored []events.Event) []*token.TransferEvent {
			return decodeTransferEvents(tokenID, stored, func(payload transferPayload) bool {
				return payload.From == ownerB64 || payload.To == ownerB64
			})
		},
	)
}

// GetTransferEventsByToken pages over every transfer in a token's audit
// stream. Every scan restricts both aggregate and event type, so interleaved
// mint, approve and burn events cannot consume a history page.
func (r *TokenEventReader) GetTransferEventsByToken(tokenID token.TokenID, limit, offset int) ([]*token.TransferEvent, error) {
	return pageTransferEvents(
		limit,
		offset,
		func(pageLimit, pageOffset int) ([]events.Event, error) {
			return r.store.GetByAggregateAndType(string(tokenID), "token.transfer", pageLimit, pageOffset)
		},
		func(stored []events.Event) []*token.TransferEvent {
			return decodeTransferEvents(tokenID, stored, nil)
		},
	)
}

const transferEventScanBatch = 100

// pageTransferEvents applies limit/offset over decoded transfers rather than
// raw audit rows. The event reader has to decode JSON and amounts to build the
// domain objects, and those rows may be corrupt. If SQL applied the offset
// first, one corrupt row would both shorten a full page and shift the next
// page, making a short page an unreliable end marker.
func pageTransferEvents(
	limit, offset int,
	fetch func(limit, offset int) ([]events.Event, error),
	decode func([]events.Event) []*token.TransferEvent,
) ([]*token.TransferEvent, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if offset > int(^uint(0)>>1)-limit {
		return nil, fmt.Errorf("transfer history page too large")
	}

	wanted := offset + limit
	batch := max(wanted, transferEventScanBatch)
	result := make([]*token.TransferEvent, 0, wanted)
	rawOffset := 0

	for {
		stored, err := fetch(batch, rawOffset)
		if err != nil {
			return nil, err
		}
		result = append(result, decode(stored)...)
		rawOffset += len(stored)
		if len(result) >= wanted || len(stored) < batch {
			break
		}
	}

	if offset >= len(result) {
		return nil, nil
	}
	end := wanted
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
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
