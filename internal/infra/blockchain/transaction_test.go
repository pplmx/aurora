package blockchain

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/ledger"
)

type testDBTxManager struct {
	db *sql.DB
}

func (m testDBTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type testNilTxManager struct{}

func (testNilTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

type testNilTxManagerError struct {
	err error
}

func (m testNilTxManagerError) WithTransaction(fn func(tx *sql.Tx) error) error {
	if err := fn(nil); err != nil {
		return err
	}
	return m.err
}

func createBusinessTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE business_records (id TEXT PRIMARY KEY, block_height INTEGER NOT NULL)`)
	require.NoError(t, err)
}

func businessRecordCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM business_records`).Scan(&count))
	return count
}

func TestChain_WithBlockTransaction_CommitsBlockAndBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	var callbackHeight int64
	err := chain.WithBlockTransaction("mint|T1|alice", testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
		callbackHeight = height
		require.Len(t, chain.Blocks, 1, "the uncommitted block must not be visible in memory")
		_, insertErr := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "record-1", height)
		return insertErr
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), callbackHeight)

	require.Len(t, chain.Blocks, 2)
	require.Equal(t, int64(1), chain.Blocks[1].Height)
	require.Equal(t, "mint|T1|alice", string(readBlockAt(t, db, 1).Data))
	require.Equal(t, 1, businessRecordCount(t, db))
	require.True(t, chain.VerifyIntegrity().Valid)
}

func TestChain_WithRecordTransaction_StampsHeightAndCommitsBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	payload := `{"id":"draw-1","block_height":0,"seed":"s"}`
	var callbackHeight int64
	err := chain.WithRecordTransaction(payload, testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
		callbackHeight = height
		require.Len(t, chain.Blocks, 1, "the uncommitted block must not be visible in memory")
		_, insertErr := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "record-1", height)
		return insertErr
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), callbackHeight)

	require.Len(t, chain.Blocks, 2)
	require.Equal(t, int64(1), chain.Blocks[1].Height)
	require.Equal(t, 1, businessRecordCount(t, db))
	require.True(t, chain.VerifyIntegrity().Valid)

	stored := readBlockAt(t, db, 1)
	require.JSONEq(t, `{"id":"draw-1","block_height":1,"seed":"s"}`, string(stored.Data))
	require.Equal(t, stored.Data, chain.Blocks[1].Data)
}

func TestChain_WithRecordTransaction_RollsBackStampAndBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	rollbackErr := errors.New("business mutation failed")
	err := chain.WithRecordTransaction(`{"id":"draw-1","block_height":0}`, testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
		if _, insertErr := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "record-1", height); insertErr != nil {
			return insertErr
		}
		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	require.Len(t, chain.Blocks, 1, "rolled-back transaction must not leave an in-memory block")
	require.Equal(t, 0, businessRecordCount(t, db))

	var blockCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
	require.Zero(t, blockCount, "rolled-back transaction must not leave a persisted block")
}

func TestChain_WithBlockTransaction_RollsBackBlockWithBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	rollbackErr := errors.New("business mutation failed")
	err := chain.WithBlockTransaction("transfer|lost-race", testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
		if _, insertErr := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "record-1", height); insertErr != nil {
			return insertErr
		}
		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	require.Len(t, chain.Blocks, 1, "rolled-back transaction must not leave an in-memory block")
	require.Equal(t, 0, businessRecordCount(t, db))

	var blockCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
	require.Zero(t, blockCount, "rolled-back transaction must not leave a persisted block")
}

func TestChain_WithBlockTransaction_NoOpFailureRollsBackPersistedBlock(t *testing.T) {
	tests := []struct {
		name         string
		manager      func(error) ledger.TransactionManager
		failCallback bool
	}{
		{
			name: "callback failure",
			manager: func(error) ledger.TransactionManager {
				return testNilTxManager{}
			},
			failCallback: true,
		},
		{
			name: "manager failure after callback",
			manager: func(err error) ledger.TransactionManager {
				return testNilTxManagerError{err: err}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
			chain := &Chain{Blocks: []*Block{Genesis()}}
			bindChainToDB(chain, db)

			transactionErr := errors.New("no-op transaction failed")
			var failedHeight int64
			err := chain.WithBlockTransaction("failed-transfer", tt.manager(transactionErr), func(tx *sql.Tx, height int64) error {
				require.Nil(t, tx)
				require.Len(t, chain.Blocks, 1, "the persisted block must not be visible in memory")
				failedHeight = height
				if tt.failCallback {
					return transactionErr
				}
				return nil
			})
			require.ErrorIs(t, err, transactionErr)
			require.Equal(t, int64(1), failedHeight)
			require.Len(t, chain.Blocks, 1)

			var blockCount int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
			require.Zero(t, blockCount, "failed no-op transaction must not leave a persisted block")

			var successHeight int64
			err = chain.WithBlockTransaction("successful-transfer", testNilTxManager{}, func(tx *sql.Tx, height int64) error {
				require.Nil(t, tx)
				successHeight = height
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, int64(1), successHeight, "the next success must reuse height 1")
			require.Len(t, chain.Blocks, 2)
			require.Equal(t, "successful-transfer", string(readBlockAt(t, db, 1).Data))

			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
			require.Equal(t, 1, blockCount)
			require.True(t, chain.VerifyIntegrity().Valid)
		})
	}
}

func TestChain_WithBlockTransaction_NoOpHeightConflictRetriesBeforeCallback(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	peer := peerChain(t, 1, "peer-payload")
	insertBlockRaw(t, db, peer.Blocks[1])

	realSync := chain.syncFromDB
	staleFirst := true
	chain.syncFromDB = func() error {
		if staleFirst {
			staleFirst = false
			return nil
		}
		return realSync()
	}

	callbacks := 0
	var callbackHeight int64
	err := chain.WithBlockTransaction("local-payload", testNilTxManager{}, func(tx *sql.Tx, height int64) error {
		require.Nil(t, tx)
		callbacks++
		callbackHeight = height
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, callbacks, "a height-conflicting no-op attempt must fail before running the business callback")
	require.Equal(t, int64(2), callbackHeight)
	require.Equal(t, "peer-payload", string(readBlockAt(t, db, 1).Data))
	require.Equal(t, "local-payload", string(readBlockAt(t, db, 2).Data))
	require.Len(t, chain.Blocks, 3)
	require.True(t, chain.VerifyIntegrity().Valid)
}

func TestChain_WithBlockTransaction_HeightConflictRetriesBeforeBusinessWrite(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	peer := peerChain(t, 1, "peer-payload")
	insertBlockRaw(t, db, peer.Blocks[1])

	realSync := chain.syncFromDB
	staleFirst := true
	chain.syncFromDB = func() error {
		if staleFirst {
			staleFirst = false
			return nil
		}
		return realSync()
	}

	callbacks := 0
	err := chain.WithBlockTransaction("local-payload", testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
		callbacks++
		_, insertErr := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "record-1", height)
		return insertErr
	})
	require.NoError(t, err)
	require.Equal(t, 1, callbacks, "a height-conflicting attempt must fail before running the business callback")
	require.Equal(t, 1, businessRecordCount(t, db))
	require.Equal(t, int64(2), chain.Blocks[2].Height)
	require.Equal(t, "local-payload", string(readBlockAt(t, db, 2).Data))
}

func TestChain_WithBlockTransaction_HoldsChainLockThroughCommit(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	callbackEntered := make(chan struct{})
	releaseCallback := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- chain.WithBlockTransaction("first", testDBTxManager{db: db}, func(tx *sql.Tx, height int64) error {
			close(callbackEntered)
			<-releaseCallback
			_, err := tx.Exec(`INSERT INTO business_records (id, block_height) VALUES (?, ?)`, "first", height)
			return err
		})
	}()

	select {
	case <-callbackEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for transaction callback")
	}

	secondDone := make(chan error, 1)
	go func() {
		_, err := chain.AddBlock("second")
		secondDone <- err
	}()

	select {
	case err := <-secondDone:
		t.Fatalf("AddBlock completed before the first transaction committed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseCallback)
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)

	require.Len(t, chain.Blocks, 3)
	require.Equal(t, "first", string(chain.Blocks[1].Data))
	require.Equal(t, "second", string(chain.Blocks[2].Data))
	require.True(t, chain.VerifyIntegrity().Valid)
}
