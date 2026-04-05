package token

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/pplmx/aurora/internal/domain/blockchain"
)

type Service interface {
	CreateToken(req *CreateTokenRequest) (*Token, error)
	GetTokenInfo(tokenID TokenID) (*Token, error)

	Mint(req *MintRequest) (*MintEvent, error)

	Transfer(req *TransferRequest) (*TransferEvent, error)
	TransferFrom(req *TransferFromRequest) (*TransferEvent, error)

	Approve(req *ApproveRequest) (*ApproveEvent, error)
	IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error)
	DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error)

	Burn(req *BurnRequest) (*BurnEvent, error)

	GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error)
	GetAllowance(tokenID TokenID, owner, spender PublicKey) (*Amount, error)
	GetTransferHistory(tokenID TokenID, owner PublicKey, limit int) ([]*TransferEvent, error)
}

type Repository interface {
	SaveToken(token *Token) error
	GetToken(id TokenID) (*Token, error)

	SaveApproval(approval *Approval) error
	GetApproval(tokenID TokenID, owner, spender PublicKey) (*Approval, error)
	GetApprovalsByOwner(tokenID TokenID, owner PublicKey) ([]*Approval, error)

	GetAccountBalance(tokenID TokenID, owner PublicKey) (*Amount, error)
	SetAccountBalance(tokenID TokenID, owner PublicKey, amount *Amount) error
}

type EventStore interface {
	SaveTransferEvent(event *TransferEvent) error
	SaveMintEvent(event *MintEvent) error
	SaveBurnEvent(event *BurnEvent) error
	SaveApproveEvent(event *ApproveEvent) error

	GetTransferEventsByToken(tokenID TokenID) ([]*TransferEvent, error)
	GetTransferEventsByOwner(tokenID TokenID, owner PublicKey) ([]*TransferEvent, error)
	GetMintEventsByToken(tokenID TokenID) ([]*MintEvent, error)
	GetBurnEventsByToken(tokenID TokenID) ([]*BurnEvent, error)

	GetLastNonce(tokenID TokenID, owner PublicKey) (uint64, error)
}

type TokenService struct {
	repo       Repository
	eventStore EventStore
	chain      blockchain.BlockWriter
}

func NewService(repo Repository, eventStore EventStore, chain blockchain.BlockWriter) *TokenService {
	return &TokenService{
		repo:       repo,
		eventStore: eventStore,
		chain:      chain,
	}
}

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply *Amount
	Owner       PublicKey
}

type MintRequest struct {
	TokenID    TokenID
	To         PublicKey
	Amount     *Amount
	PrivateKey []byte
}

type TransferRequest struct {
	TokenID    TokenID
	From       PublicKey
	To         PublicKey
	Amount     *Amount
	PrivateKey []byte
}

type TransferFromRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	To         PublicKey
	Amount     *Amount
	Spender    PublicKey
	SpenderKey []byte
}

type ApproveRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	Spender    PublicKey
	Amount     *Amount
	PrivateKey []byte
}

type AllowanceRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	Spender    PublicKey
	Amount     *Amount
	PrivateKey []byte
}

type BurnRequest struct {
	TokenID    TokenID
	From       PublicKey
	Amount     *Amount
	PrivateKey []byte
}

func (s *TokenService) CreateToken(req *CreateTokenRequest) (*Token, error) {
	if err := ValidateTokenName(req.Name); err != nil {
		return nil, err
	}
	if err := ValidateTokenSymbol(req.Symbol); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.TotalSupply); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}

	token := NewToken(TokenID(req.Symbol), req.Name, req.Symbol, req.TotalSupply, req.Owner)

	if err := s.repo.SaveToken(token); err != nil {
		return nil, err
	}

	if err := s.repo.SetAccountBalance(token.ID(), req.Owner, req.TotalSupply); err != nil {
		return nil, err
	}

	return token, nil
}

func (s *TokenService) GetTokenInfo(tokenID TokenID) (*Token, error) {
	return s.repo.GetToken(tokenID)
}

func (s *TokenService) GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error) {
	return s.repo.GetAccountBalance(tokenID, owner)
}

func (s *TokenService) GetAllowance(tokenID TokenID, owner, spender PublicKey) (*Amount, error) {
	approval, err := s.repo.GetApproval(tokenID, owner, spender)
	if err != nil {
		return nil, err
	}
	if approval == nil {
		return NewAmount(0), nil
	}
	return approval.Amount(), nil
}

func (s *TokenService) Mint(req *MintRequest) (*MintEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if !token.IsMintable() {
		return nil, ErrTokenNotMintable
	}

	if err := ValidatePublicKey(req.To); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	currentBalance, err := s.repo.GetAccountBalance(req.TokenID, req.To)
	if err != nil {
		return nil, err
	}
	newBalance := &Amount{new(big.Int).Add(currentBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.To, newBalance); err != nil {
		return nil, err
	}

	event := NewMintEvent(req.TokenID, req.To, req.Amount)
	if err := s.eventStore.SaveMintEvent(event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) Transfer(req *TransferRequest) (*TransferEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := ValidatePublicKey(req.From); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.To); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	fromBalance, err := s.repo.GetAccountBalance(req.TokenID, req.From)
	if err != nil {
		return nil, err
	}
	if fromBalance.Int.Cmp(req.Amount.Int) < 0 {
		return nil, ErrInsufficientBalance
	}

	nonce, err := s.eventStore.GetLastNonce(req.TokenID, req.From)
	if err != nil {
		return nil, err
	}
	nonce++

	signature := ed25519.Sign(req.PrivateKey, s.signMessage(req.TokenID, req.From, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.From, req.To, req.Amount, nonce, signature)
	if err := s.eventStore.SaveTransferEvent(event); err != nil {
		return nil, err
	}

	fromNewBalance := &Amount{new(big.Int).Sub(fromBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.From, fromNewBalance); err != nil {
		return nil, err
	}

	toBalance, err := s.repo.GetAccountBalance(req.TokenID, req.To)
	if err != nil {
		return nil, err
	}
	toNewBalance := &Amount{new(big.Int).Add(toBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.To, toNewBalance); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) TransferFrom(req *TransferFromRequest) (*TransferEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.To); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	approval, err := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender)
	if err != nil {
		return nil, err
	}
	if approval == nil {
		return nil, ErrInsufficientAllowance
	}
	if approval.Amount().Int.Cmp(req.Amount.Int) < 0 {
		return nil, ErrInsufficientAllowance
	}

	ownerBalance, err := s.repo.GetAccountBalance(req.TokenID, req.Owner)
	if err != nil {
		return nil, err
	}
	if ownerBalance.Int.Cmp(req.Amount.Int) < 0 {
		return nil, ErrInsufficientBalance
	}

	nonce, err := s.eventStore.GetLastNonce(req.TokenID, req.Spender)
	if err != nil {
		return nil, err
	}
	nonce++

	signature := ed25519.Sign(req.SpenderKey, s.signMessage(req.TokenID, req.Owner, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.Owner, req.To, req.Amount, nonce, signature)
	if err := s.eventStore.SaveTransferEvent(event); err != nil {
		return nil, err
	}

	ownerNewBalance := &Amount{new(big.Int).Sub(ownerBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.Owner, ownerNewBalance); err != nil {
		return nil, err
	}

	toBalance, err := s.repo.GetAccountBalance(req.TokenID, req.To)
	if err != nil {
		return nil, err
	}
	toNewBalance := &Amount{new(big.Int).Add(toBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.To, toNewBalance); err != nil {
		return nil, err
	}

	newApprovalAmount := &Amount{new(big.Int).Sub(approval.Amount().Int, req.Amount.Int)}
	newApproval := NewApproval(req.TokenID, req.Owner, req.Spender, newApprovalAmount)
	if err := s.repo.SaveApproval(newApproval); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) Approve(req *ApproveRequest) (*ApproveEvent, error) {
	if _, err := s.repo.GetToken(req.TokenID); err != nil {
		return nil, err
	}

	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	approval := NewApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err := s.repo.SaveApproval(approval); err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err := s.eventStore.SaveApproveEvent(event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	currentApproval, err := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender)
	if err != nil {
		return nil, err
	}

	var currentAmount int64
	if currentApproval != nil {
		currentAmount = currentApproval.Amount().Int.Int64()
	}

	newAmount := &Amount{new(big.Int).Add(big.NewInt(currentAmount), req.Amount.Int)}

	approveReq := &ApproveRequest{
		TokenID: req.TokenID,
		Owner:   req.Owner,
		Spender: req.Spender,
		Amount:  newAmount,
	}

	return s.Approve(approveReq)
}

func (s *TokenService) DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	currentApproval, err := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender)
	if err != nil {
		return nil, err
	}

	var currentAmount int64
	if currentApproval != nil {
		currentAmount = currentApproval.Amount().Int.Int64()
	}

	newAmount := &Amount{new(big.Int).Sub(big.NewInt(currentAmount), req.Amount.Int)}
	if newAmount.Int.Sign() < 0 {
		newAmount = NewAmount(0)
	}

	approveReq := &ApproveRequest{
		TokenID: req.TokenID,
		Owner:   req.Owner,
		Spender: req.Spender,
		Amount:  newAmount,
	}

	return s.Approve(approveReq)
}

func (s *TokenService) Burn(req *BurnRequest) (*BurnEvent, error) {
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if !token.IsBurnable() {
		return nil, ErrTokenNotBurnable
	}

	if err := ValidatePublicKey(req.From); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	fromBalance, err := s.repo.GetAccountBalance(req.TokenID, req.From)
	if err != nil {
		return nil, err
	}
	if fromBalance.Int.Cmp(req.Amount.Int) < 0 {
		return nil, ErrInsufficientBalance
	}

	event := NewBurnEvent(req.TokenID, req.From, req.Amount)
	if err := s.eventStore.SaveBurnEvent(event); err != nil {
		return nil, err
	}

	newBalance := &Amount{new(big.Int).Sub(fromBalance.Int, req.Amount.Int)}
	if err := s.repo.SetAccountBalance(req.TokenID, req.From, newBalance); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) GetTransferHistory(tokenID TokenID, owner PublicKey, limit int) ([]*TransferEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	events, err := s.eventStore.GetTransferEventsByOwner(tokenID, owner)
	if err != nil {
		return nil, err
	}

	if len(events) > limit {
		events = events[:limit]
	}

	return events, nil
}

func (s *TokenService) signMessage(tokenID TokenID, from, to PublicKey, amount *Amount, nonce uint64) []byte {
	msg := fmt.Sprintf("%s|%s|%s|%s|%d",
		tokenID,
		base64.StdEncoding.EncodeToString(from),
		base64.StdEncoding.EncodeToString(to),
		amount.String(),
		nonce,
	)
	hash := sha256.Sum256([]byte(msg))
	return hash[:]
}
