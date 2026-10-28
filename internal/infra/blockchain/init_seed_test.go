package blockchain

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestBindChainToDB_SeedsGenesisOnce pins the genesis seeding contract
// (TASK-238, ISS-236): binding must insert the genesis block ONLY into an empty
// blocks table. On a reload the height-0 row is skipped during the in-memory
// rebuild (chain.Blocks holds only genesis again), so the OLD guard
// `len(Blocks) <= 1` stayed true and re-inserted height 0 on every boot —
// tripping the PRIMARY KEY and logging a spurious "Failed to insert genesis
// block" on a healthy restart. Keying on any persisted row fixes it.
func TestBindChainToDB_SeedsGenesisOnce(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "seed.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	countRows := func() int {
		var n int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&n))
		return n
	}

	chain := &Chain{Blocks: []*Block{Genesis()}}

	// First boot: empty table -> genesis is seeded exactly once.
	require.NoError(t, bindChainToDB(chain, db))
	require.Equal(t, 1, countRows(), "first boot must seed genesis")

	// Reload: the h0 row is already persisted -> the seed must be skipped. If
	// it were attempted the PRIMARY KEY conflict would surface (and the old
	// code logged a false error every start).
	reloaded := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(reloaded, db))
	require.Equal(t, 1, countRows(), "reload must not re-insert genesis (PK conflict)")

	// A reload of a DB that has gone on to append blocks is also a no-op.
	_, err = chain.AddBlock("post-genesis")
	require.NoError(t, err)
	reloadedAgain := &Chain{Blocks: []*Block{Genesis()}}
	require.NoError(t, bindChainToDB(reloadedAgain, db))
	require.Equal(t, 2, countRows(), "genesis plus the appended block")
	var genesisRows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height = 0").Scan(&genesisRows))
	require.Equal(t, 1, genesisRows, "later boots must still not seed a duplicate genesis")
}
