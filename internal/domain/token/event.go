package token

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TransferEvent represents a token transfer operation.
type TransferEvent struct {
	id          string
	tokenID     TokenID
	from        PublicKey
	to          PublicKey
	amount      *Amount
	nonce       uint64
	signature   Signature
	blockHeight int64
	timestamp   time.Time
}

func NewTransferEvent(tokenID TokenID, from, to PublicKey, amount *Amount, nonce uint64, signature Signature) *TransferEvent {
	return &TransferEvent{
		id:        uuid.New().String(),
		tokenID:   tokenID,
		from:      from,
		to:        to,
		amount:    amount,
		nonce:     nonce,
		signature: signature,
		timestamp: time.Now(),
	}
}

func NewTransferEventFromData(id string, tokenID TokenID, from, to PublicKey, amount *Amount, nonce uint64, signature Signature, blockHeight int64, timestamp time.Time) *TransferEvent {
	return &TransferEvent{
		id:          id,
		tokenID:     tokenID,
		from:        from,
		to:          to,
		amount:      amount,
		nonce:       nonce,
		signature:   signature,
		blockHeight: blockHeight,
		timestamp:   timestamp,
	}
}

func (e *TransferEvent) ID() string             { return e.id }
func (e *TransferEvent) TokenID() TokenID       { return e.tokenID }
func (e *TransferEvent) From() PublicKey        { return e.from }
func (e *TransferEvent) To() PublicKey          { return e.to }
func (e *TransferEvent) Amount() *Amount        { return e.amount }
func (e *TransferEvent) Nonce() uint64          { return e.nonce }
func (e *TransferEvent) Signature() Signature   { return e.signature }
func (e *TransferEvent) BlockHeight() int64     { return e.blockHeight }
func (e *TransferEvent) SetBlockHeight(h int64) { e.blockHeight = h }
func (e *TransferEvent) Timestamp() time.Time   { return e.timestamp }

func (e *TransferEvent) EventType() string   { return "token.transfer" }
func (e *TransferEvent) Module() string      { return "token" }
func (e *TransferEvent) AggregateID() string { return string(e.tokenID) }
func (e *TransferEvent) Payload() []byte {
	payload := map[string]interface{}{
		"from":   base64.StdEncoding.EncodeToString(e.from),
		"to":     base64.StdEncoding.EncodeToString(e.to),
		"amount": e.amount.String(),
		"nonce":  e.nonce,
		"sig":    base64.StdEncoding.EncodeToString(e.signature),
	}
	data, _ := json.Marshal(payload)
	return data
}

// MintEvent represents a token minting operation.
type MintEvent struct {
	id          string
	tokenID     TokenID
	to          PublicKey
	amount      *Amount
	blockHeight int64
	timestamp   time.Time
}

func NewMintEvent(tokenID TokenID, to PublicKey, amount *Amount) *MintEvent {
	return &MintEvent{
		id:        uuid.New().String(),
		tokenID:   tokenID,
		to:        to,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func NewMintEventFromData(id string, tokenID TokenID, to PublicKey, amount *Amount, blockHeight int64, timestamp time.Time) *MintEvent {
	return &MintEvent{
		id:          id,
		tokenID:     tokenID,
		to:          to,
		amount:      amount,
		blockHeight: blockHeight,
		timestamp:   timestamp,
	}
}

func (e *MintEvent) ID() string             { return e.id }
func (e *MintEvent) TokenID() TokenID       { return e.tokenID }
func (e *MintEvent) To() PublicKey          { return e.to }
func (e *MintEvent) Amount() *Amount        { return e.amount }
func (e *MintEvent) BlockHeight() int64     { return e.blockHeight }
func (e *MintEvent) SetBlockHeight(h int64) { e.blockHeight = h }
func (e *MintEvent) Timestamp() time.Time   { return e.timestamp }

func (e *MintEvent) EventType() string   { return "token.mint" }
func (e *MintEvent) Module() string      { return "token" }
func (e *MintEvent) AggregateID() string { return string(e.tokenID) }
func (e *MintEvent) Payload() []byte {
	payload := map[string]interface{}{
		"to":     base64.StdEncoding.EncodeToString(e.to),
		"amount": e.amount.String(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// BurnEvent represents a token burning operation.
type BurnEvent struct {
	id          string
	tokenID     TokenID
	from        PublicKey
	amount      *Amount
	blockHeight int64
	timestamp   time.Time
}

func NewBurnEvent(tokenID TokenID, from PublicKey, amount *Amount) *BurnEvent {
	return &BurnEvent{
		id:        uuid.New().String(),
		tokenID:   tokenID,
		from:      from,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func NewBurnEventFromData(id string, tokenID TokenID, from PublicKey, amount *Amount, blockHeight int64, timestamp time.Time) *BurnEvent {
	return &BurnEvent{
		id:          id,
		tokenID:     tokenID,
		from:        from,
		amount:      amount,
		blockHeight: blockHeight,
		timestamp:   timestamp,
	}
}

func (e *BurnEvent) ID() string             { return e.id }
func (e *BurnEvent) TokenID() TokenID       { return e.tokenID }
func (e *BurnEvent) From() PublicKey        { return e.from }
func (e *BurnEvent) Amount() *Amount        { return e.amount }
func (e *BurnEvent) BlockHeight() int64     { return e.blockHeight }
func (e *BurnEvent) SetBlockHeight(h int64) { e.blockHeight = h }
func (e *BurnEvent) Timestamp() time.Time   { return e.timestamp }

func (e *BurnEvent) EventType() string   { return "token.burn" }
func (e *BurnEvent) Module() string      { return "token" }
func (e *BurnEvent) AggregateID() string { return string(e.tokenID) }
func (e *BurnEvent) Payload() []byte {
	payload := map[string]interface{}{
		"from":   base64.StdEncoding.EncodeToString(e.from),
		"amount": e.amount.String(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// ApproveEvent represents a token approval operation.
type ApproveEvent struct {
	id        string
	tokenID   TokenID
	owner     PublicKey
	spender   PublicKey
	amount    *Amount
	expiresAt time.Time
	timestamp time.Time
}

func NewApproveEvent(tokenID TokenID, owner, spender PublicKey, amount *Amount) *ApproveEvent {
	return &ApproveEvent{
		id:        uuid.New().String(),
		tokenID:   tokenID,
		owner:     owner,
		spender:   spender,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func (e *ApproveEvent) ID() string               { return e.id }
func (e *ApproveEvent) TokenID() TokenID         { return e.tokenID }
func (e *ApproveEvent) Owner() PublicKey         { return e.owner }
func (e *ApproveEvent) Spender() PublicKey       { return e.spender }
func (e *ApproveEvent) Amount() *Amount          { return e.amount }
func (e *ApproveEvent) ExpiresAt() time.Time     { return e.expiresAt }
func (e *ApproveEvent) SetExpiresAt(t time.Time) { e.expiresAt = t }
func (e *ApproveEvent) Timestamp() time.Time     { return e.timestamp }

func (e *ApproveEvent) EventType() string   { return "token.approve" }
func (e *ApproveEvent) Module() string      { return "token" }
func (e *ApproveEvent) AggregateID() string { return string(e.tokenID) }
func (e *ApproveEvent) Payload() []byte {
	payload := map[string]interface{}{
		"owner":   base64.StdEncoding.EncodeToString(e.owner),
		"spender": base64.StdEncoding.EncodeToString(e.spender),
		"amount":  e.amount.String(),
	}
	data, _ := json.Marshal(payload)
	return data
}
