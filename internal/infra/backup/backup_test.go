package backup

import (
	"context"
	"os"
	"testing"
)

func TestBackupService_Create(t *testing.T) {
	svc := NewBackupService(map[string]string{
		"blockchain": "data/blockchain.db",
		"tokens":     "data/tokens.db",
	})

	result, err := svc.Create(context.Background(), "/tmp/test_backup.json")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if result.File != "/tmp/test_backup.json" {
		t.Errorf("Expected file /tmp/test_backup.json, got %s", result.File)
	}

	if result.Size == 0 {
		t.Error("Expected non-zero size")
	}

	if result.Checksum == "" {
		t.Error("Expected non-empty checksum")
	}

	os.Remove("/tmp/test_backup.json")
}

func TestBackupService_Verify(t *testing.T) {
	svc := NewBackupService(map[string]string{
		"blockchain": "data/blockchain.db",
		"tokens":     "data/tokens.db",
	})

	_, err := svc.Create(context.Background(), "/tmp/test_verify.json")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer os.Remove("/tmp/test_verify.json")

	if err := svc.Verify(context.Background(), "/tmp/test_verify.json"); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestBackupService_VerifyInvalidFile(t *testing.T) {
	svc := NewBackupService(nil)

	err := svc.Verify(context.Background(), "/nonexistent/file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestBackupService_VerifyInvalidJSON(t *testing.T) {
	svc := NewBackupService(nil)

	os.WriteFile("/tmp/invalid.json", []byte("not json"), 0644)
	defer os.Remove("/tmp/invalid.json")

	err := svc.Verify(context.Background(), "/tmp/invalid.json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestBackupService_Restore(t *testing.T) {
	svc := NewBackupService(nil)

	err := svc.Restore(context.Background(), "/tmp/test_backup.json")
	if err == nil {
		t.Error("Expected error for unimplemented Restore")
	}
}