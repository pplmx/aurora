package nft

import "database/sql"

// TransactionManager abstracts the infra transaction runner so domain code
// can group multiple repository writes into a single atomic unit without
// importing database/sql plumbing. Implementations commit on nil and roll
// back on error.
type TransactionManager interface {
	WithTransaction(fn func(tx *sql.Tx) error) error
}

type Repository interface {
	SaveNFT(nft *NFT) error
	GetNFT(id string) (*NFT, error)
	// GetNFTsByOwner returns the NFTs owned by owner, paged with SQL
	// LIMIT/OFFSET. limit <= 0 means unbounded (0, 0 is the CLI/TUI default;
	// the REST layer always passes a bounded limit, TASK-101, ISS-093).
	GetNFTsByOwner(owner []byte, limit, offset int) ([]*NFT, error)
	GetNFTsByCreator(creator []byte) ([]*NFT, error)
	UpdateNFT(nft *NFT) error

	// TryTransferOwnership atomically transfers ownership from
	// `from` to `to` for the given nftID. If the current owner
	// does not match `from` (e.g. a concurrent transfer has
	// already moved it), the operation fails and returns
	// ErrOwnershipChanged without modifying state.
	//
	// This is the atomic primitive that closes the TOCTOU window
	// in NFTService.Transfer (the read-modify-write path silently
	// lost concurrent transfers and let double-spend-style
	// ownership swaps complete).
	TryTransferOwnership(nftID string, from, to []byte) error

	// TryDeleteNFTIfOwned atomically deletes the NFT if and only
	// if its current owner matches `expectedOwner`. Returns
	// ErrOwnershipChanged if ownership moved under us (concurrent
	// transfer or burn), ErrNFTNotFound if the NFT doesn't exist.
	//
	// This is the atomic primitive that closes the TOCTOU window
	// in NFTService.Burn (the read-then-delete path let two
	// concurrent burns both succeed, or a transfer-vs-burn race
	// produce an inconsistent state where the audit log shows
	// both a transfer and a burn for the same NFT).
	TryDeleteNFTIfOwned(nftID string, expectedOwner []byte) error

	DeleteNFT(id string) error
	SaveOperation(op *Operation) error
	GetOperations(nftID string) ([]*Operation, error)
}

// TransactableRepository is a Repository that can additionally scope its
// operations to an open transaction. Services that must persist several rows
// atomically (e.g. an NFT row together with its audit operation) obtain a
// tx-scoped repository via WithTx and drive all writes of the unit through
// it. Follows the token module's decision-token-tx-scoped-repos pattern.
type TransactableRepository interface {
	Repository
	// WithTx returns a Repository whose operations participate in tx.
	// Implementations that have no real transaction support (in-memory)
	// return the receiver unchanged.
	WithTx(tx *sql.Tx) Repository
}

// ErrOwnershipChanged is returned by TryTransferOwnership when the
// NFT's current owner does not match the expected `from`. This
// signals to the caller that a concurrent operation has already
// moved ownership and the transfer must be aborted (or retried
// against the new state).
var ErrOwnershipChanged = &OwnershipChangedError{}

type OwnershipChangedError struct{}

func (*OwnershipChangedError) Error() string {
	return "nft ownership changed under concurrent transfer"
}
