package deskapp

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/stretchr/testify/require"
)

// nftReadServices builds the adapter over an in-memory NFT repository and a
// real chain. The reads under test are field mappings, so the store only has to
// hold what a mint wrote.
func nftReadServices(t *testing.T) (*nativeServices, string, string) {
	t.Helper()
	services := &nativeServices{nftRepo: nft.NewInmemRepo(), chain: blockchain.New()}
	// The creator is a real key: the mint validates its encoding, so a stand-in
	// string would fail before the read path is reached.
	creator, _, err := services.GenerateKeyPair()
	require.NoError(t, err)
	row, err := services.MintNFT("Star Certificate", "a described asset",
		"https://example.test/star.png", "https://example.test/star.json", creator)
	require.NoError(t, err)
	return services, row.ID, creator
}

// TestNFTReadsCarryEveryField pins the read adapter the Inspect card uses. A
// dropped field here is invisible until an operator compares the window with
// `aurora nft get`: the card simply shows nothing where the store has a value.
func TestNFTReadsCarryEveryField(t *testing.T) {
	services, id, creator := nftReadServices(t)

	row, err := services.GetNFT(id)
	require.NoError(t, err)
	require.Equal(t, id, row.ID)
	require.Equal(t, "Star Certificate", row.Name)
	require.Equal(t, "a described asset", row.Description)
	require.Equal(t, creator, row.Owner, "the minter becomes the first owner")
	require.Equal(t, creator, row.Creator)
	require.Equal(t, "https://example.test/star.png", row.ImageURL)
	require.Equal(t, "https://example.test/star.json", row.TokenURI)
	require.Greater(t, row.BlockHeight, int64(0), "a minted asset carries the block it was recorded at")

	listed, err := services.ListNFTs(creator)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, id, listed[0].ID)
}

// TestNFTReadsSeparateOwners covers the filter the Collection card uses: an
// owner with no assets must read as empty rather than as the whole collection,
// and the owner who minted must see the asset. The use case requires a real
// key, so both queries use generated ones — an empty owner is a usage error
// rather than a request for everything.
func TestNFTReadsSeparateOwners(t *testing.T) {
	services, _, creator := nftReadServices(t)

	// Another real key, so the empty result means "no assets for this owner"
	// rather than "the key did not parse".
	stranger, _, err := services.GenerateKeyPair()
	require.NoError(t, err)
	other, err := services.ListNFTs(stranger)
	require.NoError(t, err)
	require.Empty(t, other, "another owner's query must not return this asset")

	mine, err := services.ListNFTs(creator)
	require.NoError(t, err)
	require.Len(t, mine, 1, "the owner who minted it must see it")

}

// TestNFTReadReportsAMissingID keeps the failure path honest: the Inspect card
// must show the store's reason, not an empty record.
func TestNFTReadReportsAMissingID(t *testing.T) {
	services, _, _ := nftReadServices(t)

	_, err := services.GetNFT("no-such-id")
	require.Error(t, err)
}
