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

	// TrySubtractFromSupply atomically subtracts amount from the
	// token's total_supply (Burn), failing if the current supply
	// is less than amount. Returns the new total supply.
	TrySubtractFromSupply(tokenID TokenID, amount *Amount) (*Amount, error)
}

type TransactableRepository interface {
	Repository
	// WithTx returns a Repository whose read/write operations execute
	// within the given transaction. tx is nil for the non-transactional
	// path (e.g. in-memory repos / mock tx managers).
	WithTx(tx *sql.Tx) Repository
}

type EventReader interface {
	GetTransferEventsByOwner(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error)
	GetMintEventsByToken(tokenID TokenID) ([]*MintEvent, error)
	GetBurnEventsByToken(tokenID TokenID) ([]*BurnEvent, error)
}

type TokenService struct {
	repo        TransactableRepository
	txManager   TransactionManager
	eventBus    infraevents.EventBus
	eventReader EventReader
	replay      infraevents.ReplayProtection
	chain       blockchain.BlockWriter
}

func NewService(repo TransactableRepository, txManager TransactionManager, eventBus infraevents.EventBus, eventReader EventReader, replay infraevents.ReplayProtection, chain blockchain.BlockWriter) *TokenService {
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

func NewServiceWithoutTx(repo TransactableRepository, eventBus infraevents.EventBus, eventReader EventReader, replay infraevents.ReplayProtection, chain blockchain.BlockWriter) *TokenService {
	return NewService(repo, noOpTxManager{}, eventBus, eventReader, replay, chain)
}

// txRepo returns the repository scoped to the given transaction when one is
// active, otherwise the service's default repository. Every state mutation
// performed inside a WithTransaction callback MUST go through txRepo so the
// writes share the same SQLite transaction and roll back together.
func (s *TokenService) txRepo(tx *sql.Tx) Repository {
	if tx == nil {
		return s.repo
	}
	return s.repo.WithTx(tx)
}

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply *Amount
	Owner       PublicKey
	// Decimals is the number of decimal places stored on the token (default
	// defaultDecimals when 0 — the unset sentinel, TASK-099, ISS-091).
	Decimals int8
}

type MintRequest struct {
	TokenID    TokenID
	To         PublicKey
	Amount     *Amount
	PrivateKey PrivateKey
}

type TransferRequest struct {
	TokenID    TokenID
	From       PublicKey
	To         PublicKey
	Amount     *Amount
	PrivateKey PrivateKey
}

type TransferFromRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	To         PublicKey
	Amount     *Amount
	Spender    PublicKey
	SpenderKey PrivateKey
}

type ApproveRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	Spender    PublicKey
	Amount     *Amount
	PrivateKey PrivateKey
}

type AllowanceRequest struct {
	TokenID    TokenID
	Owner      PublicKey
	Spender    PublicKey
	Amount     *Amount
	PrivateKey PrivateKey
}

type BurnRequest struct {
	TokenID    TokenID
	From       PublicKey
	Amount     *Amount
	PrivateKey PrivateKey
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
	if err := ValidateTokenDecimals(req.Decimals); err != nil {
		return nil, err
	}

	// A token's ID is its symbol (see NewToken). The persistence layer uses
	// INSERT OR REPLACE keyed on that ID, so creating a token with a symbol
	// that already exists would silently overwrite the existing token's row
	// and owner balance — losing the first token's data. Reject duplicates up
	// front so a create is either a genuine new token or an explicit error,
	// never a silent data-loss (TASK-075, ISS-067).
	tokenID := TokenID(req.Symbol)
	existing, err := s.repo.GetToken(tokenID)
	if err != nil && err != ErrTokenNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTokenExists
	}

	token := NewTokenWithDecimals(tokenID, req.Name, req.Symbol, req.TotalSupply, req.Owner, req.Decimals)

	// CreateToken performs two writes (token row + owner balance) that must
	// be atomic: a failure between them would otherwise leave a token with
	// zero owner balance.
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		r := s.txRepo(tx)
		if err := r.SaveToken(token); err != nil {
			return err
		}
		return r.SetAccountBalance(token.ID(), req.Owner, req.TotalSupply)
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (s *TokenService) GetTokenInfo(tokenID TokenID) (*Token, error) {
	return s.repo.GetToken(tokenID)
}

// requireToken returns ErrTokenNotFound unless a token with tokenID exists,
// mirroring the existence guard every mutator (Mint/Burn/Transfer/TransferFrom/
// Approve) applies before touching a row. The read paths (GetBalance/
// GetAllowance/GetTransferHistory) previously skipped it: a typo'd or never
// created token_id read back as a legitimate zero balance / empty history on
// the REST surface while /token/info 404'd — masking an existence probe as
// "no activity" instead of the unknown-resource->404 contract the rest of the
// platform enforces (ISS-250). Handles both not-found conventions the repos
// use (ErrTokenNotFound as err, or a nil token without error).
func (s *TokenService) requireToken(tokenID TokenID) error {
	t, err := s.repo.GetToken(tokenID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenNotFound
	}
	return nil
}

func (s *TokenService) GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error) {
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
	return s.repo.GetAccountBalance(tokenID, owner)
}

func (s *TokenService) GetAllowance(tokenID TokenID, owner, spender PublicKey) (*Amount, error) {
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
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

	if err := VerifyPrivateKeyMatches(token.Owner(), req.PrivateKey); err != nil {
		return nil, err
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
		r := s.txRepo(tx)

		// Atomic supply increment: closes the TOCTOU window
		// where two concurrent mints both read total_supply,
		// both added their amount in memory via AddToSupply,
		// and the last SaveToken clobbered the other mint's
		// increment — silently producing less total_supply
		// than the sum of all mints.
		if _, err := r.TryAddToSupply(req.TokenID, req.Amount); err != nil {
			return err
		}

		// Atomic add: closes the race where two concurrent Mints
		// to the same account both read currentBalance, compute
		// currentBalance + amount, and one overwrites the other.
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// The audit event is published only AFTER the transaction commits
	// (ISS-074). eventBus writes to its own events store, which a token DB
	// rollback cannot undo, so publishing inside the transaction left a
	// phantom event behind whenever a later step rolled the mint back.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
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
	if err := VerifyPrivateKeyMatches(req.From, req.PrivateKey); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.Amount); err != nil {
		return nil, err
	}

	// No balance pre-check here: the atomic TrySubtractBalance in the
	// transaction below is the single authority — a non-atomic read up front
	// ran against a different snapshot, so a concurrent top-up between the
	// read and the transaction returned a spurious INSUFFICIENT_BALANCE 400
	// even though the atomic debit would (and did) succeed (TASK-265,
	// ISS-261). The atomic primitive classifies the error itself, and the
	// handler maps its sentinel to the same 400 code.

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.From)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(ed25519.PrivateKey(req.PrivateKey), s.signMessage(req.TokenID, req.From, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.From, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transfer|%s|%s|%s", req.TokenID, req.From, req.To)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		r := s.txRepo(tx)

		// ClaimNextNonce above already persisted the nonce atomically on the
		// replay store's own connection, so re-saving it here would be a
		// no-op — and where that store shares the token DB file (the API
		// server wiring), this write through a SECOND connection would
		// deadlock against this transaction's write lock (BEGIN IMMEDIATE,
		// v1.70 / ISS-076): the tx would wait on the separate
		// SaveNonce while SaveNonce waits for the tx's lock, tripping the 5s
		// busy timeout into a spurious 500 (ISS-078).

		// Atomic subtract: closes the TOCTOU race where two
		// concurrent transfers both read fromBalance, both pass
		// the check, and both write back (fromBalance - amount).
		// The debit and credit below run in the same transaction:
		// if either fails, the whole transfer rolls back, so a
		// partial debit can never be left behind.
		if _, err := r.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			return err
		}

		// Atomic add: closes the symmetric race on the credit
		// side (two concurrent transfers to the same recipient
		// could both read toBalance and both write back
		// toBalance + amount, losing one transfer's credit).
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Publish the audit event only after the transaction commits (ISS-074);
	// see the identical note in Mint.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
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
	if err := VerifyPrivateKeyMatches(req.Spender, req.SpenderKey); err != nil {
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
		// A nonexistent allowance is distinguished up front so the caller
		// still gets INSUFFICIENT_ALLOWANCE rather than the repo's not-found
		// error; the atomic TryDeductApproval below is the authority on the
		// remaining amount checks (TASK-265, ISS-261).
		return nil, ErrInsufficientAllowance
	}

	// No balance/allowance pre-check here: the atomic TryDeductApproval and
	// TrySubtractBalance in the transaction below are the single authorities.
	// Non-atomic reads up front ran against a different snapshot, so a
	// concurrent allowance increase / owner top-up between the read and the
	// transaction returned a spurious 400 even though the atomic primitives
	// would (and did) succeed (TASK-265, ISS-261). Each primitive classifies
	// its own error and the handler maps the sentinels to the same codes.

	nonce, err := s.replay.ClaimNextNonce(string(req.TokenID), req.Spender)
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(ed25519.PrivateKey(req.SpenderKey), s.signMessage(req.TokenID, req.Owner, req.To, req.Amount, nonce))

	event := NewTransferEvent(req.TokenID, req.Owner, req.To, req.Amount, nonce, signature)

	data := fmt.Sprintf("transferfrom|%s|%s|%s", req.TokenID, req.Owner, req.To)
	height, err := s.chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	event.SetBlockHeight(height)

	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		r := s.txRepo(tx)

		// No replay.SaveNonce here for the same reason as Transfer: the
		// nonce was already persisted atomically by ClaimNextNonce above, and
		// writing it again via the replay store's separate connection would
		// deadlock against this transaction's write lock where both share the
		// same SQLite file (v1.70 / ISS-076, see Transfer).

		// All three mutations (allowance deduction, owner debit,
		// recipient credit) run in the same transaction. If any
		// step fails the transaction rolls back, so the allowance
		// and balances stay consistent — no hand-rolled
		// compensation is needed.
		//
		// Atomic allowance deduction: closes the TOCTOU race
		// where two concurrent TransferFroms both read
		// approval.Amount(), both pass the check, and both write
		// back approval - amount, allowing double-spend of the
		// allowance.
		if _, err := r.TryDeductApproval(req.TokenID, req.Owner, req.Spender, req.Amount); err != nil {
			return err
		}

		// Atomic balance subtract (owner) and add (recipient).
		if _, err := r.TrySubtractBalance(req.TokenID, req.Owner, req.Amount); err != nil {
			return err
		}
		if _, err := r.TryAddBalance(req.TokenID, req.To, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Publish the audit event only after the transaction commits (ISS-074);
	// see the identical note in Mint.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}

	return event, nil
}

func (s *TokenService) Approve(req *ApproveRequest) (*ApproveEvent, error) {
	// Guard: refuse to approve an allowance for a token that does not
	// exist. Consistent with the other mutators (Mint/Burn/Transfer/
	// TransferFrom/Increase/DecreaseAllowance): without it a repo that
	// returns a nil token without error (e.g. a mock) would let SaveApproval
	// create an orphaned allowance row referencing a token that never existed.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if err == ErrTokenNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}

	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
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
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}

	return event, nil
}

func (s *TokenService) IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	// Guard: refuse to adjust allowance for a token that does not exist.
	// Every other token mutator (Approve/Mint/Burn/Transfer/TransferFrom)
	// verifies the token first; without this check TryAdjustApproval would
	// happily INSERT an allowance row for a dangling token_id, creating
	// orphaned approval data that references nothing.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if err == ErrTokenNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}
	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
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

	// Atomic primitive: closes the TOCTOU race where two concurrent
	// IncreaseAllowance(req, +10) calls both read allowance=50, both
	// compute 60, and both write 60, silently losing one increment.
	newAmount, err := s.repo.TryAdjustApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	if err != nil {
		return nil, err
	}

	event := NewApproveEvent(req.TokenID, req.Owner, req.Spender, newAmount)
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}
	return event, nil
}

func (s *TokenService) DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error) {
	// Guard: see IncreaseAllowance — never adjust allowance for a token
	// that does not exist. Prevents orphaned allowance rows for a dangling
	// token_id.
	token, err := s.repo.GetToken(req.TokenID)
	if err != nil {
		if err == ErrTokenNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if token == nil {
		return nil, ErrTokenNotFound
	}
	if err := VerifyPrivateKeyMatches(req.Owner, req.PrivateKey); err != nil {
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
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
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

	if err := VerifyPrivateKeyMatches(req.From, req.PrivateKey); err != nil {
		return nil, err
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
		r := s.txRepo(tx)

		// Atomic balance subtract: closes the TOCTOU race the
		// pre-fix path had, where two concurrent burns both
		// read the same balance, both passed the Cmp(amount)
		// check, and both wrote back balance - amount,
		// silently allowing overdraw. The same primitive that
		// fixed Transfer/Mint/TransferFrom in Round 20 closes
		// this gap too.
		if _, err := r.TrySubtractBalance(req.TokenID, req.From, req.Amount); err != nil {
			return err
		}

		// Burning removes tokens from circulation, so it must also
		// decrement total_supply to preserve the ledger invariant
		// total_supply == sum of all balances. Both writes are in
		// the same transaction and roll back together.
		if _, err := r.TrySubtractFromSupply(req.TokenID, req.Amount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Publish the audit event only after the transaction commits (ISS-074);
	// see the identical note in Mint.
	if err := s.eventBus.Publish(event); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuditPublishFailed, err)
	}

	return event, nil
}

func (s *TokenService) GetTransferHistory(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error) {
	if err := s.requireToken(tokenID); err != nil {
		return nil, err
	}
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
