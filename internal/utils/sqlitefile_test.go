package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRestrictSQLiteFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		if err := os.WriteFile(path, []byte("probe"), 0o644); err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("chmod %s: %v", path, err)
		}
	}

	if err := RestrictSQLiteFiles(dbPath); err != nil {
		t.Fatalf("RestrictSQLiteFiles: %v", err)
	}
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %04o, want 0600", path, got)
		}
	}
}

func TestRestrictSQLiteFilesAllowsMissingSidecars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	if err := os.WriteFile(dbPath, []byte("probe"), 0o644); err != nil {
		t.Fatalf("create database: %v", err)
	}
	if err := RestrictSQLiteFiles(dbPath); err != nil {
		t.Fatalf("RestrictSQLiteFiles: %v", err)
	}
}

func TestRestrictSQLiteFilesRequiresMainDatabase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	if err := RestrictSQLiteFiles(filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("RestrictSQLiteFiles(missing main DB) = nil, want error")
	}
}
