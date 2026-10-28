package sqlite

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRepositoryRestrictsExistingSQLiteStateFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	tests := []struct {
		name string
		open func(string) (io.Closer, error)
	}{
		{name: "lottery", open: func(path string) (io.Closer, error) { return NewLotteryRepository(path) }},
		{name: "nft", open: func(path string) (io.Closer, error) { return NewNFTRepository(path) }},
		{name: "token", open: func(path string) (io.Closer, error) { return NewTokenRepository(path) }},
		{name: "oracle", open: func(path string) (io.Closer, error) { return NewOracleRepository(path) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "state", "aurora.db")
			repo, err := tt.open(dbPath)
			if err != nil {
				t.Fatalf("create repository: %v", err)
			}
			if err := repo.Close(); err != nil {
				t.Fatalf("close initial repository: %v", err)
			}

			// Simulate files created by an older release under the common 022
			// umask. Reopening must tighten every file that exists.
			for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
				if err := os.Chmod(path, 0o644); err != nil && !os.IsNotExist(err) {
					t.Fatalf("make %s world-readable: %v", path, err)
				}
			}

			repo, err = tt.open(dbPath)
			if err != nil {
				t.Fatalf("reopen repository: %v", err)
			}
			t.Cleanup(func() { _ = repo.Close() })

			for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
				info, err := os.Stat(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					t.Fatalf("stat %s: %v", path, err)
				}
				if got := info.Mode().Perm(); got != 0o600 {
					t.Errorf("%s mode = %04o, want 0600", path, got)
				}
			}
		})
	}
}

func TestRepositorySecuresDiskFileURIStateFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	tests := []struct {
		name string
		open func(string) (io.Closer, error)
	}{
		{name: "lottery", open: func(path string) (io.Closer, error) { return NewLotteryRepository(path) }},
		{name: "nft", open: func(path string) (io.Closer, error) { return NewNFTRepository(path) }},
		{name: "token", open: func(path string) (io.Closer, error) { return NewTokenRepository(path) }},
		{name: "oracle", open: func(path string) (io.Closer, error) { return NewOracleRepository(path) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "state with spaces", "aurora.db")
			dsn := (&url.URL{Scheme: "file", Path: dbPath}).String()
			launchDir := t.TempDir()
			t.Chdir(launchDir)

			repo, err := tt.open(dsn)
			if err != nil {
				t.Fatalf("create repository through file URI: %v", err)
			}
			if err := repo.Close(); err != nil {
				t.Fatalf("close initial repository: %v", err)
			}
			if _, err := os.Stat(dbPath); err != nil {
				t.Fatalf("database was not created at the URI path %s: %v", dbPath, err)
			}

			// Simulate state left by an older release under the common 022
			// umask, including whichever sidecars that release created.
			for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
				if err := os.Chmod(path, 0o644); err != nil && !os.IsNotExist(err) {
					t.Fatalf("make %s world-readable: %v", path, err)
				}
			}

			repo, err = tt.open(dsn)
			if err != nil {
				t.Fatalf("reopen repository through file URI: %v", err)
			}
			t.Cleanup(func() { _ = repo.Close() })

			for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
				info, err := os.Stat(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					t.Fatalf("stat %s: %v", path, err)
				}
				if got := info.Mode().Perm(); got != 0o600 {
					t.Errorf("%s mode = %04o, want 0600", path, got)
				}
			}
			if _, err := os.Stat(filepath.Join(launchDir, "file:")); !os.IsNotExist(err) {
				t.Fatalf("file URI created a path under the launch directory: %v", err)
			}
		})
	}
}
