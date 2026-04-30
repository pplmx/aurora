package backup

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestBackupService_Create(t *testing.T) {
	os.MkdirAll("/tmp/test_backup_src", 0755)
	defer os.RemoveAll("/tmp/test_backup_src")

	db, err := sql.Open("sqlite3", "/tmp/test_backup_src/blockchain.db")
	if err != nil {
		t.Fatalf("Create test db: %v", err)
	}
	db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	db.Close()

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

	os.RemoveAll("/tmp/test_backup_src")
	os.RemoveAll("/tmp/test_backup_out")
}

func TestBackupService_Verify(t *testing.T) {
	os.MkdirAll("/tmp/test_verify_src", 0755)
	defer os.RemoveAll("/tmp/test_verify_src")

	db, _ := sql.Open("sqlite3", "/tmp/test_verify_src/blockchain.db")
	db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	db.Close()

	svc := NewBackupService(map[string]string{
		"blockchain": "/tmp/test_verify_src/blockchain.db",
	})

	_, err := svc.Create(context.Background(), "/tmp/test_verify_out")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Verify(context.Background(), "/tmp/test_verify_out"); err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	os.RemoveAll("/tmp/test_verify_src")
	os.RemoveAll("/tmp/test_verify_out")
}

func TestBackupService_VerifyInvalidPath(t *testing.T) {
	svc := NewBackupService(nil)

	err := svc.Verify(context.Background(), "/nonexistent/dir")
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestBackupService_VerifyCorruptMetadata(t *testing.T) {
	os.MkdirAll("/tmp/test_corrupt", 0755)
	defer os.RemoveAll("/tmp/test_corrupt")

	db, _ := sql.Open("sqlite3", "/tmp/test_corrupt/blockchain.db")
	db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	db.Close()

	os.WriteFile("/tmp/test_corrupt/metadata.json", []byte("not json"), 0644)

	svc := NewBackupService(nil)
	err := svc.Verify(context.Background(), "/tmp/test_corrupt")
	if err == nil {
		t.Error("Expected error for corrupt metadata")
	}
}

func TestBackupService_Restore(t *testing.T) {
	os.MkdirAll("/tmp/test_restore_backup", 0755)
	defer os.RemoveAll("/tmp/test_restore_backup")

	db, _ := sql.Open("sqlite3", "/tmp/test_restore_backup/blockchain.db")
	db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	db.Close()

	metadata := `{"version":"1.2","timestamp":"2026-04-30T00:00:00Z","checksum":"","databases":["blockchain"],"schema_version":1}`
	os.WriteFile("/tmp/test_restore_backup/metadata.json", []byte(metadata), 0644)

	os.MkdirAll("/tmp/test_restore_dest", 0755)
	defer os.RemoveAll("/tmp/test_restore_dest")

	svc := NewBackupService(map[string]string{
		"blockchain": "/tmp/test_restore_dest/blockchain.db",
	})

	err := svc.Restore(context.Background(), "/tmp/test_restore_backup")
	if err != nil {
		t.Errorf("Restore failed: %v", err)
	}

	if _, err := os.Stat("/tmp/test_restore_dest/blockchain.db"); os.IsNotExist(err) {
		t.Error("Expected restored database to exist")
	}
}
