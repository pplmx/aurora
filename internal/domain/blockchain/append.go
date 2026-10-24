package blockchain

import (
	"database/sql"
	"fmt"
)

// AppendWithRecord writes data as the next block and runs record inside the
// same transaction, returning the height the block was written at.
//
// Two paths exist because a chain may or may not be database-backed:
//
//   - A chain that implements BlockTransactionWriter appends the block and runs
//     record in one transaction, so a failed business write never leaves a
//     block behind.
//   - An in-memory chain cannot join a SQL transaction. It appends first, then
//     runs record in its own transaction; the caller must treat a record error
//     as fatal rather than assuming the block is absent.
//
// The token and NFT services each carried a byte-identical copy of this
// branching; it belongs with the interfaces it switches on.
func AppendWithRecord(
	chain BlockWriter,
	txManager TransactionManager,
	data string,
	record func(tx *sql.Tx, height int64) error,
) (int64, error) {
	if chain == nil {
		return 0, fmt.Errorf("blockchain not initialized")
	}

	if writer, ok := chain.(BlockTransactionWriter); ok {
		var height int64
		err := writer.WithBlockTransaction(data, txManager, func(tx *sql.Tx, blockHeight int64) error {
			height = blockHeight
			return record(tx, blockHeight)
		})
		return height, err
	}

	height, err := chain.AddBlock(data)
	if err != nil {
		return 0, err
	}
	if err := txManager.WithTransaction(func(tx *sql.Tx) error {
		return record(tx, height)
	}); err != nil {
		return 0, err
	}
	return height, nil
}
