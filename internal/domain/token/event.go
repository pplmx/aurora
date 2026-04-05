package token

import (
	"time"

	"github.com/google/uuid"
)

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

func (e *MintEvent) ID() string             { return e.id }
func (e *MintEvent) TokenID() TokenID       { return e.tokenID }
func (e *MintEvent) To() PublicKey          { return e.to }
func (e *MintEvent) Amount() *Amount        { return e.amount }
func (e *MintEvent) BlockHeight() int64     { return e.blockHeight }
func (e *MintEvent) SetBlockHeight(h int64) { e.blockHeight = h }
func (e *MintEvent) Timestamp() time.Time   { return e.timestamp }

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

func (e *BurnEvent) ID() string             { return e.id }
func (e *BurnEvent) TokenID() TokenID       { return e.tokenID }
func (e *BurnEvent) From() PublicKey        { return e.from }
func (e *BurnEvent) Amount() *Amount        { return e.amount }
func (e *BurnEvent) BlockHeight() int64     { return e.blockHeight }
func (e *BurnEvent) SetBlockHeight(h int64) { e.blockHeight = h }
func (e *BurnEvent) Timestamp() time.Time   { return e.timestamp }

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

func (e *ApproveEvent) ID() string           { return e.id }
func (e *ApproveEvent) TokenID() TokenID     { return e.tokenID }
func (e *ApproveEvent) Owner() PublicKey     { return e.owner }
func (e *ApproveEvent) Spender() PublicKey   { return e.spender }
func (e *ApproveEvent) Amount() *Amount      { return e.amount }
func (e *ApproveEvent) Timestamp() time.Time { return e.timestamp }
