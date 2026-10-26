package ledger

import "database/sql"

type BlockWriter interface {
	AddBlock(data string) (int64, error)
}

// TransactionManager runs a callback in one database transaction. It is
// structurally identical to the transaction managers used by the domain
// repositories, so blockchain does not need to depend on the SQLite package.
type TransactionManager interface {
	WithTransaction(fn func(tx *sql.Tx) error) error
}

// BlockTransactionWriter appends a block and applies the business mutation in
// one SQLite transaction. The callback receives the transaction-scoped block
// height before the transaction commits.
type BlockTransactionWriter interface {
	WithBlockTransaction(
		data string,
		txManager TransactionManager,
		fn func(tx *sql.Tx, height int64) error,
	) error
}

// LotteryRecordTransactionWriter is the height-stamping variant used by
// lottery and oracle records. The on-chain JSON carries the final height even
// when a cross-process height conflict forces a retry.
type LotteryRecordTransactionWriter interface {
	WithRecordTransaction(
		data string,
		txManager TransactionManager,
		fn func(tx *sql.Tx, height int64) error,
	) error
}
