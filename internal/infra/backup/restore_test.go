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

	_ "github.com/mattn/go-sqlite3"
)

// newDB creates a SQLite file at path with the given DDL statements applied.
// Returns once the file is closed; caller owns the path.
func newDB(t *testing.T, path string, ddl ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

// sha256hex returns the lowercase hex SHA256 of data.
func sha256hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// --- getSchemaVersion / querySchemaVersion ---------------------------------

// TestGetSchemaVersion_ReturnsHighestAcrossDBs locks in the bug fix:
//
// Originally getSchemaVersion() returned the first non-zero version found
// in map iteration order. For a service that backs up several DBs with
// different migration histories (which is the realistic case for a
// multi-store application like Aurora — blockchain, nft, token, oracle,
// voting tables all live in different DB files), the backup metadata
// recorded whichever schema version Go's map iteration hit first, not
// the actual highest. The fix scans every DB and returns the max.
func TestGetSchemaVersion_ReturnsHighestAcrossDBs(t *testing.T) {
	tmp := t.TempDir()
	low := filepath.Join(tmp, "low.db")
	mid := filepath.Join(tmp, "mid.db")
	none := filepath.Join(tmp, "none.db")
	// Missing path — file intentionally never created on disk.
	missing := filepath.Join(tmp, "missing.db")

	newDB(t, low,
		"CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)",
		"INSERT INTO schema_migrations (version) VALUES (3)",
	)
	newDB(t, mid,
		"CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)",
		"INSERT INTO schema_migrations (version) VALUES (5)",
	)
	newDB(t, none,
		"CREATE TABLE users (id INTEGER PRIMARY KEY)",
	)

	svc := NewBackupService(map[string]string{
		"low":     low,
		"mid":     mid,
		"none":    none,
		"missing": missing,
	})

	got := svc.getSchemaVersion()
	if got != 5 {
		t.Fatalf("getSchemaVersion() = %d, want 5 (max across all DBs)", got)
	}
}

// TestGetSchemaVersion_NoDBsHaveMigrations confirms graceful degradation
// when none of the configured DBs have a schema_migrations table.
func TestGetSchemaVersion_NoDBsHaveMigrations(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "nohistory.db")
	newDB(t, db, "CREATE TABLE users (id INTEGER PRIMARY KEY)")

	svc := NewBackupService(map[string]string{"x": db})
	if got := svc.getSchemaVersion(); got != 0 {
		t.Errorf("expected 0 when no DB has migrations, got %d", got)
	}
}

// TestQuerySchemaVersion_NegativeVersion guards against a hypothetical
// schema_migrations table containing a negative value (operator typo).
// We clamp to 0 instead of returning a huge uint.
func TestQuerySchemaVersion_NegativeVersion(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "neg.db")
	newDB(t, db,
		"CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)",
		"INSERT INTO schema_migrations (version) VALUES (-1)",
	)

	if got := querySchemaVersion(db); got != 0 {
		t.Errorf("negative version should clamp to 0, got %d", got)
	}
}

// TestQuerySchemaVersion_MissingFile confirms querySchemaVersion returns
// 0 (not panics) when the path doesn't exist.
func TestQuerySchemaVersion_MissingFile(t *testing.T) {
	if got := querySchemaVersion("/tmp/does-not-exist-aurora-backup.db"); got != 0 {
		t.Errorf("expected 0 for missing DB, got %d", got)
	}
}

// --- Restore ---------------------------------------------------------------

// TestRestore_RoundTrip proves Restore actually overwrites the destination
// DB with the bytes from the backup directory AND that the resulting DB
// is readable and contains the original data.
//
// Previously the only Restore test verified the destination file existed
// — it never read the restored DB or seeded it with distinguishable data,
// so a Restore that silently zeroed out the destination would have passed.
func TestRestore_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	dstDir := filepath.Join(tmp, "dst")
	backupDir := filepath.Join(tmp, "backup")
	for _, d := range []string{dstDir, backupDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	dstDB := filepath.Join(dstDir, "blockchain.db")
	backupDB := filepath.Join(backupDir, "blockchain.db")

	// 1. Write the backup DB with known data.
	newDB(t, backupDB,
		"CREATE TABLE kv (k TEXT PRIMARY KEY, v TEXT NOT NULL)",
		"INSERT INTO kv (k, v) VALUES ('answer', '42')",
	)

	// 2. Write the destination DB with DIFFERENT data — Restore must
	//    overwrite it.
	newDB(t, dstDB,
		"CREATE TABLE kv (k TEXT PRIMARY KEY, v TEXT NOT NULL)",
		"INSERT INTO kv (k, v) VALUES ('answer', 'WRONG')",
	)

	// 3. Write a valid metadata.json referencing the backup DB.
	writeMeta(t, backupDir, "blockchain")

	svc := NewBackupService(map[string]string{"blockchain": dstDB})
	if err := svc.Restore(context.Background(), backupDir); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Verify the destination now has the BACKUP's data, not the old data.
	if got := readKV(t, dstDB, "answer"); got != "42" {
		t.Errorf("after Restore, dstDB.kv[answer] = %q, want %q", got, "42")
	}

	// Verify the pre-restore backup of the original DB exists and holds
	// the OLD data.
	prePath := filepath.Join(backupDir+".pre_restore", "blockchain.db")
	if _, err := os.Stat(prePath); err != nil {
		t.Fatalf("pre_restore backup missing: %v", err)
	}
	if preData := readKV(t, prePath, "answer"); preData != "WRONG" {
		t.Errorf("pre_restore should contain the OLD data, got %q", preData)
	}
}

// TestRestore_NoExistingDest confirms Restore works when the destination
// DB does not yet exist (first-time restore). The pre-restore backup step
// must be skipped, not error.
func TestRestore_NoExistingDest(t *testing.T) {
	tmp := t.TempDir()
	dstDir := filepath.Join(tmp, "dst")
	backupDir := filepath.Join(tmp, "backup")
	for _, d := range []string{dstDir, backupDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	dstDB := filepath.Join(dstDir, "blockchain.db")
	backupDB := filepath.Join(backupDir, "blockchain.db")
	newDB(t, backupDB,
		"CREATE TABLE kv (k TEXT PRIMARY KEY, v TEXT NOT NULL)",
		"INSERT INTO kv (k, v) VALUES ('k', 'v')",
	)
	writeMeta(t, backupDir, "blockchain")

	svc := NewBackupService(map[string]string{"blockchain": dstDB})
	if err := svc.Restore(context.Background(), backupDir); err != nil {
		t.Fatalf("Restore with no existing dest: %v", err)
	}

	if _, err := os.Stat(dstDB); err != nil {
		t.Fatalf("expected destination to exist after Restore: %v", err)
	}
	if got := readKV(t, dstDB, "k"); got != "v" {
		t.Errorf("restored value = %q, want %q", got, "v")
	}
}

// TestRestore_MissingBackupDB covers the error path when the backup
// directory is missing the DB file listed in metadata.
func TestRestore_MissingBackupDB(t *testing.T) {
	tmp := t.TempDir()
	dstDir := filepath.Join(tmp, "dst")
	backupDir := filepath.Join(tmp, "backup")
	for _, d := range []string{dstDir, backupDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// metadata.json claims "blockchain" is in the backup, but the file
	// is absent. Restore must error rather than silently leaving the
	// destination unchanged.
	writeMeta(t, backupDir, "blockchain")

	svc := NewBackupService(map[string]string{
		"blockchain": filepath.Join(dstDir, "blockchain.db"),
	})
	err := svc.Restore(context.Background(), backupDir)
	if err == nil {
		t.Fatal("expected error when backup DB is missing, got nil")
	}
}

// TestRestore_MissingMetadata covers the error path when metadata.json
// is absent.
func TestRestore_MissingMetadata(t *testing.T) {
	tmp := t.TempDir()
	backupDir := filepath.Join(tmp, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	svc := NewBackupService(map[string]string{"x": "/tmp/nope.db"})
	err := svc.Restore(context.Background(), backupDir)
	if err == nil {
		t.Fatal("expected error when metadata.json is missing, got nil")
	}
}

// TestRestore_CorruptedMetadata covers the error path when metadata.json
// is present but contains invalid JSON.
func TestRestore_CorruptedMetadata(t *testing.T) {
	tmp := t.TempDir()
	backupDir := filepath.Join(tmp, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "metadata.json"), []byte("not json {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewBackupService(map[string]string{"x": "/tmp/nope.db"})
	err := svc.Restore(context.Background(), backupDir)
	if err == nil {
		t.Fatal("expected error when metadata.json is corrupted, got nil")
	}
}

// --- Verify ----------------------------------------------------------------

// TestVerify_MissingDBFile covers the error path in Verify when a database
// file listed in metadata.json is missing.
func TestVerify_MissingDBFile(t *testing.T) {
	tmp := t.TempDir()
	backupDir := filepath.Join(tmp, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// metadata references "ghost.db" but the file doesn't exist.
	writeMeta(t, backupDir, "ghost")

	svc := NewBackupService(map[string]string{"ghost": "/tmp/ghost.db"})
	err := svc.Verify(context.Background(), backupDir)
	if err == nil {
		t.Fatal("expected error for missing DB file, got nil")
	}
}

// TestVerify_ChecksumMismatch covers the error path when the metadata's
// recorded checksum does not match the recomputed one.
func TestVerify_ChecksumMismatch(t *testing.T) {
	tmp := t.TempDir()
	backupDir := filepath.Join(tmp, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// metadata.json with a deliberately wrong checksum.
	bad := BackupMetadata{
		Version:       "1.2",
		Timestamp:     "2026-04-30T00:00:00Z",
		Checksum:      "deadbeef", // wrong
		Databases:     []string{"x"},
		SchemaVersion: 1,
	}
	raw, _ := json.MarshalIndent(bad, "", "  ")
	if err := os.WriteFile(filepath.Join(backupDir, "metadata.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// Provide a valid DB file so Verify gets past the missing-file check
	// and reaches the checksum check.
	newDB(t, filepath.Join(backupDir, "x.db"),
		"CREATE TABLE t (id INTEGER PRIMARY KEY)",
	)

	svc := NewBackupService(map[string]string{"x": "/tmp/x.db"})
	err := svc.Verify(context.Background(), backupDir)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

// --- helpers --------------------------------------------------------------

// readKV opens the SQLite DB at path and returns the value of kv[k].
func readKV(t *testing.T, path, k string) string {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	var v string
	if err := db.QueryRow("SELECT v FROM kv WHERE k = ?", k).Scan(&v); err != nil {
		t.Fatalf("read kv[%s]: %v", k, err)
	}
	return v
}

// writeMeta writes a valid metadata.json referencing the named DB.
func writeMeta(t *testing.T, backupDir, dbName string) {
	t.Helper()
	meta := BackupMetadata{
		Version:       "1.2",
		Timestamp:     "2026-04-30T00:00:00Z",
		Checksum:      "",
		Databases:     []string{dbName},
		SchemaVersion: 1,
	}
	raw, _ := json.Marshal(meta)
	meta.Checksum = sha256hex(raw)
	raw, _ = json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(backupDir, "metadata.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
