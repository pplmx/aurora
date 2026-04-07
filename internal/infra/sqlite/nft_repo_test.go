package sqlite

import (
	"os"
	"path/filepath"
	"testing"

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
		os.RemoveAll(tmpDir)
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
	if err != nil {
		t.Fatalf("Failed to get NFT: %v", err)
	}

	if retrieved == nil {
		t.Fatal("NFT should not be nil")
	}

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
