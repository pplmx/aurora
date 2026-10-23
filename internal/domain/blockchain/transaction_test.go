package blockchain

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func TestBlockChain_WithBlockTransaction_CommitsBlockAndBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &BlockChain{Blocks: []*Block{Genesis()}}
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

func TestBlockChain_WithBlockTransaction_RollsBackBlockWithBusinessRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &BlockChain{Blocks: []*Block{Genesis()}}
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

func TestBlockChain_WithBlockTransaction_HeightConflictRetriesBeforeBusinessWrite(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &BlockChain{Blocks: []*Block{Genesis()}}
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

func TestBlockChain_WithBlockTransaction_HoldsChainLockThroughCommit(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "aurora.db"))
	createBusinessTable(t, db)

	chain := &BlockChain{Blocks: []*Block{Genesis()}}
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
