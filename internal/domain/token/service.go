package token

import (
	"database/sql"

	"github.com/pplmx/aurora/internal/ledger"
)

// defaultHistoryLimit is the default maximum number of transfer events to return
// when querying transaction history.
const defaultHistoryLimit = 50

type Service interface {
	CreateToken(req *CreateTokenRequest) (*Token, error)
	GetTokenInfo(tokenID TokenID) (*Token, error)
	// ListTokens returns stored tokens newest first. limit <= 0 returns the
	// remainder from offset, which keeps CLI callers able to request the whole
	// collection while API/desktop callers ask for bounded pages.
	ListTokens(limit, offset int) ([]*Token, error)

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

// TransactionManager is the shared transaction port. The alias keeps existing
// token callers and mocks source-compatible after the port moved to ledger.
type TransactionManager = ledger.TransactionManager

type Repository interface {
	SaveToken(token *Token) error
	GetToken(id TokenID) (*Token, error)
	// ListTokens returns stored tokens newest first, using LIMIT/OFFSET in
	// storage so a large collection is never materialised just to show one
	// page. limit <= 0 returns the remainder from offset.
	ListTokens(limit, offset int) ([]*Token, error)

	SaveApproval(approval *Approval) error
	GetApproval(tokenID TokenID, owner, spender PublicKey) (*Approval, error)

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
	GetTransferEventsByToken(tokenID TokenID, limit, offset int) ([]*TransferEvent, error)
}

type TokenService struct {
	repo        TransactableRepository
	txManager   TransactionManager
	eventBus    EventBus
	eventReader EventReader
	replay      ReplayProtection
	chain       ledger.BlockWriter
}

func NewService(repo TransactableRepository, txManager TransactionManager, eventBus EventBus, eventReader EventReader, replay ReplayProtection, chain ledger.BlockWriter) *TokenService {
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

func NewServiceWithoutTx(repo TransactableRepository, eventBus EventBus, eventReader EventReader, replay ReplayProtection, chain ledger.BlockWriter) *TokenService {
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

// withBlockTransaction runs the token mutation and its audit block in one
// database transaction when the chain supports it. The fallback preserves the
// legacy best-effort order for in-memory/fake chains without transaction
// support.
// withBlockTransaction appends the audit block and applies fn in one
// transaction, via the chain's own branching (see ledger.AppendWithRecord).
func (s *TokenService) withBlockTransaction(data string, fn func(repo Repository, height int64) error) (int64, error) {
	return ledger.AppendWithRecord(s.chain, s.txManager, data,
		func(tx *sql.Tx, height int64) error { return fn(s.txRepo(tx), height) })
}

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply *Amount
	Owner       PublicKey
	// Decimals is optional. nil means the caller omitted it and the token gets
	// defaultDecimals; a non-nil value is explicit and may be 0 (TASK-099,
	// ISS-091, ISS-126).
	Decimals *int8
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
