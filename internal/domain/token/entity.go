package token

import (
	"math/big"
	"time"
)

type TokenID string

type PublicKey []byte

type Signature []byte

type Amount struct {
	*big.Int
}

func NewAmount(value int64) *Amount {
	return &Amount{big.NewInt(value)}
}

func NewAmountFromString(s string) (*Amount, error) {
	if s == "" {
		return nil, ErrInvalidAmount
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, ErrInvalidAmount
	}
	return &Amount{v}, nil
}

func (a *Amount) String() string {
	if a == nil || a.Int == nil {
		return "0"
	}
	return a.Int.String()
}

func (a *Amount) IsPositive() bool {
	return a != nil && a.Int != nil && a.Int.Sign() > 0
}

type Token struct {
	id          TokenID
	name        string
	symbol      string
	totalSupply *Amount
	decimals    int8
	owner       PublicKey
	isMintable  bool
	isBurnable  bool
	createdAt   time.Time
}

func NewToken(id TokenID, name, symbol string, totalSupply *Amount, owner PublicKey) *Token {
	return &Token{
		id:          id,
		name:        name,
		symbol:      symbol,
		totalSupply: totalSupply,
		decimals:    8,
		owner:       owner,
		isMintable:  true,
		isBurnable:  true,
		createdAt:   time.Now(),
	}
}

func (t *Token) ID() TokenID          { return t.id }
func (t *Token) Name() string         { return t.name }
func (t *Token) Symbol() string       { return t.symbol }
func (t *Token) TotalSupply() *Amount { return t.totalSupply }
func (t *Token) Decimals() int8       { return t.decimals }
func (t *Token) Owner() PublicKey     { return t.owner }
func (t *Token) IsMintable() bool     { return t.isMintable }
func (t *Token) IsBurnable() bool     { return t.isBurnable }
func (t *Token) CreatedAt() time.Time { return t.createdAt }

type Approval struct {
	tokenID   TokenID
	owner     PublicKey
	spender   PublicKey
	amount    *Amount
	expiresAt time.Time
}

func NewApproval(tokenID TokenID, owner, spender PublicKey, amount *Amount) *Approval {
	return &Approval{
		tokenID: tokenID,
		owner:   owner,
		spender: spender,
		amount:  amount,
	}
}

func (a *Approval) TokenID() TokenID     { return a.tokenID }
func (a *Approval) Owner() PublicKey     { return a.owner }
func (a *Approval) Spender() PublicKey   { return a.spender }
func (a *Approval) Amount() *Amount      { return a.amount }
func (a *Approval) ExpiresAt() time.Time { return a.expiresAt }
