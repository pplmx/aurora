package blockchain

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestSeedGenesisIfEmpty pins the genesis seeding contract (TASK-238, ISS-236):
// seedGenesisIfEmpty must insert the genesis block ONLY into an empty blocks
// table. On a reload the height-0 row is skipped during the in-memory rebuild
// (chain.Blocks holds only genesis again), so the OLD guard `len(Blocks) <= 1`
// stayed true and re-inserted height 0 on every boot — tripping the PRIMARY
// KEY and logging a spurious "Failed to insert genesis block" on a healthy
// restart. Keying on `persisted` (any row exists) fixes it.
func TestSeedGenesisIfEmpty(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(dir, "seed.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
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

	countRows := func() int {
		var n int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&n))
		return n
	}

	chain := &BlockChain{Blocks: []*Block{Genesis()}}

	// First boot: empty table -> genesis is seeded exactly once.
	seedGenesisIfEmpty(db, chain, false)
	require.Equal(t, 1, countRows(), "first boot must seed genesis")

	// Reload: the h0 row is already persisted -> the seed must be skipped. If
	// it were attempted the PRIMARY KEY conflict would surface (and the old
	// code logged a false error every start).
	seedGenesisIfEmpty(db, chain, true)
	require.Equal(t, 1, countRows(), "reload must not re-insert genesis (PK conflict)")

	// A reload of a DB that has gone on to append blocks is also a no-op.
	seedGenesisIfEmpty(db, chain, true)
	require.Equal(t, 1, countRows(), "later boots must still not seed")
}
