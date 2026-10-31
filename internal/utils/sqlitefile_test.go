package utils

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteFilePath(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "nested state", "aurora?copy#one.db")
	diskURI := sqliteFileURI(t, dbPath)

	parsed, err := url.Parse(diskURI)
	if err != nil {
		t.Fatal(err)
	}
	localhost := *parsed
	localhost.Host = "localhost"
	remote := *parsed
	remote.Host = "fileserver"
	uppercaseLocalhost := *parsed
	uppercaseLocalhost.Host = "LOCALHOST"

	tests := []struct {
		name      string
		dsn       string
		wantPath  string
		ephemeral bool
		wantErr   bool
	}{
		{name: "plain path is preserved", dsn: dbPath, wantPath: dbPath},
		{name: "plain question mark is part of the path", dsn: "aurora?copy.db?mode=memory", wantPath: "aurora?copy.db?mode=memory"},
		{name: "empty name is temporary", dsn: "", ephemeral: true},
		{name: "memory sentinel", dsn: ":memory:", ephemeral: true},
		{name: "file memory sentinel", dsn: "file::memory:", ephemeral: true},
		{name: "named memory database", dsn: "file:shared?mode=memory&cache=shared", ephemeral: true},
		{name: "empty name with memory mode", dsn: "file:?mode=memory", ephemeral: true},
		{name: "empty file URI", dsn: "file:", ephemeral: true},
		{name: "relative file URI", dsn: "file:relative.db", wantPath: filepath.FromSlash("relative.db")},
		{name: "dot relative file URI", dsn: "file:./relative.db", wantPath: filepath.FromSlash("./relative.db")},
		{name: "absolute escaped file URI", dsn: diskURI, wantPath: dbPath},
		{name: "localhost authority", dsn: localhost.String(), wantPath: dbPath},
		{name: "plus is not a query space", dsn: "file:/tmp/state+copy.db", wantPath: filepath.FromSlash("/tmp/state+copy.db")},
		{name: "mode memory", dsn: diskURI + "?mode=memory", ephemeral: true},
		{name: "last mode wins when memory is last", dsn: diskURI + "?mode=rwc&mode=memory", ephemeral: true},
		{name: "last mode wins when disk is last", dsn: diskURI + "?mode=memory&mode=rwc", wantPath: dbPath},
		{name: "mode value is percent decoded", dsn: diskURI + "?mode=mem%6Fry", ephemeral: true},
		{name: "fragment ends the database name", dsn: diskURI + "?mode=ro#ignored?mode=memory", wantPath: dbPath},
		{name: "remote authority is rejected", dsn: remote.String(), wantErr: true},
		{name: "localhost is case sensitive like SQLite", dsn: uppercaseLocalhost.String(), wantErr: true},
		{name: "NUL escape is rejected", dsn: "file:/tmp/aurora%00.db", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ephemeral, err := SQLiteFilePath(tt.dsn)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SQLiteFilePath(%q) = (%q, %v, nil), want error", tt.dsn, path, ephemeral)
				}
				return
			}
			if err != nil {
				t.Fatalf("SQLiteFilePath(%q): %v", tt.dsn, err)
			}
			if path != tt.wantPath || ephemeral != tt.ephemeral {
				t.Fatalf("SQLiteFilePath(%q) = (%q, %v), want (%q, %v)",
					tt.dsn, path, ephemeral, tt.wantPath, tt.ephemeral)
			}
		})
	}
}

func TestEnsureSQLiteDirectoryUsesDecodedURIPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested state", "aurora.db")
	launchDir := t.TempDir()
	t.Chdir(launchDir)

	if err := EnsureSQLiteDirectory(sqliteFileURI(t, dbPath)); err != nil {
		t.Fatalf("EnsureSQLiteDirectory: %v", err)
	}

	info, err := os.Stat(filepath.Dir(dbPath))
	if err != nil {
		t.Fatalf("stat database directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", filepath.Dir(dbPath))
	}
	if entries, err := os.ReadDir(launchDir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Fatalf("EnsureSQLiteDirectory created %d path(s) under the launch directory, want none", len(entries))
	}
}

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

func TestRestrictSQLiteFilesSecuresSidecarsOfSymlinkedDatabase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	linkDir := filepath.Join(root, "link")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(realDir, "aurora.db")
	linkPath := filepath.Join(linkDir, "aurora.db")
	paths := []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("probe"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(dbPath, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := RestrictSQLiteFiles(linkPath); err != nil {
		t.Fatalf("RestrictSQLiteFiles: %v", err)
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %04o, want 0600", path, got)
		}
	}
}

func TestRestrictSQLiteFilesSecuresDiskFileURI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	dbPath := filepath.Join(t.TempDir(), "aurora state.db")
	paths := []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("probe"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dsn := sqliteFileURI(t, dbPath)

	if err := RestrictSQLiteFiles(dsn); err != nil {
		t.Fatalf("RestrictSQLiteFiles(%q): %v", dsn, err)
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %04o, want 0600", path, got)
		}
	}
}

func TestRestrictSQLiteFilesSkipsMemoryDSNs(t *testing.T) {
	for _, dsn := range []string{
		":memory:",
		"file::memory:",
		"file:shared?mode=memory&cache=shared",
		"file:?mode=memory",
	} {
		if err := RestrictSQLiteFiles(dsn); err != nil {
			t.Errorf("RestrictSQLiteFiles(%q): %v", dsn, err)
		}
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

func sqliteFileURI(t *testing.T, path string) string {
	t.Helper()
	uriPath := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	return (&url.URL{Scheme: "file", Path: uriPath}).String()
}

// TestSQLiteDiskUsageCountsSidecars pins the definition both `aurora lottery
// db-info` and the desktop's Settings readout use: the database plus the files
// SQLite keeps beside it. A missing file contributes nothing.
func TestSQLiteDiskUsageCountsSidecars(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	require.NoError(t, os.WriteFile(dbPath, make([]byte, 120), 0o600))
	require.NoError(t, os.WriteFile(dbPath+"-wal", make([]byte, 30), 0o600))

	assert.Equal(t, int64(150), SQLiteDiskUsage(dbPath))
	assert.Equal(t, int64(0), SQLiteDiskUsage(filepath.Join(dir, "never-written.db")))

	require.NoError(t, os.Remove(dbPath+"-wal"))
	assert.Equal(t, int64(120), SQLiteDiskUsage(dbPath))
}

// TestEnsureSQLiteDirectoryRejectsADirectoryPath: a configured db.path that
// names a directory used to surface as SQLite's "unable to open database file
// (14)" on the first write, which names neither the path nor the reason.
func TestEnsureSQLiteDirectoryRejectsADirectoryPath(t *testing.T) {
	dir := t.TempDir()
	err := EnsureSQLiteDirectory(dir)
	if err == nil {
		t.Fatal("EnsureSQLiteDirectory accepted a directory as the database path")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("error = %v, want it to say the path is a directory", err)
	}
}
