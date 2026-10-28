package blockchain

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
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
			require.Nil(t, reloaded)
		})
	}
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
	db, err := OpenDB(dbPath)
	require.NoError(t, err)
	seedUnscannableBlocks(t, db)
	require.NoError(t, db.Close())
	viper.Set("db.path", dbPath)

	chain, err := InitWithError()

	require.Error(t, err)
	require.Nil(t, chain)
}
