package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	makeValidBackup(t, backupDir, "blockchain")

	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(dest, 0755))
	svc := NewBackupService(map[string]string{
		"blockchain": filepath.Join(dest, "blockchain.db"),
	})

	err := svc.Restore(context.Background(), backupDir)
	require.NoError(t, err, "restore of a valid backup should succeed")
	require.FileExists(t, filepath.Join(dest, "blockchain.db"))
}

// TestBackupService_Restore_RefusesCorruptBackup proves the v1.20 safety
// guard: Restore validates the backup's integrity first and refuses to clobber
// the live database with a corrupted backup.
func TestBackupService_Restore_RefusesCorruptBackup(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	makeValidBackup(t, backupDir, "main")
	// Corrupt the DB copy so integrity verification fails.
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "main.db"), []byte("garbage"), 0644))

	dest := filepath.Join(dir, "dest")
	current := filepath.Join(dest, "main.db")
	makeTestDB(t, current)

	svc := NewBackupService(map[string]string{"main": current})
	err := svc.Restore(context.Background(), backupDir)
	require.ErrorContains(t, err, "refusing to restore")
	// The live DB must be untouched.
	require.FileExists(t, current)
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

// TestBackupService_Verify_DetectsDBContentCorruption proves the v1.55
// per-database content checksum: a corrupted .db whose data changed (but which
// still opens and has a table) must be rejected by Verify. Before v1.55 Verify
// hashed only the metadata JSON and checked each DB merely for "opens with
// >=1 table", so a truncated or bit-rotten database passed as valid and could
// be restored over a good one (TASK-068, ISS-060).
func TestBackupService_Verify_DetectsDBContentCorruption(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	srcDir := filepath.Join(dir, "src")
	srcDB := filepath.Join(srcDir, "blockchain.db")
	makeTestDB(t, srcDB)

	// Create the backup via Create() so the metadata records the per-database
	// content checksum (the field makeValidBackup/writeMeta do not set).
	svc := NewBackupService(map[string]string{"blockchain": srcDB})
	_, err := svc.Create(context.Background(), backupDir)
	require.NoError(t, err)

	// A valid backup verifies cleanly.
	require.NoError(t, svc.Verify(context.Background(), backupDir))

	// Corrupt the DB file content in place, keeping it byte-valid enough to
	// still open with a table — the exact failure mode the metadata-only
	// checksum previously missed.
	backupDB := filepath.Join(backupDir, "blockchain.db")
	b, err := os.ReadFile(backupDB)
	require.NoError(t, err)
	require.Greater(t, len(b), 0)
	b[len(b)-1] ^= 0xFF // flip the last byte
	require.NoError(t, os.WriteFile(backupDB, b, 0640))

	err = svc.Verify(context.Background(), backupDir)
	require.Error(t, err, "Verify must reject a database whose content changed")
	require.Contains(t, err.Error(), "checksum mismatch")
}

// makeValidBackup writes a real SQLite DB and a checksum-consistent
// metadata.json into dir so the backup passes Verify. Restore now verifies the
// backup before touching live databases (v1.20), so restore tests must use
// genuinely valid backups to reach the restore code paths.
func makeValidBackup(t *testing.T, dir, name string) {
	t.Helper()
	dbPath := filepath.Join(dir, name+".db")
	makeTestDB(t, dbPath)

	meta := BackupMetadata{
		Version:       "1.2",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Databases:     []string{name},
		SchemaVersion: 1,
	}
	checksumData, err := json.Marshal(meta)
	require.NoError(t, err)
	meta.Checksum = fmt.Sprintf("%x", sha256.Sum256(checksumData))
	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0640))
}

// writeMetadata writes a valid backup for DB name "main" (see makeValidBackup).
func writeMetadata(t *testing.T, dir string) {
	t.Helper()
	makeValidBackup(t, dir, "main")
}

// TestBackupService_Create_CheckpointError covers the snapshot failure branch:
// a configured "DB" that is actually a directory opens lazily but fails on the
// first executed statement (previously the PRAGMA wal_checkpoint, now the
// VACUUM INTO snapshot, v1.75).
func TestBackupService_Create_CheckpointError(t *testing.T) {
	dir := t.TempDir()
	dbAsDir := filepath.Join(dir, "src", "blockchain.db")
	require.NoError(t, os.MkdirAll(dbAsDir, 0755))

	svc := NewBackupService(map[string]string{"main": dbAsDir})
	_, err := svc.Create(context.Background(), filepath.Join(dir, "out"))
	require.Error(t, err, "a directory masquerading as a DB must fail at checkpoint")
}

// TestBackupService_Create_MetadataWrittenAtomically pins the atomic-metadata
// tail (TASK-120). The old code os.WriteFile'd metadata.json directly after
// promoting the .db files: a crash or short write mid-way left a truncated or
// absent metadata.json on top of brand-new .db files — a backup set that fails
// Verify with the previous good metadata already gone. Create now writes a
// metadata.json.tmp sibling and renames it into place, so the visible
// metadata.json is always a complete, verify-able file and no .tmp leaks after
// success. This test asserts exactly those two invariants.
func TestBackupService_Create_MetadataWrittenAtomically(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	res, err := svc.Create(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, res.Checksum)

	metaPath := filepath.Join(out, "metadata.json")
	metaData, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	// The metadata must be complete and self-consistent: its Checksum field
	// must equal the SHA of the OTHER fields as marshaled (the same
	// compute-then-fill order Create uses), proving the write is not truncated.
	var meta BackupMetadata
	require.NoError(t, json.Unmarshal(metaData, &meta), "metadata.json is complete, valid JSON")
	storedChecksum := meta.Checksum
	meta.Checksum = "" // the stored checksum covers the pre-checksum fields only
	raw, _ := json.Marshal(meta)
	require.Equal(t, sha256hex(raw), storedChecksum, "metadata checksum field verifies")

	require.NoFileExists(t, metaPath+".tmp", "no metadata temp sibling leaks after success")
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

// TestBackupService_Restore_RejectsDirDB covers the verify guard on the
// "backup DB path is a directory" vector: instead of copying a non-file as the
// live DB, Restore refuses before touching anything (v1.20).
func TestBackupService_Restore_RejectsDirDB(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	writeMetadata(t, backupDir)
	// Replace the valid backup DB with a directory so Verify fails.
	require.NoError(t, os.Remove(filepath.Join(backupDir, "main.db")))
	require.NoError(t, os.MkdirAll(filepath.Join(backupDir, "main.db"), 0755))

	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.MkdirAll(dest, 0755))
	current := filepath.Join(dest, "main.db")
	require.NoError(t, os.WriteFile(current, []byte("existing state"), 0644))

	svc := NewBackupService(map[string]string{"main": current})
	err := svc.Restore(context.Background(), backupDir)
	require.ErrorContains(t, err, "refusing to restore")
	require.FileExists(t, current)
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

// TestBackupService_Create_RefusesSelfOverwrite locks ISS-071 / TASK-079:
// backing up into a directory that already contains the live source DB (e.g.
// `aurora backup create ./data` when the DB lives at ./data/aurora.db) must
// fail with an explicit error instead of O_TRUNC-ing the source to 0 bytes and
// reporting success — the live database has to survive untouched.
func TestBackupService_Create_RefusesSelfOverwrite(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	dbPath := filepath.Join(dataDir, "aurora.db")

	makeTestDB(t, dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO test (id) VALUES (42)")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	svc := NewBackupService(map[string]string{"aurora": dbPath})
	_, err = svc.Create(context.Background(), dataDir)
	require.Error(t, err, "backing up into the directory that holds the live DB must be refused")
	require.ErrorContains(t, err, "into itself")

	// The live database must still hold the marker row — no truncation.
	reopened, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer reopened.Close()
	var count int
	require.NoError(t, reopened.QueryRow("SELECT COUNT(*) FROM test WHERE id = 42").Scan(&count))
	require.Equal(t, 1, count)
}

// TestBackupService_Create_AbortLeavesPreviousSetIntact is the regression
// test for the destructive-on-failure backup bug (TASK-109, ISS-101): Create
// removed each existing <name>.db before VACUUM INTO, so a mid-run failure
// deleted the previous good snapshot of a DB while metadata.json still
// described it — the whole backup set became unverifiable and the operator's
// only good copy was gone. Create is now two-phase (snapshot-all → promote-all),
// so a failed run must leave the previous files, metadata, and verifiability
// exactly as they were, with no leftover .tmp files.
func TestBackupService_Create_AbortLeavesPreviousSetIntact(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "backups")

	mkDB := func(rel string, table string) string {
		p := filepath.Join(dir, "src", rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		db, err := sql.Open("sqlite3", p)
		require.NoError(t, err)
		_, err = db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY)")
		require.NoError(t, err)
		require.NoError(t, db.Close())
		return p
	}
	mainPath := mkDB("main.db", "things")
	extraPath := mkDB("extra.db", "more_things")

	svc := NewBackupService(map[string]string{"main": mainPath, "extra": extraPath})
	_, err := svc.Create(context.Background(), output)
	require.NoError(t, err, "first (good) backup must succeed")

	// Snapshot the good state and confirm it verifies.
	goodMain, err := os.ReadFile(filepath.Join(output, "main.db"))
	require.NoError(t, err)
	goodMeta, err := os.ReadFile(filepath.Join(output, "metadata.json"))
	require.NoError(t, err)
	require.NoError(t, svc.Verify(context.Background(), output))
	require.NoError(t, err, "good backup must verify before the aborted run")

	// Make the second DB fail, then re-run Create. The removed source must
	// ALSO stay removed: the schema probe used to open it (SQLite creates a
	// missing DB on first use), silently materializing an empty extra.db and
	// letting Create "succeed" with a fake empty backup.
	require.NoError(t, os.Remove(extraPath))
	_, err = svc.Create(context.Background(), output)
	require.Error(t, err, "Create must fail when a configured DB is gone")
	require.NoFileExists(t, extraPath, "a Create run must not materialize a missing DB source")

	// The previous set must be byte-identical and still verifiable.
	afterMain, err := os.ReadFile(filepath.Join(output, "main.db"))
	require.NoError(t, err)
	require.Equal(t, goodMain, afterMain, "previous main.db must survive an aborted Create")
	afterMeta, err := os.ReadFile(filepath.Join(output, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, goodMeta, afterMeta, "previous metadata.json must survive an aborted Create")

	// No half-written temp snapshots may remain.
	dirEntries, err := os.ReadDir(output)
	require.NoError(t, err)
	for _, e := range dirEntries {
		require.NotContains(t, e.Name(), ".tmp", "aborted Create must not leave temp snapshot files")
	}

	require.NoError(t, svc.Verify(context.Background(), output))
	require.NoError(t, err, "aborted Create must not corrupt the previous backup set")
}
