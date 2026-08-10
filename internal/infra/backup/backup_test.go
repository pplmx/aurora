package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestBackupService_Create(t *testing.T) {
	if err := os.MkdirAll("/tmp/test_backup_src", 0755); err != nil {
		t.Fatalf("Create test dir: %v", err)
	}
	defer func() { _ = os.RemoveAll("/tmp/test_backup_src") }()

	db, err := sql.Open("sqlite3", "/tmp/test_backup_src/blockchain.db")
	if err != nil {
		t.Fatalf("Create test db: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("Create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close db: %v", err)
	}

	svc := NewBackupService(map[string]string{
		"blockchain": "/tmp/test_backup_src/blockchain.db",
	})

	result, err := svc.Create(context.Background(), "/tmp/test_backup_out")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if result.File != "/tmp/test_backup_out" {
		t.Errorf("Expected file /tmp/test_backup_out, got %s", result.File)
	}

	if result.Size == 0 {
		t.Error("Expected non-zero size")
	}

	if result.Checksum == "" {
		t.Error("Expected non-empty checksum")
	}

	_ = os.RemoveAll("/tmp/test_backup_src")
	_ = os.RemoveAll("/tmp/test_backup_out")
}

// TestBackupService_Create_OutputPathIsFile proves Create fails cleanly when
// the requested output path already exists as a regular file (so MkdirAll
// cannot create the directory).
func TestBackupService_Create_OutputPathIsFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0755))

	// A real DB so the failure we assert is the output-path one, not an
	// earlier step.
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	output := filepath.Join(dir, "out")
	require.NoError(t, os.WriteFile(output, []byte("occupied"), 0644))

	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err = svc.Create(context.Background(), output)
	require.Error(t, err, "Create must fail when the output path is a regular file")
}

// TestBackupService_Create_MissingDB proves Create surfaces a DB that cannot
// be opened (path inside a nonexistent directory) instead of silently skipping
// it.
func TestBackupService_Create_MissingDB(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out")

	svc := NewBackupService(map[string]string{
		"missing": filepath.Join(dir, "no-such-dir", "blockchain.db"),
	})
	_, err := svc.Create(context.Background(), output)
	require.Error(t, err, "Create must fail when a configured DB cannot be opened")
}

func TestBackupService_Verify(t *testing.T) {
	if err := os.MkdirAll("/tmp/test_verify_src", 0755); err != nil {
		t.Fatalf("Create test dir: %v", err)
	}
	defer func() { _ = os.RemoveAll("/tmp/test_verify_src") }()

	db, err := sql.Open("sqlite3", "/tmp/test_verify_src/blockchain.db")
	if err != nil {
		t.Fatalf("Create test db: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("Create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close db: %v", err)
	}

	svc := NewBackupService(map[string]string{
		"blockchain": "/tmp/test_verify_src/blockchain.db",
	})

	_, err = svc.Create(context.Background(), "/tmp/test_verify_out")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Verify(context.Background(), "/tmp/test_verify_out"); err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	_ = os.RemoveAll("/tmp/test_verify_src")
	_ = os.RemoveAll("/tmp/test_verify_out")
}

func TestBackupService_VerifyInvalidPath(t *testing.T) {
	svc := NewBackupService(nil)

	err := svc.Verify(context.Background(), "/nonexistent/dir")
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestBackupService_VerifyCorruptMetadata(t *testing.T) {
	if err := os.MkdirAll("/tmp/test_corrupt", 0755); err != nil {
		t.Fatalf("Create test dir: %v", err)
	}
	defer func() { _ = os.RemoveAll("/tmp/test_corrupt") }()

	db, err := sql.Open("sqlite3", "/tmp/test_corrupt/blockchain.db")
	if err != nil {
		t.Fatalf("Create test db: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("Create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close db: %v", err)
	}

	if err := os.WriteFile("/tmp/test_corrupt/metadata.json", []byte("not json"), 0644); err != nil {
		t.Fatalf("Write metadata: %v", err)
	}

	svc := NewBackupService(nil)
	err = svc.Verify(context.Background(), "/tmp/test_corrupt")
	if err == nil {
		t.Error("Expected error for corrupt metadata")
	}
}

func TestBackupService_Restore(t *testing.T) {
	if err := os.MkdirAll("/tmp/test_restore_backup", 0755); err != nil {
		t.Fatalf("Create test dir: %v", err)
	}
	defer func() { _ = os.RemoveAll("/tmp/test_restore_backup") }()

	db, err := sql.Open("sqlite3", "/tmp/test_restore_backup/blockchain.db")
	if err != nil {
		t.Fatalf("Create test db: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("Create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close db: %v", err)
	}

	metadata := `{"version":"1.2","timestamp":"2026-04-30T00:00:00Z","checksum":"","databases":["blockchain"],"schema_version":1}`
	if err := os.WriteFile("/tmp/test_restore_backup/metadata.json", []byte(metadata), 0644); err != nil {
		t.Fatalf("Write metadata: %v", err)
	}

	if err := os.MkdirAll("/tmp/test_restore_dest", 0755); err != nil {
		t.Fatalf("Create dest dir: %v", err)
	}
	defer func() { _ = os.RemoveAll("/tmp/test_restore_dest") }()

	svc := NewBackupService(map[string]string{
		"blockchain": "/tmp/test_restore_dest/blockchain.db",
	})

	err = svc.Restore(context.Background(), "/tmp/test_restore_backup")
	if err != nil {
		t.Errorf("Restore failed: %v", err)
	}

	if _, err := os.Stat("/tmp/test_restore_dest/blockchain.db"); os.IsNotExist(err) {
		t.Error("Expected restored database to exist")
	}
}
