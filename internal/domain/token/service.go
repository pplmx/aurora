package token

import (
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

// defaultHistoryLimit is the default maximum number of transfer events to return
// when querying transaction history.
const defaultHistoryLimit = 50

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
	GetTransferHistory(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error)
}

type TransactionManager interface {
	WithTransaction(fn func(tx *sql.Tx) error) error
}

type Repository interface {
	SaveToken(token *Token) error
	GetToken(id TokenID) (*Token, error)

	SaveApproval(approval *Approval) error
	GetApproval(tokenID TokenID, owner, spender PublicKey) (*Approval, error)
	GetApprovalsByOwner(tokenID TokenID, owner PublicKey) ([]*Approval, error)

	// TryDeductApproval atomically subtracts amount from the
	// allowance (tokenID, owner, spender) and returns the new
	// allowance amount. Returns ErrInsufficientAllowance if the
	// current allowance is less than amount. This is the atomic
	// primitive that closes the TOCTOU window in TransferFrom.
	TryDeductApproval(tokenID TokenID, owner, spender PublicKey, amount *Amount) (*Amount, error)

	// TryAdjustApproval atomically applies a signed delta to the
	// allowance (tokenID, owner, spender), creating the allowance
	// row if it does not yet exist. Negative delta that would push
	// the allowance below zero is clamped to zero (DecreaseAllowance
	// semantics). Returns the new allowance amount.
	//
	// This is the atomic primitive that closes the TOCTOU window
	// in IncreaseAllowance / DecreaseAllowance (the read-modify-
	// write path silently lost concurrent increments under the
	// pre-fix implementation).
	TryAdjustApproval(tokenID TokenID, owner, spender PublicKey, delta *Amount) (*Amount, error)

	GetAccountBalance(tokenID TokenID, owner PublicKey) (*Amount, error)
	SetAccountBalance(tokenID TokenID, owner PublicKey, amount *Amount) error

	// TrySubtractBalance atomically subtracts amount from (tokenID, owner).
	// Returns the new balance, or ErrInsufficientBalance if the
	// account's current balance is less than amount. This is the
	// atomic primitive that closes the TOCTOU window in Transfer.
	TrySubtractBalance(tokenID TokenID, owner PublicKey, amount *Amount) (*Amount, error)

	// TryAddBalance atomically adds amount to (tokenID, owner),
	// creating the account row if it doesn't exist. Returns the new
	// balance. This is the atomic primitive used by Mint and the
	// credit side of Transfer.
	TryAddBalance(tokenID TokenID, owner PublicKey, amount *Amount) (*Amount, error)

	// TryAddToSupply atomically adds amount to the token's
	// total_supply. Returns the new total supply.
	//
	// This is the atomic primitive that closes the TOCTOU window
	// in Mint: the pre-fix flow did GetToken → token.AddToSupply
	// (in-memory increment) → SaveToken (full-row write). Two
	// concurrent mints both read the same total_supply, both
	// added their amount in memory, and the last SaveToken
	// clobbered the other mint's increment — silently producing
	// less total_supply than the sum of all mints.
	TryAddToSupply(tokenID TokenID, amount *Amount) (*Amount, error)
}

type TransactableRepository interface {
	Repository
	WithTx(tx *sql.Tx) TransactableRepository
}

type EventReader interface {
	GetTransferEventsByOwner(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error)
	GetMintEventsByToken(tokenID TokenID) ([]*MintEvent, error)
	GetBurnEventsByToken(tokenID TokenID) ([]*BurnEvent, error)
}

type TokenService struct {
	repo        Repository
	txManager   TransactionManager
	eventBus    infraevents.EventBus
	eventReader EventReader
	replay      infraevents.ReplayProtection
	chain       blockchain.BlockWriter
}

func NewService(repo Repository, txManager TransactionManager, eventBus infraevents.EventBus, eventReader EventReader, replay infraevents.ReplayProtection, chain blockchain.BlockWriter) *TokenService {
	return &TokenService{
		repo:        repo,
		txManager:   txManager,
		eventBus:    eventBus,
		eventReader: eventReader,
		replay:      replay,
		chain:       chain,
	}
}

type noOpTxManager struct{}

func (noOpTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

func NewServiceWithoutTx(repo Repository, eventBus infraevents.EventBus, eventReader EventReader, replay infraevents.ReplayProtection, chain blockchain.BlockWriter) *TokenService {
	return NewService(repo, noOpTxManager{}, eventBus, eventReader, replay, chain)
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

	event := NewMintEvent(req.TokenID, req.To, req.Amount)

	data := fmt.Sprintf("mint|%s|%s", req.TokenID, req.To)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		// Atomic supply increment: closes the TOCTOU window
		// where two concurrent mints both read total_supply,
		// both added their amount in memory via AddToSupply,
		// and the last SaveToken clobbered the other mint's
		// increment — silently producing less total_supply
		// than the sum of all mints.
		if _, err := s.repo.TryAddToSupply(req.TokenID, req.Amount); err != nil {
			return err
		}

		if err := s.eventBus.Publish(event); err != nil {
			return err
		}

		// Atomic add: closes the race where two concurrent Mints
		// to the same account both read currentBalance, compute
		// currentBalance + amount, and one overwrites the other.
		if _, err := s.repo.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
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
	if fromBalance.Cmp(req.Amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.From)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(req.PrivateKey, s.signMessage(req.TokenID, req.From, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.From, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transfer|%s|%s|%s", req.TokenID, req.From, req.To)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	var transferErr error
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		if err := s.replay.SaveNonce(string(req.TokenID), req.From, nonce); err != nil {
			return err
		}

		if err := s.eventBus.Publish(event); err != nil {
			return err
		}

		// Atomic subtract: closes the TOCTOU race where two
		// concurrent transfers both read fromBalance, both pass
		// the check, and both write back (fromBalance - amount).
		if _, err := s.repo.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			transferErr = err
			return err
		}

		// Atomic add: closes the symmetric race on the credit
		// side (two concurrent transfers to the same recipient
		// could both read toBalance and both write back
		// toBalance + amount, losing one transfer's credit).
		if _, err := s.repo.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			transferErr = err
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	if transferErr != nil {
		return nil, transferErr
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
	if approval.Amount().Cmp(req.Amount) < 0 {
		return nil, ErrInsufficientAllowance
	}

	ownerBalance, err := s.repo.GetAccountBalance(req.TokenID, req.Owner)
	if err != nil {
		return nil, err
	}
	if ownerBalance.Cmp(req.Amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.Spender)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(req.SpenderKey, s.signMessage(req.TokenID, req.Owner, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.Owner, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transferfrom|%s|%s|%s", req.TokenID, req.Owner, req.To)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	var transferErr error
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		if err := s.replay.SaveNonce(string(req.TokenID), req.Spender, nonce); err != nil {
			return err
		}

		if err := s.eventBus.Publish(event); err != nil {
			return err
		}

		// Atomic allowance deduction: closes the TOCTOU race
		// where two concurrent TransferFroms both read
		// approval.Amount(), both pass the check, and both write
		// back approval - amount, allowing double-spend of the
		// allowance.
		newApprovalAmount, err := s.repo.TryDeductApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
		if err != nil {
			transferErr = err
			return err
		}

		// Atomic balance subtract (owner) and add (recipient).
		if _, err := s.repo.TrySubtractBalance(req.TokenID, req.Owner, req.Amount); err != nil {
			transferErr = err
			// Best-effort: restore the allowance so the spender
			// can retry. The newApprovalAmount is the post-deduct
			// value, so adding req.Amount back gives us the
			// pre-deduct value.
			if comp, getErr := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender); getErr == nil && comp != nil {
				restored := &Amount{Int: new(big.Int).Add(comp.Amount().Int, req.Amount.Int)}
				_ = s.repo.SaveApproval(NewApproval(req.TokenID, req.Owner, req.Spender, restored))
			}
			return transferErr
		}
		if _, err := s.repo.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			transferErr = err
			// Best-effort: restore both the owner's balance and
			// the allowance. If the rollback fails, surface both
			// errors so the operator can reconcile manually.
			if _, compErr := s.repo.TryAddBalance(req.TokenID, req.Owner, req.Amount); compErr != nil {
				transferErr = fmt.Errorf("transferfrom add failed (%v) and balance compensation failed (%v)", err, compErr)
			}
			if comp, getErr := s.repo.GetApproval(req.TokenID, req.Owner, req.Spender); getErr == nil && comp != nil {
				restored := &Amount{Int: new(big.Int).Add(comp.Amount().Int, req.Amount.Int)}
				if saveErr := s.repo.SaveApproval(NewApproval(req.TokenID, req.Owner, req.Spender, restored)); saveErr != nil {
					transferErr = fmt.Errorf("transferfrom add failed (%v) and allowance compensation failed (%v)", err, saveErr)
				}
			}
			return transferErr
		}

		newApproval := NewApproval(req.TokenID, req.Owner, req.Spender, newApprovalAmount)
		if err := s.repo.SaveApproval(newApproval); err != nil {
			transferErr = err
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	if transferErr != nil {
		return nil, transferErr
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
	if err := s.eventBus.Publish(event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// Atomic primitive: closes the TOCTOU race where two concurrent
	// IncreaseAllowance(req, +10) calls both read allowance=50, both
	// compute 60, and both write 60, silently losing one increment.
	newAmount, err := s.repo.TryAdjustApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, newAmount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *TokenService) DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Spender); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// Negate the amount to express "subtract this much" as a
	// signed delta for TryAdjustApproval. The primitive clamps at
	// zero, matching the pre-fix behaviour where newAmount.Sign()
	// < 0 was replaced with NewAmount(0).
	negDelta := &Amount{new(big.Int).Neg(req.Amount.Int)}

	// Atomic primitive: closes the TOCTOU race where two concurrent
	// DecreaseAllowance(req, -10) calls both read allowance=50, both
	// compute 40, and both write 40, silently losing one decrement.
	newAmount, err := s.repo.TryAdjustApproval(req.TokenID, req.Owner, req.Spender, negDelta)
	if err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, newAmount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, err
	}
	return event, nil
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

	event := NewBurnEvent(req.TokenID, req.From, req.Amount)

	data := fmt.Sprintf("burn|%s|%s", req.TokenID, req.From)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		if err := s.eventBus.Publish(event); err != nil {
			return err
		}

		// Atomic balance subtract: closes the TOCTOU race the
		// pre-fix path had, where two concurrent burns both
		// read the same balance, both passed the Cmp(amount)
		// check, and both wrote back balance - amount,
		// silently allowing overdraw. The same primitive that
		// fixed Transfer/Mint/TransferFrom in Round 20 closes
		// this gap too.
		if _, err := s.repo.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *TokenService) GetTransferHistory(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error) {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	events, err := s.eventReader.GetTransferEventsByOwner(tokenID, owner, limit, offset)
	if err != nil {
		return nil, err
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
