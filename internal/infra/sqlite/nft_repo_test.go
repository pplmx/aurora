package sqlite

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
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

	nfts, err := repo.GetNFTsByOwner(owner, 0, 0)
	if err != nil {
		t.Fatalf("Failed to get NFTs by owner: %v", err)
	}

	if len(nfts) != 2 {
		t.Errorf("Expected 2 NFTs, got %d", len(nfts))
	}
}

// TestNFTRepository_GetNFTsByOwnerPaged locks the v1.79 bounded-paging fix
// (TASK-101, ISS-093): the SQL layer must honor LIMIT/OFFSET with a stable
// insertion (rowid) order so pages don't overlap or skip, and 0,0 stays
// unbounded for the CLI/TUI.
func TestNFTRepository_GetNFTsByOwnerPaged(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	owner := []byte("pageowner")
	for i := 0; i < 5; i++ {
		err := repo.SaveNFT(&nft.NFT{ID: fmt.Sprintf("nft-%d", i), Name: fmt.Sprintf("NFT %d", i), Owner: owner})
		if err != nil {
			t.Fatalf("Failed to save NFT %d: %v", i, err)
		}
	}

	// First page of 2 returns the two earliest (insertion-order) NFTs.
	page, err := repo.GetNFTsByOwner(owner, 2, 0)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "nft-0", page[0].ID)
	require.Equal(t, "nft-1", page[1].ID)

	// Second page continues without overlap: still 2 rows, but advancing.
	page, err = repo.GetNFTsByOwner(owner, 2, 2)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "nft-2", page[0].ID)
	require.Equal(t, "nft-3", page[1].ID)

	// Offset past the end -> empty, not an error.
	page, err = repo.GetNFTsByOwner(owner, 10, 100)
	require.NoError(t, err)
	require.Len(t, page, 0)

	// A negative limit must not trip SQLite (treated as unbounded).
	page, err = repo.GetNFTsByOwner(owner, -1, 0)
	require.NoError(t, err)
	require.Len(t, page, 5)
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

// TestNFTRepository_GetOperations_ServiceCreatedOpsDoNotCollapse locks
// ISS-072 / TASK-080. The operations here are created through the real
// production path (nft.NewOperation), which previously left Operation.ID at
// "": since nft_operations.id is the PRIMARY KEY, each INSERT OR REPLACE then
// hit the same "" key and collapsed the whole per-NFT audit history to a
// single row. With NewOperation assigning a UUID, mint + transfer for one NFT
// must both survive.
func TestNFTRepository_GetOperations_ServiceCreatedOpsDoNotCollapse(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveNFT(&nft.NFT{ID: "nft-1", Name: "A", Owner: []byte("owner")}))

	op1 := nft.NewOperation("nft-1", "mint", nil, []byte("owner"), nil)
	op2 := nft.NewOperation("nft-1", "transfer", []byte("owner"), []byte("to"), []byte("sig"))
	require.NotEmpty(t, op1.ID, "NewOperation must assign a UUID id")
	require.NotEmpty(t, op2.ID, "NewOperation must assign a UUID id")
	require.NotEqual(t, op1.ID, op2.ID, "each operation must get its own UUID")

	require.NoError(t, repo.SaveOperation(op1))
	require.NoError(t, repo.SaveOperation(op2))

	ops, err := repo.GetOperations("nft-1")
	require.NoError(t, err)
	require.Len(t, ops, 2, "mint + transfer ops for one NFT must both survive the INSERT OR REPLACE")
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

// =================================================================
// WithTx — transaction-scoped repository
// =================================================================

func mintTestNFT(t *testing.T, repo *NFTRepository, id string, owner []byte) {
	t.Helper()
	require.NoError(t, repo.SaveNFT(&nft.NFT{
		ID:          id,
		Name:        "Tx NFT",
		Owner:       owner,
		Creator:     owner,
		BlockHeight: 1,
		Timestamp:   1,
	}))
}

// TestNFTRepository_WithTx_CommitAndVisibility verifies that writes through a
// tx-scoped repository are visible to reads in the same transaction before
// commit, and durable after commit.
func TestNFTRepository_WithTx_CommitAndVisibility(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	txMgr := NewTxManager(repo.GetDB())
	err := txMgr.WithTransaction(func(tx *sql.Tx) error {
		txRepo := repo.WithTx(tx)

		n := &nft.NFT{ID: "nft-tx", Name: "In Tx", Owner: []byte("alice"), Creator: []byte("alice"), BlockHeight: 1, Timestamp: 1}
		if err := txRepo.SaveNFT(n); err != nil {
			return err
		}
		op := &nft.Operation{ID: "op-tx", NFTID: "nft-tx", Type: "mint", To: []byte("alice"), BlockHeight: 1, Timestamp: 1}
		if err := txRepo.SaveOperation(op); err != nil {
			return err
		}

		// Same-tx visibility: the uncommitted rows must be readable through
		// the tx-scoped repo.
		got, err := txRepo.GetNFT("nft-tx")
		if err != nil {
			return err
		}
		require.NotNil(t, got, "tx-scoped read must see uncommitted NFT row")

		ops, err := txRepo.GetOperations("nft-tx")
		if err != nil {
			return err
		}
		require.Len(t, ops, 1, "tx-scoped read must see uncommitted operation")
		return nil
	})
	require.NoError(t, err)

	// After commit the rows are visible through the plain pool repo.
	got, err := repo.GetNFT("nft-tx")
	require.NoError(t, err)
	require.NotNil(t, got)
	ops, err := repo.GetOperations("nft-tx")
	require.NoError(t, err)
	require.Len(t, ops, 1)
}

// TestNFTRepository_WithTx_RollbackLeavesNoPartialState proves that when the
// transaction callback fails after some writes, ALL writes in the unit are
// rolled back — no orphaned NFT row, no orphaned operation. This is the
// guarantee that replaces the old best-effort DeleteNFT compensation.
func TestNFTRepository_WithTx_RollbackLeavesNoPartialState(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	boom := errors.New("mid-transaction failure")
	txMgr := NewTxManager(repo.GetDB())
	err := txMgr.WithTransaction(func(tx *sql.Tx) error {
		txRepo := repo.WithTx(tx)

		n := &nft.NFT{ID: "nft-rb", Name: "Rollback", Owner: []byte("alice"), Creator: []byte("alice"), BlockHeight: 1, Timestamp: 1}
		if err := txRepo.SaveNFT(n); err != nil {
			return err
		}
		op := &nft.Operation{ID: "op-rb", NFTID: "nft-rb", Type: "mint", To: []byte("alice"), BlockHeight: 1, Timestamp: 1}
		if err := txRepo.SaveOperation(op); err != nil {
			return err
		}
		return boom // force rollback after both writes
	})
	require.ErrorIs(t, err, boom)

	got, err := repo.GetNFT("nft-rb")
	require.NoError(t, err)
	require.Nil(t, got, "NFT row must be rolled back")

	ops, err := repo.GetOperations("nft-rb")
	require.NoError(t, err)
	require.Empty(t, ops, "operation must be rolled back")
}

// TestNFTRepository_WithTx_RollbackDoesNotTouchCommittedState proves a
// rollback only undoes the current transaction's writes, not previously
// committed rows.
func TestNFTRepository_WithTx_RollbackDoesNotTouchCommittedState(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	mintTestNFT(t, repo, "nft-committed", []byte("alice"))

	txMgr := NewTxManager(repo.GetDB())
	_ = txMgr.WithTransaction(func(tx *sql.Tx) error {
		txRepo := repo.WithTx(tx)
		if err := txRepo.TryTransferOwnership("nft-committed", []byte("alice"), []byte("bob")); err != nil {
			return err
		}
		return errors.New("abort after mutation")
	})

	got, err := repo.GetNFT("nft-committed")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("alice")),
		base64.StdEncoding.EncodeToString(got.Owner),
		"committed owner must be untouched by a later rolled-back tx")
}
