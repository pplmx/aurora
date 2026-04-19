package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/stretchr/testify/require"
)

func setupBlockchainTestDB(t *testing.T) (*BlockchainRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_blockchain.db")

	repo, err := NewBlockchainRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create blockchain repository: %v", err)
	}

	cleanup := func() {
		if repo != nil {
			_ = repo.Close()
		}
		_ = os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewBlockchainRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewBlockchainRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
}

func TestBlockchainRepository_SaveBlock(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	block := blockchain.NewBlock("test data", []byte{})
	block.Height = 1

	err := repo.SaveBlock(1, block)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}
}

func TestBlockchainRepository_GetBlock(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	block := blockchain.NewBlock("test data", []byte{})
	block.Height = 1

	err := repo.SaveBlock(1, block)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	retrieved, err := repo.GetBlock(1)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	if string(retrieved.Data) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(retrieved.Data))
	}
}

func TestBlockchainRepository_GetBlock_NotFound(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	_, err := repo.GetBlock(999)
	if err == nil {
		t.Error("Expected error for non-existent block")
	}
}

func TestBlockchainRepository_GetAllBlocks(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	block1 := blockchain.NewBlock("data1", []byte{})
	block1.Height = 1

	block2 := blockchain.NewBlock("data2", block1.Hash)
	block2.Height = 2

	err := repo.SaveBlock(1, block1)
	if err != nil {
		t.Fatalf("Failed to save block1: %v", err)
	}

	err = repo.SaveBlock(2, block2)
	if err != nil {
		t.Fatalf("Failed to save block2: %v", err)
	}

	blocks, err := repo.GetAllBlocks()
	if err != nil {
		t.Fatalf("Failed to get all blocks: %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}
}

func TestBlockchainRepository_GetLotteryRecords(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	block1 := blockchain.NewBlock("record1", []byte{})
	block1.Height = 1

	block2 := blockchain.NewBlock("record2", block1.Hash)
	block2.Height = 2

	err := repo.SaveBlock(1, block1)
	if err != nil {
		t.Fatalf("Failed to save block1: %v", err)
	}

	err = repo.SaveBlock(2, block2)
	if err != nil {
		t.Fatalf("Failed to save block2: %v", err)
	}

	records, err := repo.GetLotteryRecords()
	if err != nil {
		t.Fatalf("Failed to get lottery records: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 lottery records, got %d", len(records))
	}
}

func TestBlockchainRepository_AddLotteryRecord(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	height, err := repo.AddLotteryRecord("test lottery record")
	if err != nil {
		t.Fatalf("Failed to add lottery record: %v", err)
	}

	if height != 1 {
		t.Errorf("Expected height 1, got %d", height)
	}

	records, err := repo.GetLotteryRecords()
	if err != nil {
		t.Fatalf("Failed to get lottery records: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}
}

func TestBlockchainRepository_Chain(t *testing.T) {
	repo, cleanup := setupBlockchainTestDB(t)
	defer cleanup()

	chain := repo.Chain()
	require.NotNil(t, chain)

	if len(chain.Blocks) == 0 {
		t.Error("Chain should have at least genesis block")
	}
}

func TestBlockchainRepository_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewBlockchainRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}
}
