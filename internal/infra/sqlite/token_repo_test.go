package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
)

func setupTokenTestDB(t *testing.T) (*TokenRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_token.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create token repository: %v", err)
	}

	cleanup := func() {
		if repo != nil {
			_ = repo.Close()
		}
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewTokenRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
	if repo.db == nil {
		t.Fatal("Database connection should not be nil")
	}
}

func TestTokenRepository_SaveToken(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	testToken := token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), token.PublicKey([]byte("owner")))

	err := repo.SaveToken(testToken)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}
}

func TestTokenRepository_GetToken(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	testToken := token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), token.PublicKey([]byte("owner")))

	err := repo.SaveToken(testToken)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	retrieved, err := repo.GetToken("TEST")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Token should not be nil")
	}

	if retrieved.ID() != "TEST" {
		t.Errorf("Expected token ID 'TEST', got '%s'", retrieved.ID())
	}

	if retrieved.Name() != "Test Token" {
		t.Errorf("Expected token name 'Test Token', got '%s'", retrieved.Name())
	}
}

func TestTokenRepository_GetToken_NotFound(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	_, err := repo.GetToken("NOTEXIST")
	if err != token.ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestTokenRepository_SaveApproval(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))
	approval := token.NewApproval("TEST", owner, spender, token.NewAmount(500))

	err := repo.SaveApproval(approval)
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}
}

func TestTokenRepository_GetApproval(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))
	approval := token.NewApproval("TEST", owner, spender, token.NewAmount(500))

	err := repo.SaveApproval(approval)
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	retrieved, err := repo.GetApproval("TEST", owner, spender)
	if err != nil {
		t.Fatalf("Failed to get approval: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Approval should not be nil")
	}

	if retrieved.Amount().String() != "500" {
		t.Errorf("Expected amount '500', got '%s'", retrieved.Amount().String())
	}
}

func TestTokenRepository_GetApproval_NotFound(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))

	retrieved, err := repo.GetApproval("TEST", owner, spender)
	if err != nil {
		t.Fatalf("Failed to get approval: %v", err)
	}

	if retrieved != nil {
		t.Error("Approval should be nil for non-existent approval")
	}
}

func TestTokenRepository_GetApprovalsByOwner(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender1 := token.PublicKey([]byte("spender1"))
	spender2 := token.PublicKey([]byte("spender2"))

	err := repo.SaveApproval(token.NewApproval("TEST", owner, spender1, token.NewAmount(100)))
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	err = repo.SaveApproval(token.NewApproval("TEST2", owner, spender2, token.NewAmount(200)))
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	approvals, err := repo.GetApprovalsByOwner("TEST", owner)
	if err != nil {
		t.Fatalf("Failed to get approvals: %v", err)
	}

	if len(approvals) != 1 {
		t.Errorf("Expected 1 approval, got %d", len(approvals))
	}
}

func TestTokenRepository_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Double close should not fail: %v", err)
	}
}
