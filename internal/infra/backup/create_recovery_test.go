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

// TestBackupService_CreateRecoversInterruptedFirstCreate proves an interrupted
// first backup does not poison the output directory forever. There is no prior
// committed archive to verify, so recovery must discard the partial set and let
// Create retry instead of demanding that a rollback set exist.
func TestBackupService_CreateRecoversInterruptedFirstCreate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(out, 0o700))
	destPath := filepath.Join(out, "blockchain.db")
	tmpPath := destPath + ".tmp"
	snaps := []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  tmpPath,
	}}
	_, err := stageCreateRollback(out, snaps)
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(out, "metadata.json"))

	require.NoError(t, vacuumInto(dbPath, tmpPath))
	require.NoError(t, os.Rename(tmpPath, destPath))

	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err = svc.Create(context.Background(), out)
	require.NoError(t, err, "a retry must recover from an interrupted first backup")
	require.NoError(t, svc.Verify(context.Background(), out))
	requireNoCreateRollbackDirs(t, out)
}

// TestBackupService_RecoveryRejectsUnsafeConfiguredDatabaseName proves the
// first-create cleanup path validates configuration names before using them as
// filesystem paths. Create validates names later, but recovery runs first.
func TestBackupService_RecoveryRejectsUnsafeConfiguredDatabaseName(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(out, 0o700))
	victim := filepath.Join(dir, "victim.db")
	require.NoError(t, os.WriteFile(victim, []byte("keep"), 0o600))

	_, err := stageCreateRollback(out, []snapshot{{
		name:     "blockchain",
		destPath: filepath.Join(out, "blockchain.db"),
		tmpPath:  filepath.Join(out, "blockchain.db.tmp"),
	}})
	require.NoError(t, err)

	svc := NewBackupService(map[string]string{
		"../victim": filepath.Join(dir, "live", "victim.db"),
	})
	err = svc.Verify(context.Background(), out)
	require.ErrorContains(t, err, "invalid configured database name")

	content, err := os.ReadFile(victim)
	require.NoError(t, err, "recovery must not touch files outside the backup output")
	require.Equal(t, "keep", string(content))
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

// TestBackupService_VerifyRestoresRollbackOverDuplicateMetadataSet proves
// recovery treats a logically invalid archive as incomplete even when its
// metadata checksum and database bytes are self-consistent.
func TestBackupService_VerifyRestoresRollbackOverDuplicateMetadataSet(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)
	oldMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)

	destPath := filepath.Join(out, "blockchain.db")
	_, err = stageCreateRollback(out, []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  destPath + ".tmp",
	}})
	require.NoError(t, err)
	duplicateMetaDir := filepath.Join(dir, "duplicate-meta")
	require.NoError(t, os.MkdirAll(duplicateMetaDir, 0o700))
	writeMeta(t, duplicateMetaDir, "blockchain", "blockchain")
	metaTmpPath := filepath.Join(out, "metadata.json.duplicate")
	require.NoError(t, copyFileAtomic(filepath.Join(duplicateMetaDir, "metadata.json"), metaTmpPath))
	require.NoError(t, os.Rename(metaTmpPath, filepath.Join(out, "metadata.json")))
	require.NoError(t, syncDirFn(out))

	require.NoError(t, svc.Verify(context.Background(), out))
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
}

// TestBackupService_VerifyKeepsCommittedSetForDifferentConfig proves crash
// recovery is based on whether the archive agrees with its own metadata, not
// whether it matches the currently configured database set. Otherwise a
// complete snapshot can be rolled back merely by running Verify from another
// installation.
func TestBackupService_VerifyKeepsCommittedSetForDifferentConfig(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "src", "blockchain.db")
	makeTestDB(t, dbPath)

	out := filepath.Join(dir, "out")
	svc := NewBackupService(map[string]string{"blockchain": dbPath})
	_, err := svc.Create(context.Background(), out)
	require.NoError(t, err)

	insertTestRow(t, dbPath, 2)
	latest := filepath.Join(dir, "latest")
	_, err = svc.Create(context.Background(), latest)
	require.NoError(t, err)

	destPath := filepath.Join(out, "blockchain.db")
	_, err = stageCreateRollback(out, []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  destPath + ".tmp",
	}})
	require.NoError(t, err)

	// Commit the newer archive while the hidden rollback set is still present,
	// matching a crash after the metadata rename but before cleanup.
	for _, name := range []string{"blockchain.db", "metadata.json"} {
		tmpPath := filepath.Join(out, name+".latest")
		require.NoError(t, copyFileAtomic(filepath.Join(latest, name), tmpPath))
		require.NoError(t, os.Rename(tmpPath, filepath.Join(out, name)))
	}
	require.NoError(t, syncDirFn(out))

	latestDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	latestMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)

	differentConfig := NewBackupService(map[string]string{
		"blockchain": dbPath,
		"extra":      filepath.Join(dir, "extra.db"),
	})
	err = differentConfig.Verify(context.Background(), out)
	require.ErrorContains(t, err, "backup contains 1 databases but 2 are configured")

	recoveredDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(latestDB), sha256hex(recoveredDB),
		"a configured-set mismatch must not roll back a complete committed snapshot")
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, latestMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
}

// TestBackupService_VerifyRepairsInterruptedSetForDifferentConfig proves
// recovery restores the database names recorded by the hidden rollback set
// rather than whichever names the current caller happens to configure.
func TestBackupService_VerifyRepairsInterruptedSetForDifferentConfig(t *testing.T) {
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
	latest := filepath.Join(dir, "latest")
	_, err = svc.Create(context.Background(), latest)
	require.NoError(t, err)

	destPath := filepath.Join(out, "blockchain.db")
	_, err = stageCreateRollback(out, []snapshot{{
		name:     "blockchain",
		destPath: destPath,
		tmpPath:  destPath + ".tmp",
	}})
	require.NoError(t, err)

	// Leave the newer database under the old metadata: recovery must use the
	// rollback set's names even when the caller configures only another DB.
	tmpPath := destPath + ".latest"
	require.NoError(t, copyFileAtomic(filepath.Join(latest, "blockchain.db"), tmpPath))
	require.NoError(t, os.Rename(tmpPath, destPath))
	require.NoError(t, syncDirFn(out))

	differentConfig := NewBackupService(map[string]string{
		"extra": filepath.Join(dir, "extra.db"),
	})
	err = differentConfig.Verify(context.Background(), out)
	require.ErrorContains(t, err, `backup database "blockchain" is not configured`)

	recoveredDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(recoveredDB),
		"recovery must restore the rollback set despite a configuration mismatch")
	recoveredMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, recoveredMeta)
	requireNoCreateRollbackDirs(t, out)
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
