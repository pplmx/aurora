package nft

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// The in-memory repository stands in for the SQLite one in app and transport
// tests, so its listing order has to be the store's order: rowid descending,
// i.e. newest write first. It used to sort by id, and the owner-filtered list
// returned map order, which made a test agree with whatever sequence it got.

func TestInmemRepo_ListNFTsIsNewestFirst(t *testing.T) {
	repo := NewInmemRepo()
	owner := []byte("owner")
	for _, id := range []string{"first", "second", "third"} {
		require.NoError(t, repo.SaveNFT(&NFT{ID: id, Name: id, Owner: owner, Creator: owner}))
	}

	all, err := repo.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"third", "second", "first"}, nftIDs(all),
		"the listing leads with the asset just written")

	page, err := repo.ListNFTs(2, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"second", "first"}, nftIDs(page),
		"a page is a window on that order, not a different one")
}

func TestInmemRepo_OwnerListIsNewestFirst(t *testing.T) {
	repo := NewInmemRepo()
	owner := []byte("owner")
	other := []byte("other")
	require.NoError(t, repo.SaveNFT(&NFT{ID: "mine-old", Owner: owner, Creator: owner}))
	require.NoError(t, repo.SaveNFT(&NFT{ID: "theirs", Owner: other, Creator: other}))
	require.NoError(t, repo.SaveNFT(&NFT{ID: "mine-new", Owner: owner, Creator: owner}))

	byOwner, err := repo.GetNFTsByOwner(owner, 0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"mine-new", "mine-old"}, nftIDs(byOwner),
		"filtering by owner keeps the listing's order")

	// The order must be repeatable: a map walk gave every call its own.
	for i := 0; i < 5; i++ {
		again, err := repo.GetNFTsByOwner(owner, 0, 0)
		require.NoError(t, err)
		require.Equal(t, nftIDs(byOwner), nftIDs(again))
	}
}

func TestInmemRepo_UpdateKeepsItsPlaceInTheList(t *testing.T) {
	repo := NewInmemRepo()
	require.NoError(t, repo.SaveNFT(&NFT{ID: "a", Name: "first write", Owner: []byte("o")}))
	require.NoError(t, repo.SaveNFT(&NFT{ID: "b", Name: "second write", Owner: []byte("o")}))

	// The SQLite UPDATE keeps the rowid, so an edited asset does not become
	// the newest one.
	require.NoError(t, repo.UpdateNFT(&NFT{ID: "a", Name: "edited", Owner: []byte("o")}))

	all, err := repo.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"b", "a"}, nftIDs(all))
	require.Equal(t, "edited", all[1].Name, "the edit is what the row now holds")
}

// TestInmemRepo_SaveIsInsertOrReplace covers the other half of the SQLite
// behaviour: SaveNFT is INSERT OR REPLACE, which allocates a new rowid, so a
// re-saved asset moves to the front.
func TestInmemRepo_SaveIsInsertOrReplace(t *testing.T) {
	repo := NewInmemRepo()
	require.NoError(t, repo.SaveNFT(&NFT{ID: "a", Owner: []byte("o")}))
	require.NoError(t, repo.SaveNFT(&NFT{ID: "b", Owner: []byte("o")}))
	require.NoError(t, repo.SaveNFT(&NFT{ID: "a", Name: "rewritten", Owner: []byte("o")}))

	all, err := repo.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, nftIDs(all),
		"INSERT OR REPLACE gives the rewritten row the newest position")
	require.Equal(t, "rewritten", all[0].Name)
}

// errTxAborted is the failure the mock transaction manager rolls back on.
var errTxAborted = errors.New("callback aborted")

func nftIDs(rows []*NFT) []string {
	out := make([]string, len(rows))
	for i, n := range rows {
		out[i] = n.ID
	}
	return out
}

// TestInmemRepo_RollbackRestoresTheWriteOrder covers the snapshot hooks the
// service tests drive. A rolled-back re-save must leave the listing order
// exactly as it was: the row survives the rollback, so a stale write position
// would promote it to the front for reasons the caller never observed.
func TestInmemRepo_RollbackRestoresTheWriteOrder(t *testing.T) {
	inner := NewInmemRepo().(*inmemRepo)
	require.NoError(t, inner.SaveNFT(&NFT{ID: "a", Owner: []byte("o")}))
	require.NoError(t, inner.SaveNFT(&NFT{ID: "b", Owner: []byte("o")}))

	before, err := inner.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"b", "a"}, nftIDs(before))

	tx := newMockTxManagerWithRepo(inner)
	require.ErrorIs(t, tx.WithTransaction(func(*sql.Tx) error {
		// Re-saving an existing row rewrites its position inside the
		// transaction, which the rollback has to undo along with the fields.
		if err := inner.SaveNFT(&NFT{ID: "a", Name: "rewritten", Owner: []byte("o")}); err != nil {
			return err
		}
		// Fail after the write, so the rollback runs on state that changed.
		return errTxAborted
	}), errTxAborted)

	after, err := inner.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, nftIDs(before), nftIDs(after),
		"a rolled-back re-save must not change the listing order")

	// The next real write must still append ahead of everything.
	require.NoError(t, inner.SaveNFT(&NFT{ID: "c", Owner: []byte("o")}))
	final, err := inner.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"c", "b", "a"}, nftIDs(final))
}
