package blockchain

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/store"
)

func seedUnscannableBlocks(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE blocks (
			height INTEGER PRIMARY KEY,
			hash TEXT NOT NULL,
			previous_hash TEXT NOT NULL,
			data TEXT NOT NULL,
			nonce INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)
	genesis := Genesis()
	_, err = db.Exec(`
		INSERT INTO blocks VALUES (0, ?, ?, ?, ?, ?, ?)
	`, string(genesis.Hash), string(genesis.PrevHash), string(genesis.Data), genesis.Nonce, genesis.Timestamp, genesis.Timestamp)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO blocks VALUES (1, 'one', 'genesis', 'one', 0, 1, 1);
		INSERT INTO blocks VALUES (2, 'two', 'one', 'unscannable', X'626164', 1, 1);
		INSERT INTO blocks VALUES (3, 'three', 'two', 'three', 0, 1, 1);
	`)
	require.NoError(t, err)
}

// TestLoadBlocksFromDB_RejectsUnscannableRow locks the boot-time integrity
// contract: a row the driver cannot decode is ledger corruption, not a row the
// process may silently skip. Skipping it would load a chain with a missing
// height and make later appends collide with the still-existing database row.
func TestLoadBlocksFromDB_RejectsUnscannableRow(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "corrupt.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	seedUnscannableBlocks(t, db)

	chain := &Chain{Blocks: []*Block{Genesis()}}
	persisted, err := loadBlocksFromDB(chain, db)

	require.Error(t, err, "an unscannable persisted block must fail the reload")
	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.True(t, persisted, "the corrupt row still proves the ledger is non-empty")
	require.Len(t, chain.Blocks, 1, "a failed reload must not expose a partial chain")
}

// TestNewWithDB_RejectsUnscannablePersistedBlock proves the error reaches the
// composition root instead of returning a chain that has silently skipped a
// row still present in SQLite.
func TestNewWithDB_RejectsUnscannablePersistedBlock(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "corrupt.db"))
	seedUnscannableBlocks(t, db)

	chain, err := NewWithDB(db)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.Nil(t, chain)
	require.Contains(t, err.Error(), "scan block row 3")
}

// TestNewWithDB_RejectsHeightGap covers corruption that SQLite can scan but
// that no longer forms the position-addressable chain appends rely on.
func TestNewWithDB_RejectsHeightGap(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "gap.db"))
	chain := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(chain, db))

	_, err := chain.AddBlock("one")
	require.NoError(t, err)
	_, err = chain.AddBlock("two")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM blocks WHERE height = 1")
	require.NoError(t, err)

	reloaded, err := NewWithDB(db)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.Nil(t, reloaded)
	require.Contains(t, err.Error(), "height discontinuity")
}

// TestNewWithDB_RejectsInvalidPersistedGenesis closes the gap left by skipping
// height 0 during reload: the in-memory chain must not silently replace a
// persisted genesis row that is missing or different.
func TestNewWithDB_RejectsInvalidPersistedGenesis(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, db *sql.DB)
	}{
		{
			name: "missing",
			mutate: func(t *testing.T, db *sql.DB) {
				_, err := db.Exec("DELETE FROM blocks WHERE height = 0")
				require.NoError(t, err)
			},
		},
		{
			name: "tampered",
			mutate: func(t *testing.T, db *sql.DB) {
				_, err := db.Exec("UPDATE blocks SET data = 'Tampered' WHERE height = 0")
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openSecondHandle(t, filepath.Join(t.TempDir(), "genesis.db"))
			chain := &Chain{Blocks: []*Block{Genesis()}}
			require.NoError(t, bindChainToDB(chain, db))
			_, err := chain.AddBlock("one")
			require.NoError(t, err)
			tt.mutate(t, db)

			reloaded, err := NewWithDB(db)

			require.Error(t, err)
			require.ErrorIs(t, err, ErrLedgerCorruption)
			require.Nil(t, reloaded)
		})
	}
}

// TestChain_AddBlock_RejectsCorruptLedgerAtSync covers corruption introduced
// after startup. The seam hash no longer matches memory, so syncFromDB rebuilds
// from SQLite; that rebuild must verify the ledger before replacing the usable
// in-memory chain or allowing another block to be appended on top.
func TestChain_AddBlock_RejectsCorruptLedgerAtSync(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "runtime-corrupt.db"))
	chain := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(chain, db))
	_, err := chain.AddBlock("one")
	require.NoError(t, err)

	_, err = db.Exec("UPDATE blocks SET hash = 'tampered' WHERE height = 1")
	require.NoError(t, err)

	_, err = chain.AddBlock("two")

	require.ErrorIs(t, err, ErrLedgerCorruption, "a corrupt persisted ledger must not accept another append")
	require.Len(t, chain.Blocks, 2, "a failed sync must leave the last known-good in-memory chain intact")
	require.True(t, chain.VerifyIntegrity().Valid)
}

// TestChain_AddBlock_RejectsCorruptNewRow covers the other sync path: the seam
// still matches, but a newly discovered row at the next height does not verify.
// It must not be appended to memory or become the parent of another block.
func TestChain_AddBlock_RejectsCorruptNewRow(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "runtime-corrupt-tail.db"))
	chain := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(chain, db))
	_, err := chain.AddBlock("one")
	require.NoError(t, err)

	insertBlockRaw(t, db, &Block{
		Height:    2,
		Hash:      []byte("tampered"),
		PrevHash:  append([]byte(nil), chain.Blocks[1].Hash...),
		Data:      []byte("two"),
		Timestamp: 1,
	})

	_, err = chain.AddBlock("three")

	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.Len(t, chain.Blocks, 2)
	require.Equal(t, "one", string(chain.Blocks[1].Data))
	require.True(t, chain.VerifyIntegrity().Valid)
}

// TestChain_WithBlockTransaction_RejectsCorruptLedgerAtSync ensures the
// transaction append path fails before running its business callback.
func TestChain_WithBlockTransaction_RejectsCorruptLedgerAtSync(t *testing.T) {
	db := openSecondHandle(t, filepath.Join(t.TempDir(), "runtime-corrupt-tx.db"))
	chain := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(chain, db))
	_, err := chain.AddBlock("one")
	require.NoError(t, err)

	_, err = db.Exec("UPDATE blocks SET hash = 'tampered' WHERE height = 1")
	require.NoError(t, err)

	callbackCalled := false
	err = chain.WithBlockTransaction("two", testDBTxManager{db: db}, func(_ *sql.Tx, _ int64) error {
		callbackCalled = true
		return nil
	})

	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.False(t, callbackCalled, "the business mutation must not run on a corrupt ledger")
	require.Len(t, chain.Blocks, 2)
	require.True(t, chain.VerifyIntegrity().Valid)
}

// TestChain_AddBlock_TransientSyncErrorUsesMemory pins the deliberate fallback
// for ordinary read failures. Only decoded/verified ledger corruption is fatal;
// preserving the in-memory append path avoids turning a busy database into an
// outage.
func TestChain_AddBlock_TransientSyncErrorUsesMemory(t *testing.T) {
	chain := New()
	chain.syncFromDB = func() error { return errors.New("database is busy") }

	height, err := chain.AddBlock("one")

	require.NoError(t, err)
	require.Equal(t, int64(1), height)
	require.Len(t, chain.Blocks, 2)
	require.True(t, chain.VerifyIntegrity().Valid)
}

// TestInitWithError_RejectsCorruptLedger keeps the process singleton's
// error-bearing constructor fail-closed. Returning the partially initialised
// chain alongside the error would let a caller that ignored the error append
// against the corrupt database.
func TestInitWithError_RejectsCorruptLedger(t *testing.T) {
	resetChainForTest(t)
	viper.Reset()
	t.Cleanup(viper.Reset)

	dbPath := filepath.Join(t.TempDir(), "corrupt.db")
	db, err := store.Open(dbPath)
	require.NoError(t, err)
	seedUnscannableBlocks(t, db)
	require.NoError(t, db.Close())
	viper.Set("db.path", dbPath)

	chain, err := InitWithError()

	require.Error(t, err)
	require.ErrorIs(t, err, ErrLedgerCorruption)
	require.Nil(t, chain)
}

// TestInitWithError_ReportsUnopenableDatabase closes the other silent-fallback
// path: a database that cannot be opened must not look like a successful
// process-wide chain whose writes only exist until exit.
func TestInitWithError_ReportsUnopenableDatabase(t *testing.T) {
	resetChainForTest(t)
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("occupied"), 0o600))
	viper.Set("db.path", filepath.Join(blocker, "aurora.db"))

	chain, err := InitWithError()

	require.Error(t, err)
	require.Nil(t, chain)
}
