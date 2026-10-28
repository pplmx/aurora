package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBackupService_VerifyRecoversInterruptedCreate simulates a process death
// after a new database snapshot replaced the visible one but before its
// metadata commit record did. The previous set is retained in the hidden
// rollback directory; Verify must restore it instead of leaving the operator
// with a checksum failure and no automatic recovery path.
func TestBackupService_VerifyRecoversInterruptedCreate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	oldDB, oldMeta := stageInterruptedCreate(t, svc, dbPath, out)

	require.NoError(t, svc.Verify(context.Background(), out))
	recoveredDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(recoveredDB),
		"recovery must restore the database described by the surviving metadata")
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
}

// TestBackupService_CreateRecoversInterruptedSetBeforeReadingSource proves a
// failed new Create does not leave an earlier backup unverifiable. Even when
// the live source is gone, the old snapshot retained by the interrupted run is
// restored before source validation begins.
func TestBackupService_CreateRecoversInterruptedSetBeforeReadingSource(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	oldDB, oldMeta := stageInterruptedCreate(t, svc, dbPath, out)
	require.NoError(t, os.Remove(dbPath))

	_, err := svc.Create(context.Background(), out)
	require.ErrorContains(t, err, "stat source blockchain")

	recoveredDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(recoveredDB))
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	require.NoError(t, svc.Verify(context.Background(), out))
	requireNoCreateRollbackDirs(t, out)
}

// TestBackupService_VerifyKeepsCommittedSetAndCleansStaleRollback covers a
// crash after metadata.json was promoted but before the rollback directory was
// removed. The committed set must stay in place while the stale copy is
// cleaned up.
func TestBackupService_VerifyKeepsCommittedSetAndCleansStaleRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	destPath := filepath.Join(out, "blockchain.db")
	_, err = stageCreateRollback(out, []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  destPath + ".tmp",
	}})
	require.NoError(t, err)
	require.NotEmpty(t, findRollbackDirs(t, out))

	require.NoError(t, svc.Verify(context.Background(), out))
	requireNoCreateRollbackDirs(t, out)
	require.NoError(t, svc.Verify(context.Background(), out))
}

func TestBackupService_VerifyRecoversInterruptedMultiDatabaseCreate(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "src", "a.db")
	bPath := filepath.Join(dir, "src", "b.db")
	makeTestDB(t, aPath)
	makeTestDB(t, bPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"a": aPath, "b": bPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	oldA, err := os.ReadFile(filepath.Join(out, "a.db"))
	require.NoError(t, err)
	oldB, err := os.ReadFile(filepath.Join(out, "b.db"))
	require.NoError(t, err)
	oldMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)

	for _, path := range []string{aPath, bPath} {
		db, err := sql.Open("sqlite", path)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO test (id) VALUES (2)`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
	}

	snaps := []snapshot{
		{name: "a", destPath: filepath.Join(out, "a.db"), tmpPath: filepath.Join(out, "a.db.tmp")},
		{name: "b", destPath: filepath.Join(out, "b.db"), tmpPath: filepath.Join(out, "b.db.tmp")},
	}
	_, err = stageCreateRollback(out, snaps)
	require.NoError(t, err)
	require.NoError(t, vacuumInto(aPath, snaps[0].tmpPath))
	require.NoError(t, vacuumInto(bPath, snaps[1].tmpPath))
	require.NoError(t, os.Rename(snaps[0].tmpPath, snaps[0].destPath))

	require.NoError(t, svc.Verify(context.Background(), out))
	recoveredA, err := os.ReadFile(filepath.Join(out, "a.db"))
	require.NoError(t, err)
	recoveredB, err := os.ReadFile(filepath.Join(out, "b.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldA), sha256hex(recoveredA))
	require.Equal(t, sha256hex(oldB), sha256hex(recoveredB))
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
}

func findRollbackDirs(t *testing.T, output string) []string {
	t.Helper()
	candidates, err := findCreateRollbackDirs(output)
	require.NoError(t, err)
	dirs := make([]string, len(candidates))
	for i, candidate := range candidates {
		dirs[i] = candidate.path
	}
	return dirs
}

func stageInterruptedCreate(
	t *testing.T,
	svc *BackupService,
	dbPath string,
	out string,
) ([]byte, []byte) {
	t.Helper()
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

	destPath := filepath.Join(out, "blockchain.db")
	newSnapshot := destPath + ".tmp"
	snaps := []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  newSnapshot,
	}}
	rb, err := stageCreateRollback(out, snaps)
	require.NoError(t, err)
	require.NoError(t, vacuumInto(dbPath, newSnapshot))
	require.NoError(t, os.Rename(newSnapshot, destPath))

	// This is the state a SIGKILL leaves: new database bytes under old
	// metadata, with the previous set hidden in the rollback directory.
	rollbackDB, err := os.ReadFile(filepath.Join(rb.dir, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(rollbackDB))
	return oldDB, oldMeta
}
