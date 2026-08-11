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

// makeTestDB writes a real SQLite file at path with one table so backup and
// verify have a genuine database to operate on.
func makeTestDB(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

// writeMetadata writes a minimal, checksum-less metadata.json into a backup dir
// (Restore does not re-verify the checksum; it only needs the file to parse).
func writeMetadata(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	meta := `{"version":"1.2","checksum":"","databases":["main"],"schema_version":1}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(meta), 0640))
}

// TestBackupService_Create_CheckpointError covers the PRAGMA wal_checkpoint
// failure branch: a configured "DB" that is actually a directory opens lazily
// but fails on the first executed statement.
func TestBackupService_Create_CheckpointError(t *testing.T) {
	dir := t.TempDir()
	dbAsDir := filepath.Join(dir, "src", "blockchain.db")
	require.NoError(t, os.MkdirAll(dbAsDir, 0755))

	svc := NewBackupService(map[string]string{"main": dbAsDir})
	_, err := svc.Create(context.Background(), filepath.Join(dir, "out"))
	require.Error(t, err, "a directory masquerading as a DB must fail at checkpoint")
}

// TestBackupService_Create_MetadataWriteError covers the metadata.json write
// failure branch: a pre-existing directory at the destination path makes
// os.WriteFile fail after the DB copies succeeded.
func TestBackupService_Create_MetadataWriteError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(filepath.Join(out, "metadata.json"), 0755))

	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.Error(t, err, "metadata.json sunken by a directory must surface as an error")
}

// TestBackupService_Verify_MissingDatabaseFile covers the branch where the
// checksum is intact but a listed database file is absent (e.g. a truncated
// or partially-recovered backup).
func TestBackupService_Verify_MissingDatabaseFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "src", "a.db")
	b := filepath.Join(dir, "src", "b.db")
	makeTestDB(t, a)
	makeTestDB(t, b)

	svc := NewBackupService(map[string]string{"a": a, "b": b})
	out := filepath.Join(dir, "out")
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	require.NoError(t, os.Remove(filepath.Join(out, "b.db")))
	err = svc.Verify(context.Background(), out)
	require.Error(t, err, "verify must report the missing database file")
}

// TestBackupService_Restore_PreRestoreDirCollision covers the failure branch
// where the pre-restore staging directory's path is already occupied by a
// regular file.
func TestBackupService_Restore_PreRestoreDirCollision(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	writeMetadata(t, backupDir)
	require.NoError(t, os.WriteFile(backupDir+".pre_restore", []byte("occupied"), 0644))

	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(dest, 0755))
	svc := NewBackupService(map[string]string{"main": filepath.Join(dest, "main.db")})

	err := svc.Restore(context.Background(), backupDir)
	require.Error(t, err, "a file squatting on the pre_restore path must fail restore")
}

// TestBackupService_Restore_PreRestoreCopyError covers the branch where the
// current database file is a directory: the pre-restore io.Copy fails while
// reading it (EISDIR), before any backup file is touched.
func TestBackupService_Restore_PreRestoreCopyError(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	writeMetadata(t, backupDir)

	current := filepath.Join(dir, "dest", "main.db")
	require.NoError(t, os.MkdirAll(current, 0755))

	svc := NewBackupService(map[string]string{"main": current})
	err := svc.Restore(context.Background(), backupDir)
	require.Error(t, err, "pre-restore copy of a directory must fail")
}

// TestBackupService_Restore_MainCopyError covers the restore io.Copy failure
// branch (backup file is a directory) AND the pre-restore happy path (a real
// current file gets staged before the restore copy fails).
func TestBackupService_Restore_MainCopyError(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	writeMetadata(t, backupDir)
	require.NoError(t, os.MkdirAll(filepath.Join(backupDir, "main.db"), 0755))

	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(dest, 0755))
	current := filepath.Join(dest, "main.db")
	require.NoError(t, os.WriteFile(current, []byte("existing state"), 0644))

	svc := NewBackupService(map[string]string{"main": current})
	err := svc.Restore(context.Background(), backupDir)
	require.Error(t, err, "restore copy of a directory backup must fail")
}

// TestBackupService_Create_DestOpenFileError covers the destination-open
// failure branch: an output dir where the destination <name>.db path is
// already a directory makes os.OpenFile fail after the source was opened.
func TestBackupService_Create_DestOpenFileError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(filepath.Join(out, "blockchain.db"), 0755))

	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.Error(t, err, "destination DB path squatting as a directory must fail")
}
