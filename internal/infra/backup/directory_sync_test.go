package backup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBackupService_CreateSyncsOutputDirectoryBeforeSuccess proves a completed
// create persists the metadata rename, not just the file contents. Without the
// directory flush, a power loss can roll back the rename after Create returned
// and leave the previous metadata beside newly promoted database snapshots.
func TestBackupService_CreateSyncsOutputDirectoryBeforeSuccess(t *testing.T) {
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

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO test (id) VALUES (2)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	originalSyncDir := syncDirFn
	t.Cleanup(func() { syncDirFn = originalSyncDir })
	syncErr := errors.New("injected directory sync failure")
	syncDirFn = func(path string) error {
		if filepath.Clean(path) == filepath.Clean(out) {
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
		"a failed directory flush must roll back the promoted snapshots")
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	require.NoError(t, svc.Verify(context.Background(), out))
}

// TestBackupService_RestoreDirectorySyncFailureRollsBack proves a successful
// rename is not treated as a successful restore until the live directory entry
// is durable. If that flush fails, the pre-restore snapshot is put back.
func TestBackupService_RestoreDirectorySyncFailureRollsBack(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live", "aurora.db")
	makeTestDB(t, livePath)
	backupPath := filepath.Join(dir, "backup")
	svc := NewBackupService(map[string]string{"aurora": livePath})
	_, err := svc.Create(context.Background(), backupPath)
	require.NoError(t, err)

	db, err := sql.Open("sqlite", livePath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO test (id) VALUES (2)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	originalSyncDir := syncDirFn
	t.Cleanup(func() { syncDirFn = originalSyncDir })
	syncErr := errors.New("injected live directory sync failure")
	syncDirFn = func(path string) error {
		if filepath.Clean(path) == filepath.Clean(filepath.Dir(livePath)) {
			return syncErr
		}
		return originalSyncDir(path)
	}

	err = svc.Restore(context.Background(), backupPath)
	require.ErrorIs(t, err, syncErr)

	syncDirFn = originalSyncDir
	db, err = sql.Open("sqlite", livePath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM test WHERE id = 2`).Scan(&count))
	require.Equal(t, 1, count, "a failed durable promotion must restore the pre-restore database, not the archive")
}
