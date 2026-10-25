package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/token"
)

// The listing paths are what let an operator (or a caller that never minted
// the asset) find a stored row again. They were added with their callers and
// covered only through the app layer, so a regression here — a lost column, a
// wrong ORDER BY — would surface as a silently empty list.

func TestTokenRepository_ListTokens(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	// An empty store lists nothing, and does so without an error: the desktop
	// page draws an empty state from this.
	empty, err := repo.ListTokens()
	require.NoError(t, err)
	assert.Empty(t, empty)

	first := token.NewToken("AAA", "First", "AAA", token.NewAmount(10), token.PublicKey([]byte("owner-a")))
	second := token.NewToken("BBB", "Second", "BBB", token.NewAmount(20), token.PublicKey([]byte("owner-b")))
	require.NoError(t, repo.SaveToken(first))
	require.NoError(t, repo.SaveToken(second))

	listed, err := repo.ListTokens()
	require.NoError(t, err)
	require.Len(t, listed, 2, "both tokens must be listed")

	// A listed token carries what a fetched one does: the list feeds the same
	// cards a lookup does.
	byID := map[string]*token.Token{}
	for _, tk := range listed {
		byID[string(tk.ID())] = tk
	}
	require.Contains(t, byID, "AAA")
	require.Contains(t, byID, "BBB")

	fetched, err := repo.GetToken("AAA")
	require.NoError(t, err)
	assert.Equal(t, fetched.Name(), byID["AAA"].Name())
	assert.Equal(t, fetched.Symbol(), byID["AAA"].Symbol())
	assert.Equal(t, fetched.TotalSupply().String(), byID["AAA"].TotalSupply().String())
	assert.Equal(t, fetched.Owner(), byID["AAA"].Owner())
	assert.Equal(t, fetched.IsMintable(), byID["AAA"].IsMintable())
	assert.Equal(t, fetched.CreatedAt().Unix(), byID["AAA"].CreatedAt().Unix(),
		"a listed token must keep its creation time, not the time it was read")
}

func TestNFTRepository_ListNFTs(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	empty, err := repo.ListNFTs(0, 0)
	require.NoError(t, err)
	assert.Empty(t, empty)

	owner := []byte("owner-key-bytes")
	creator := []byte("creator-key-bytes")
	first := &nft.NFT{ID: "n1", Name: "First Art", Owner: owner, Creator: creator, BlockHeight: 1}
	second := &nft.NFT{ID: "n2", Name: "Second Art", Owner: owner, Creator: creator, BlockHeight: 2}
	require.NoError(t, repo.SaveNFT(first))
	require.NoError(t, repo.SaveNFT(second))

	listed, err := repo.ListNFTs(0, 0)
	require.NoError(t, err)
	require.Len(t, listed, 2, "every asset must be listed, whatever its owner")

	// Newest first, so the asset an operator just minted is at the top.
	assert.Equal(t, "n2", listed[0].ID)
	assert.Equal(t, "n1", listed[1].ID)

	// The scan matches GetNFTsByOwner's, so a listed asset carries the same
	// fields a fetched one does.
	fetched, err := repo.GetNFT("n1")
	require.NoError(t, err)
	assert.Equal(t, fetched.Name, listed[1].Name)
	assert.Equal(t, fetched.Owner, listed[1].Owner)
	assert.Equal(t, fetched.Creator, listed[1].Creator)
	assert.Equal(t, fetched.BlockHeight, listed[1].BlockHeight)
}

func TestNFTRepository_ListNFTsPages(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	owner := []byte("owner-key-bytes")
	for _, id := range []string{"n1", "n2", "n3"} {
		require.NoError(t, repo.SaveNFT(&nft.NFT{ID: id, Name: id, Owner: owner, Creator: owner}))
	}

	page, err := repo.ListNFTs(1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1, "a limit of one must return one row")
	assert.Equal(t, "n2", page[0].ID, "the second row, newest first")

	// An offset past the end is an empty page, not an error.
	past, err := repo.ListNFTs(10, 99)
	require.NoError(t, err)
	assert.Empty(t, past)
}
