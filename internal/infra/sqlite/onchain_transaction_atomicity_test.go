package sqlite

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/stretchr/testify/require"
)

func countOnChainBlocks(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&count))
	return count
}

func TestTokenTransferRollbackLeavesNoOnChainBlock(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	repo, err := NewTokenRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	chain := blockchain.NewBlockChainWithDB(repo.GetDB())
	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = replay.Close() })

	service := token.NewService(
		repo,
		NewTxManager(repo.GetDB()),
		infraevents.NewSyncEventBus(),
		nil,
		replay,
		chain,
	)

	ownerPublic, ownerPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	recipientPublic, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	require.NoError(t, repo.SaveToken(token.NewToken(
		"TEST",
		"Test Token",
		"TEST",
		token.NewAmount(1000),
		token.PublicKey(ownerPublic),
	)))

	_, err = service.Transfer(&token.TransferRequest{
		TokenID:    "TEST",
		From:       token.PublicKey(ownerPublic),
		To:         token.PublicKey(recipientPublic),
		Amount:     token.NewAmount(1),
		PrivateKey: token.PrivateKey(ownerPrivate),
	})
	require.ErrorIs(t, err, token.ErrInsufficientBalance)
	require.Len(t, chain.Blocks, 1, "failed transfer must not remain in the in-memory chain")
	require.Zero(t, countOnChainBlocks(t, repo.GetDB()), "failed transfer must not persist a block")
}

func TestNFTTransferRollbackLeavesNoOnChainBlock(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	repo, err := NewNFTRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	chain := blockchain.NewBlockChainWithDB(repo.GetDB())
	service := nft.NewService(repo, NewTxManager(repo.GetDB()))

	ownerPublic, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	attackerPublic, attackerPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	recipientPublic, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	item := nft.NewNFT("Item", "test NFT", "", "", ownerPublic, ownerPublic)
	minted, err := service.Mint(item, chain)
	require.NoError(t, err)
	require.Len(t, chain.Blocks, 2)
	require.Equal(t, 1, countOnChainBlocks(t, repo.GetDB()))

	_, err = service.Transfer(minted.ID, attackerPublic, recipientPublic, attackerPrivate, chain)
	require.ErrorIs(t, err, nft.ErrNotOwner)
	require.Len(t, chain.Blocks, 2, "failed transfer must not remain in the in-memory chain")
	require.Equal(t, 1, countOnChainBlocks(t, repo.GetDB()), "failed transfer must not persist a second block")

	operations, err := repo.GetOperations(minted.ID)
	require.NoError(t, err)
	require.Len(t, operations, 1, "the rolled-back transfer operation must not survive")
	require.Equal(t, "mint", operations[0].Type)
}
