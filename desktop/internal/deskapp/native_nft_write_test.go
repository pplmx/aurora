package deskapp

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/stretchr/testify/require"
)

// nftWriteServices is the adapter over an in-memory NFT store and a real chain,
// with one asset already minted. The write paths sign with a real key, so the
// fixtures are generated rather than written by hand.
func nftWriteServices(t *testing.T) (services *nativeServices, id, ownerPub, ownerPriv string) {
	t.Helper()
	services = &nativeServices{nftRepo: nft.NewInmemRepo(), chain: blockchain.New()}
	ownerPub, ownerPriv, err := services.GenerateKeyPair()
	require.NoError(t, err)
	row, err := services.MintNFT("Star Certificate", "", "", "", ownerPub)
	require.NoError(t, err)
	return services, row.ID, ownerPub, ownerPriv
}

// TestNFTTransferMovesOwnership walks the write the Transfer card performs: the
// asset changes hands, the history records the operation at a block height, and
// the old owner can no longer move it.
func TestNFTTransferMovesOwnership(t *testing.T) {
	services, id, ownerPub, ownerPriv := nftWriteServices(t)
	nextPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)

	require.NoError(t, services.TransferNFT(id, ownerPub, nextPub, ownerPriv))

	moved, err := services.GetNFT(id)
	require.NoError(t, err)
	require.Equal(t, nextPub, moved.Owner, "the recipient owns the asset after the transfer")

	// The previous owner's key must no longer authorize a move.
	err = services.TransferNFT(id, ownerPub, nextPub, ownerPriv)
	require.Error(t, err, "only the current owner may transfer")
}

// TestNFTHistoryRecordsEachOperation pins the operation trail the History tab
// renders: one row per write, each carrying the type and the block it landed in.
func TestNFTHistoryRecordsEachOperation(t *testing.T) {
	services, id, ownerPub, ownerPriv := nftWriteServices(t)
	nextPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)
	require.NoError(t, services.TransferNFT(id, ownerPub, nextPub, ownerPriv))

	rows, err := services.NFTHistory(id, 0, 0)
	require.NoError(t, err)
	require.Len(t, rows, 2, "the mint and the transfer are both recorded")

	types := map[string]bool{}
	for _, row := range rows {
		types[row.Type] = true
		require.Greater(t, row.BlockHeight, int64(0), "every operation carries the block it was recorded at")
	}
	require.Truef(t, types["mint"], "the mint must appear, got %v", types)
	require.Truef(t, types["transfer"], "the transfer must appear, got %v", types)
}

// TestNFTBurnDestroysTheAsset covers the destructive path: after a burn the
// asset is gone, and the history still explains what happened to it.
func TestNFTBurnDestroysTheAsset(t *testing.T) {
	services, id, ownerPub, ownerPriv := nftWriteServices(t)

	require.NoError(t, services.BurnNFT(id, ownerPub, ownerPriv))

	_, err := services.GetNFT(id)
	require.Error(t, err, "a burned asset must not be readable")

	rows, err := services.NFTHistory(id, 0, 0)
	require.NoError(t, err)
	burned := false
	for _, row := range rows {
		if row.Type == "burn" {
			burned = true
		}
	}
	require.True(t, burned, "the burn must be in the operation trail, got %d rows", len(rows))
}

// TestNFTWritesRejectAMismatchedKey keeps the signature check honest: a key that
// belongs to someone else must not be accepted just because it parses.
func TestNFTWritesRejectAMismatchedKey(t *testing.T) {
	services, id, ownerPub, _ := nftWriteServices(t)
	_, strangerPriv, err := services.GenerateKeyPair()
	require.NoError(t, err)
	nextPub, _, err := services.GenerateKeyPair()
	require.NoError(t, err)

	err = services.TransferNFT(id, ownerPub, nextPub, strangerPriv)
	require.Error(t, err, "a key that does not belong to the sender must be refused")
}
