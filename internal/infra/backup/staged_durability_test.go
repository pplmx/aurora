package backup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCreateSyncsSnapshotBeforePromotion proves VACUUM INTO's new database is
// flushed before its rename can replace the previous verified snapshot.
func TestCreateSyncsSnapshotBeforePromotion(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	originalSyncFile := syncFileFn
	t.Cleanup(func() { syncFileFn = originalSyncFile })
	syncErr := errors.New("injected snapshot sync failure")
	syncFileFn = func(f *os.File) error {
		if strings.HasSuffix(filepath.Clean(f.Name()), ".db.tmp") {
			return syncErr
		}
		return originalSyncFile(f)
	}

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.ErrorIs(t, err, syncErr)

	syncFileFn = originalSyncFile
	require.NoFileExists(t, filepath.Join(out, "blockchain.db"))
	require.NoFileExists(t, filepath.Join(out, "metadata.json"))
}

// TestCreateSyncsRollbackStagingBeforePromotion proves the hidden previous set
// is durable before any visible snapshot is replaced. Losing that directory
// across a crash is exactly what makes the existing interrupted-create
// recovery impossible after a power loss.
func TestCreateSyncsRollbackStagingBeforePromotion(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)
	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	oldDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	oldMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	insertTestRow(t, dbPath, 2)

	originalSyncDir := syncDirFn
	t.Cleanup(func() { syncDirFn = originalSyncDir })
	syncErr := errors.New("injected rollback directory sync failure")
	syncDirFn = func(path string) error {
		if strings.HasPrefix(filepath.Base(path), createRollbackPrefix) {
			return syncErr
		}
		return originalSyncDir(path)
	}

	_, err = svc.Create(context.Background(), out)
	require.ErrorIs(t, err, syncErr)

	syncDirFn = originalSyncDir
	recoveredDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(recoveredDB),
		"no visible snapshot may be promoted before its rollback copy is durable")
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
}

// TestCreateSyncsNewOutputParentDirectories proves a successful desktop-style
// backup persists the new timestamp directory itself, not only its contents.
func TestCreateSyncsNewOutputParentDirectories(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)
	parent := filepath.Join(dir, "backups")
	out := filepath.Join(parent, "aurora-backup-test")

	originalSyncDir := syncDirFn
	t.Cleanup(func() { syncDirFn = originalSyncDir })
	var synced []string
	syncDirFn = func(path string) error {
		synced = append(synced, filepath.Clean(path))
		return originalSyncDir(path)
	}

	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	syncDirFn = originalSyncDir
	require.Contains(t, synced, filepath.Clean(dir),
		"the first new directory entry must be persisted in its existing parent")
	require.Contains(t, synced, filepath.Clean(parent),
		"the timestamp directory entry must be persisted in the new backup parent")
}

// TestCreateReplacesReadOnlyExistingBackup keeps the durability check from
// turning a read-only archive into an update failure when its directory is
// writable. Existing backups are commonly hardened with mode 0400.
func TestCreateReplacesReadOnlyExistingBackup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not use Unix read-only mode bits")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)
	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	for _, name := range []string{"blockchain.db", "metadata.json"} {
		require.NoError(t, os.Chmod(filepath.Join(out, name), 0o400))
	}
	insertTestRow(t, dbPath, 2)

	_, err = svc.Create(context.Background(), out)
	require.NoError(t, err, "a writable directory must be able to replace read-only backup files")
	require.NoError(t, svc.Verify(context.Background(), out))
}

// TestRestoreSyncsPreRestoreSnapshotBeforePromotion proves the manual recovery
// copy has durable bytes before the archive replaces the live database.
func TestRestoreSyncsPreRestoreSnapshotBeforePromotion(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live", "aurora.db")
	makeTestDB(t, livePath)
	backupPath := filepath.Join(dir, "backup")
	svc := NewBackupService(map[string]string{"aurora": livePath})
	_, err := svc.Create(context.Background(), backupPath)
	require.NoError(t, err)
	insertTestRow(t, livePath, 2)

	prePath := filepath.Join(backupPath+".pre_restore", "aurora.db")
	originalSyncFile := syncFileFn
	t.Cleanup(func() { syncFileFn = originalSyncFile })
	syncErr := errors.New("injected pre-restore snapshot sync failure")
	syncFileFn = func(f *os.File) error {
		if filepath.Clean(f.Name()) == filepath.Clean(prePath) {
			return syncErr
		}
		return originalSyncFile(f)
	}

	err = svc.Restore(context.Background(), backupPath)
	require.ErrorIs(t, err, syncErr)

	syncFileFn = originalSyncFile
	requireTestCount(t, livePath, 2, 1, "the live database must remain un-promoted")
}

// TestRestoreRejectsArchiveMutationAfterVerification proves Restore binds the
// bytes it promotes to the metadata it verified. The pre-restore snapshot sync
// is a deterministic point after Verify and before the archive is copied.
func TestRestoreRejectsArchiveMutationAfterVerification(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live", "aurora.db")
	makeTestDB(t, livePath)
	backupPath := filepath.Join(dir, "backup")
	svc := NewBackupService(map[string]string{"aurora": livePath})
	_, err := svc.Create(context.Background(), backupPath)
	require.NoError(t, err)
	insertTestRow(t, livePath, 2)

	prePath := filepath.Join(backupPath+".pre_restore", "aurora.db")
	backupDB := filepath.Join(backupPath, "aurora.db")
	backupMeta := filepath.Join(backupPath, "metadata.json")
	replacementMetaDir := filepath.Join(dir, "replacement-meta")
	require.NoError(t, os.MkdirAll(replacementMetaDir, 0o700))
	writeMeta(t, replacementMetaDir, "aurora")
	replacementMeta := filepath.Join(replacementMetaDir, "metadata.json")
	originalSyncFile := syncFileFn
	t.Cleanup(func() { syncFileFn = originalSyncFile })
	var mutationErr error
	mutated := false
	syncFileFn = func(f *os.File) error {
		if !mutated && filepath.Clean(f.Name()) == filepath.Clean(prePath) {
			mutated = true
			mutationErr = os.WriteFile(backupDB, []byte("replaced after verification"), 0o600)
			if mutationErr == nil {
				mutationErr = os.Remove(backupMeta)
			}
			if mutationErr == nil {
				mutationErr = os.Rename(replacementMeta, backupMeta)
			}
		}
		return originalSyncFile(f)
	}

	err = svc.Restore(context.Background(), backupPath)
	require.NoError(t, mutationErr)
	require.True(t, mutated, "the test must mutate the archive between Verify and copy")
	require.ErrorContains(t, err, "checksum mismatch")

	syncFileFn = originalSyncFile
	requireTestCount(t, livePath, 2, 1, "unverified archive bytes must not replace the live database")
}

// TestRestoreSyncsPreRestoreDirectoryBeforePromotion proves the hidden safety
// copy's directory entries are durable before the archive replaces live data.
func TestRestoreSyncsPreRestoreDirectoryBeforePromotion(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live", "aurora.db")
	makeTestDB(t, livePath)
	backupPath := filepath.Join(dir, "backup")
	svc := NewBackupService(map[string]string{"aurora": livePath})
	_, err := svc.Create(context.Background(), backupPath)
	require.NoError(t, err)
	insertTestRow(t, livePath, 2)

	preDir := backupPath + ".pre_restore"
	originalSyncDir := syncDirFn
	t.Cleanup(func() { syncDirFn = originalSyncDir })
	syncErr := errors.New("injected pre-restore directory sync failure")
	syncDirFn = func(path string) error {
		if filepath.Clean(path) == filepath.Clean(preDir) {
			return syncErr
		}
		return originalSyncDir(path)
	}

	err = svc.Restore(context.Background(), backupPath)
	require.ErrorIs(t, err, syncErr)

	syncDirFn = originalSyncDir
	requireTestCount(t, livePath, 2, 1, "the live database must remain un-promoted")
}

func insertTestRow(t *testing.T, path string, id int) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO test (id) VALUES (?)`, id)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func requireTestCount(t *testing.T, path string, id, want int, msg string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM test WHERE id = ?`, id).Scan(&count))
	require.Equal(t, want, count, msg)
}
