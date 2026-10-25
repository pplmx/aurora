package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/nft"
)

// TestEnsureNoCascadeFKRebuildsAnOldDatabase covers the upgrade path for
// databases created before TASK-092: their nft_operations carried
// `FOREIGN KEY (nft_id) REFERENCES nfts(id) ON DELETE CASCADE`, so burning an
// NFT wiped its whole audit trail — including the burn operation itself.
//
// The rebuild re-creates the table and copies the rows, which is the riskiest
// migration in the repository: a mistake loses audit history rather than
// failing loudly. It had no direct test.
func TestEnsureNoCascadeFKRebuildsAnOldDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy_nft.db")

	// Build the old schema by hand, as an older release left it.
	legacy, err := sql.Open("sqlite", DSN(dbPath))
	require.NoError(t, err)
	for _, q := range []string{
		`CREATE TABLE nfts (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT,
			image_url TEXT, token_uri TEXT, owner TEXT NOT NULL,
			creator TEXT NOT NULL, block_height INTEGER NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE TABLE nft_operations (
			id TEXT PRIMARY KEY, nft_id TEXT NOT NULL, type TEXT NOT NULL,
			from_addr TEXT, to_addr TEXT, signature TEXT,
			block_height INTEGER NOT NULL, timestamp INTEGER NOT NULL,
			FOREIGN KEY (nft_id) REFERENCES nfts(id) ON DELETE CASCADE
		)`,
		`INSERT INTO nfts VALUES ('n1','Art',NULL,NULL,NULL,'owner','creator',1,100)`,
		`INSERT INTO nft_operations VALUES ('op1','n1','mint',NULL,'owner',NULL,1,100)`,
		`INSERT INTO nft_operations VALUES ('op2','n1','transfer','owner','next',NULL,2,200)`,
	} {
		_, err := legacy.Exec(q)
		require.NoError(t, err, "build the legacy schema: %s", q)
	}
	require.NoError(t, legacy.Close())

	// Opening the repository runs the rebuild.
	repo, err := NewNFTRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	// The FK is gone...
	var fkCount int
	rows, err := repo.GetDB().Query(`PRAGMA foreign_key_list(nft_operations)`)
	require.NoError(t, err)
	for rows.Next() {
		fkCount++
	}
	require.NoError(t, rows.Close())
	assert.Zero(t, fkCount, "nft_operations must carry no foreign key after the rebuild")

	// ...and every audit row survived the copy.
	ops, err := repo.GetOperations("n1")
	require.NoError(t, err)
	require.Len(t, ops, 2, "the rebuild must not lose audit history")
	ids := []string{ops[0].ID, ops[1].ID}
	assert.ElementsMatch(t, []string{"op1", "op2"}, ids)

	// The indexes the rebuild drops and recreates are back, so a migrated
	// database keeps the same query plan as a fresh one.
	var indexCount int
	rows, err = repo.GetDB().Query(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name='nft_operations' AND name LIKE 'idx_nft_ops_%'`)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&indexCount))
	require.NoError(t, rows.Close())
	assert.Equal(t, 2, indexCount, "both operation indexes must be recreated")

	// The point of the whole rebuild: burning the NFT keeps its trail.
	require.NoError(t, repo.DeleteNFT("n1"))
	ops, err = repo.GetOperations("n1")
	require.NoError(t, err)
	assert.Len(t, ops, 2, "burning must not cascade-delete the audit trail")
}

// TestEnsureNoCascadeFKIsIdempotent proves a database already on the new
// schema is left alone: the rebuild runs on every open.
func TestEnsureNoCascadeFKIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fresh_nft.db")

	first, err := NewNFTRepository(dbPath)
	require.NoError(t, err)
	require.NoError(t, first.SaveNFT(&nft.NFT{
		ID: "n1", Name: "Art", Owner: []byte("owner"), Creator: []byte("creator"),
	}))
	require.NoError(t, first.SaveOperation(&nft.Operation{
		ID: "op1", NFTID: "n1", Type: "mint", BlockHeight: 1,
	}))
	require.NoError(t, first.Close())

	// A second open runs createTables and the rebuild again.
	second, err := NewNFTRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })

	ops, err := second.GetOperations("n1")
	require.NoError(t, err)
	require.Len(t, ops, 1, "a no-op rebuild must keep the rows")
	assert.Equal(t, "op1", ops[0].ID)
}
