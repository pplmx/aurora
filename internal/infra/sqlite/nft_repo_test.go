package sqlite

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/nft"
)

func setupNFTTestDB(t *testing.T) (*NFTRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_nft.db")

	repo, err := NewNFTRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create NFT repository: %v", err)
	}

	cleanup := func() {
		if repo != nil {
			_ = repo.Close()
		}
		_ = os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewNFTRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewNFTRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
}

func TestNFTRepository_SaveNFT(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	nft := &nft.NFT{
		ID:          "nft-1",
		Name:        "Test NFT",
		Description: "A test NFT",
		ImageURL:    "https://example.com/nft.png",
		TokenURI:    "ipfs://QmTest",
		Owner:       []byte("owner"),
		Creator:     []byte("creator"),
		BlockHeight: 1,
		Timestamp:   1234567890,
	}

	err := repo.SaveNFT(nft)
	if err != nil {
		t.Fatalf("Failed to save NFT: %v", err)
	}
}

func TestNFTRepository_GetNFT(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	nft := &nft.NFT{
		ID:          "nft-1",
		Name:        "Test NFT",
		Description: "A test NFT",
		ImageURL:    "https://example.com/nft.png",
		TokenURI:    "ipfs://QmTest",
		Owner:       []byte("owner"),
		Creator:     []byte("creator"),
		BlockHeight: 1,
		Timestamp:   1234567890,
	}

	err := repo.SaveNFT(nft)
	if err != nil {
		t.Fatalf("Failed to save NFT: %v", err)
	}

	retrieved, err := repo.GetNFT("nft-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	if retrieved.ID != "nft-1" {
		t.Errorf("Expected ID 'nft-1', got '%s'", retrieved.ID)
	}

	if retrieved.Name != "Test NFT" {
		t.Errorf("Expected name 'Test NFT', got '%s'", retrieved.Name)
	}
}

func TestNFTRepository_GetNFT_NotFound(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	_, err := repo.GetNFT("NOTEXIST")
	if err != nil {
		t.Fatalf("Expected nil for non-existent NFT, got error: %v", err)
	}
}

func TestNFTRepository_GetNFTsByOwner(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	owner := []byte("testowner")

	nft1 := &nft.NFT{
		ID:    "nft-1",
		Name:  "NFT 1",
		Owner: owner,
	}
	nft2 := &nft.NFT{
		ID:    "nft-2",
		Name:  "NFT 2",
		Owner: owner,
	}
	nft3 := &nft.NFT{
		ID:    "nft-3",
		Name:  "NFT 3",
		Owner: []byte("other"),
	}

	err := repo.SaveNFT(nft1)
	if err != nil {
		t.Fatalf("Failed to save NFT1: %v", err)
	}
	err = repo.SaveNFT(nft2)
	if err != nil {
		t.Fatalf("Failed to save NFT2: %v", err)
	}
	err = repo.SaveNFT(nft3)
	if err != nil {
		t.Fatalf("Failed to save NFT3: %v", err)
	}

	nfts, err := repo.GetNFTsByOwner(owner)
	if err != nil {
		t.Fatalf("Failed to get NFTs by owner: %v", err)
	}

	if len(nfts) != 2 {
		t.Errorf("Expected 2 NFTs, got %d", len(nfts))
	}
}

func TestNFTRepository_GetNFTsByCreator(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	creator := []byte("testcreator")

	nft1 := &nft.NFT{
		ID:      "nft-1",
		Name:    "NFT 1",
		Creator: creator,
	}
	nft2 := &nft.NFT{
		ID:      "nft-2",
		Name:    "NFT 2",
		Creator: creator,
	}
	nft3 := &nft.NFT{
		ID:      "nft-3",
		Name:    "NFT 3",
		Creator: []byte("other"),
	}

	err := repo.SaveNFT(nft1)
	if err != nil {
		t.Fatalf("Failed to save NFT1: %v", err)
	}
	err = repo.SaveNFT(nft2)
	if err != nil {
		t.Fatalf("Failed to save NFT2: %v", err)
	}
	err = repo.SaveNFT(nft3)
	if err != nil {
		t.Fatalf("Failed to save NFT3: %v", err)
	}

	nfts, err := repo.GetNFTsByCreator(creator)
	if err != nil {
		t.Fatalf("Failed to get NFTs by creator: %v", err)
	}

	if len(nfts) != 2 {
		t.Errorf("Expected 2 NFTs, got %d", len(nfts))
	}
}

func TestNFTRepository_SaveOperation(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	nft1 := &nft.NFT{
		ID:    "nft-1",
		Name:  "Test NFT",
		Owner: []byte("owner"),
	}
	err := repo.SaveNFT(nft1)
	if err != nil {
		t.Fatalf("Failed to save NFT: %v", err)
	}

	op := &nft.Operation{
		ID:          "op-1",
		NFTID:       "nft-1",
		Type:        "transfer",
		From:        []byte("from"),
		To:          []byte("to"),
		Signature:   []byte("signature"),
		BlockHeight: 1,
		Timestamp:   1234567890,
	}

	err = repo.SaveOperation(op)
	if err != nil {
		t.Fatalf("Failed to save operation: %v", err)
	}
}

func TestNFTRepository_GetOperations(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	nft1 := &nft.NFT{
		ID:    "nft-1",
		Name:  "Test NFT 1",
		Owner: []byte("owner"),
	}
	nft2 := &nft.NFT{
		ID:    "nft-2",
		Name:  "Test NFT 2",
		Owner: []byte("owner"),
	}
	err := repo.SaveNFT(nft1)
	if err != nil {
		t.Fatalf("Failed to save NFT1: %v", err)
	}
	err = repo.SaveNFT(nft2)
	if err != nil {
		t.Fatalf("Failed to save NFT2: %v", err)
	}

	op1 := &nft.Operation{
		ID:    "op-1",
		NFTID: "nft-1",
		Type:  "transfer",
		From:  []byte("from"),
		To:    []byte("to"),
	}
	op2 := &nft.Operation{
		ID:    "op-2",
		NFTID: "nft-1",
		Type:  "transfer",
		From:  []byte("to"),
		To:    []byte("new"),
	}
	op3 := &nft.Operation{
		ID:    "op-3",
		NFTID: "nft-2",
		Type:  "transfer",
		From:  []byte("from"),
		To:    []byte("to"),
	}

	err = repo.SaveOperation(op1)
	if err != nil {
		t.Fatalf("Failed to save op1: %v", err)
	}
	err = repo.SaveOperation(op2)
	if err != nil {
		t.Fatalf("Failed to save op2: %v", err)
	}
	err = repo.SaveOperation(op3)
	if err != nil {
		t.Fatalf("Failed to save op3: %v", err)
	}

	ops, err := repo.GetOperations("nft-1")
	if err != nil {
		t.Fatalf("Failed to get operations: %v", err)
	}

	if len(ops) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(ops))
	}
}

func TestNFTRepository_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewNFTRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}
}

func TestNFTRepository_UpdateNFT(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := &nft.NFT{
		ID:          "nft-1",
		Name:        "Original",
		Description: "desc",
		ImageURL:    "https://example.com/1.png",
		TokenURI:    "ipfs://Qm1",
		Owner:       []byte("owner1"),
		Creator:     []byte("creator1"),
		BlockHeight: 1,
		Timestamp:   100,
	}
	require.NoError(t, repo.SaveNFT(n))

	n.Name = "Updated"
	n.Description = "new desc"
	require.NoError(t, repo.UpdateNFT(n))

	retrieved, err := repo.GetNFT("nft-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "Updated", retrieved.Name)
	require.Equal(t, "new desc", retrieved.Description)
}

func TestNFTRepository_DeleteNFT(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveNFT(&nft.NFT{
		ID: "nft-1", Name: "Test", Owner: []byte("owner1"), Creator: []byte("creator1"),
	}))

	require.NoError(t, repo.DeleteNFT("nft-1"))

	retrieved, err := repo.GetNFT("nft-1")
	require.NoError(t, err)
	require.Nil(t, retrieved, "NFT should be deleted")
}

func TestNFTRepository_DeleteNFT_AlreadyDeleted(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	require.NoError(t, repo.DeleteNFT("nft-1"))
	require.NoError(t, repo.DeleteNFT("nft-1"))
}

func testNFT(repo *NFTRepository, t *testing.T) *nft.NFT {
	t.Helper()
	n := &nft.NFT{
		ID:          "nft-1",
		Name:        "Test NFT",
		Description: "A test NFT",
		ImageURL:    "https://example.com/nft.png",
		TokenURI:    "ipfs://QmTest",
		Owner:       []byte("owner1"),
		Creator:     []byte("creator1"),
		BlockHeight: 1,
		Timestamp:   1234567890,
	}
	require.NoError(t, repo.SaveNFT(n))
	return n
}

func TestNFTRepository_TryTransferOwnership_Success(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)
	from := n.Owner
	to := []byte("newowner")

	err := repo.TryTransferOwnership(n.ID, from, to)
	require.NoError(t, err)

	retrieved, err := repo.GetNFT(n.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.True(t, retrieved.IsOwner(to), "owner should be updated to %q", to)
}

func TestNFTRepository_TryTransferOwnership_OwnershipChanged(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)

	staleOwner := []byte("staleowner")
	wrongOwner := []byte("wrongowner")

	err := repo.TryTransferOwnership(n.ID, staleOwner, wrongOwner)
	require.ErrorIs(t, err, nft.ErrOwnershipChanged)

	retrieved, err := repo.GetNFT(n.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.True(t, retrieved.IsOwner(n.Owner), "owner must not change on failed transfer")
}

func TestNFTRepository_TryTransferOwnership_NFTNotFound(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	err := repo.TryTransferOwnership("nonexistent", []byte("a"), []byte("b"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestNFTRepository_TryDeleteNFTIfOwned_Success(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)

	err := repo.TryDeleteNFTIfOwned(n.ID, n.Owner)
	require.NoError(t, err)

	retrieved, err := repo.GetNFT(n.ID)
	require.NoError(t, err)
	require.Nil(t, retrieved, "NFT should be deleted")
}

func TestNFTRepository_TryDeleteNFTIfOwned_OwnershipChanged(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)

	staleOwner := []byte("staleowner")
	err := repo.TryDeleteNFTIfOwned(n.ID, staleOwner)
	require.ErrorIs(t, err, nft.ErrOwnershipChanged)

	retrieved, err := repo.GetNFT(n.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved, "NFT must not be deleted on failed TryDelete")
	require.True(t, retrieved.IsOwner(n.Owner), "owner must not change on failed TryDelete")
}

func TestNFTRepository_TryDeleteNFTIfOwned_NFTNotFound(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	err := repo.TryDeleteNFTIfOwned("nonexistent", []byte("a"))
	require.ErrorIs(t, err, nft.ErrNFTNotFound)
}

func TestNFTRepository_TryTransferOwnership_ConcurrentOnlyOneWinner(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)

	originalOwner := make([]byte, len(n.Owner))
	copy(originalOwner, n.Owner)

	type result struct {
		to    []byte
		err   error
		owner []byte
	}
	results := make([]result, 8)
	var wg sync.WaitGroup
	wg.Add(len(results))
	for i := range results {
		to := []byte("buyer" + string(rune('A'+i)))
		go func(idx int, to []byte) {
			defer wg.Done()
			err := repo.TryTransferOwnership(n.ID, originalOwner, to)
			retrieved, _ := repo.GetNFT(n.ID)
			ownerB64 := ""
			if retrieved != nil {
				ownerB64 = base64.StdEncoding.EncodeToString(retrieved.Owner)
			}
			results[idx] = result{to: to, err: err, owner: []byte(ownerB64)}
		}(i, to)
	}
	wg.Wait()

	successCount := 0
	for i := range results {
		if results[i].err == nil {
			successCount++
		}
	}
	require.Equal(t, 1, successCount, "exactly one concurrent transfer should succeed")

	for i := range results {
		if results[i].err == nil {
			require.True(t, base64.StdEncoding.EncodeToString(results[i].to) == string(results[i].owner),
				"winner's recipient must match stored owner")
		} else {
			require.ErrorIs(t, results[i].err, nft.ErrOwnershipChanged,
				"loser should get ErrOwnershipChanged")
		}
	}
}

func TestNFTRepository_TryDeleteNFTIfOwned_ConcurrentBurnVsTransfer(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)
	originalOwner := make([]byte, len(n.Owner))
	copy(originalOwner, n.Owner)

	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errorsCh <- repo.TryTransferOwnership(n.ID, originalOwner, []byte("newowner"))
	}()

	go func() {
		defer wg.Done()
		errorsCh <- repo.TryDeleteNFTIfOwned(n.ID, originalOwner)
	}()
	wg.Wait()
	close(errorsCh)

	successCount := 0
	for err := range errorsCh {
		if err == nil {
			successCount++
		}
	}

	require.Equal(t, 1, successCount, "exactly one of burn or transfer should succeed")

	retrieved, err := repo.GetNFT(n.ID)
	require.NoError(t, err)
	if successCount == 1 {
		if retrieved == nil {
			// burn won
			require.True(t, true, "burn succeeded, NFT deleted")
		} else {
			// transfer won
			require.True(t, retrieved.IsOwner([]byte("newowner")), "transfer succeeded, owner updated")
		}
	} else {
		t.Fatal("exactly one operation should succeed, both failed or both succeeded")
	}
}

func TestNFTRepository_TryDeleteNFTIfOwned_ConcurrentOnlyOneBurnWinner(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	n := testNFT(repo, t)
	originalOwner := make([]byte, len(n.Owner))
	copy(originalOwner, n.Owner)

	errorsCh := make(chan error, 16)
	var wg sync.WaitGroup
	wg.Add(16)
	for i := 0; i < 16; i++ {
		go func() {
			defer wg.Done()
			errorsCh <- repo.TryDeleteNFTIfOwned(n.ID, originalOwner)
		}()
	}
	wg.Wait()
	close(errorsCh)

	successCount := 0
	var losers []error
	for err := range errorsCh {
		if err == nil {
			successCount++
		} else {
			losers = append(losers, err)
		}
	}

	require.Equal(t, 1, successCount, "exactly one concurrent burn should succeed")
	for _, err := range losers {
		require.True(t, errors.Is(err, nft.ErrOwnershipChanged) || errors.Is(err, nft.ErrNFTNotFound),
			"losers should get either ErrOwnershipChanged (owner mismatch) or "+
				"ErrNFTNotFound (winner already deleted the NFT), got: %v", err)
	}
}
