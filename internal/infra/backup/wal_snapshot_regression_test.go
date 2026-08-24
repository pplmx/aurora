package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBackupService_Create_IncludesUncheckpointedWAL is the ISS-082 regression:
// the old backup took a best-effort PRAGMA wal_checkpoint then copied the bare
// .db, so any commit that had landed in the still-open -wal (the live server's
// case) was silently missing from the backup. VACUUM INTO snapshots the
// database INCLUDING committed WAL frames, so a WAL-mode source with
// uncheckpointed writes must appear fully in the archive.
func TestBackupService_Create_IncludesUncheckpointedWAL(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "live.db")

	// Keep the writer connection open so its committed frames stay in -wal
	// uncheckpointed when the backup runs (the live-server shape).
	db, err := sql.Open("sqlite3", srcPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t (v) VALUES ('committed-but-uncheckpointed')")
	require.NoError(t, err)

	// Sanity: the write is not yet visible in the bare .db file (it lives only
	// in -wal), which is exactly what the old copy would have missed.
	bare, err := sql.Open("sqlite3", srcPath)
	require.NoError(t, err)
	var before int
	require.NoError(t, bare.QueryRow("SELECT COUNT(*) FROM t").Scan(&before))
	require.Equal(t, 1, before, "the write is committed and visible via the DB handle")
	require.NoError(t, bare.Close())

	svc := NewBackupService(map[string]string{"live": srcPath})
	out := filepath.Join(dir, "backup")
	_, err = svc.Create(context.Background(), out)
	require.NoError(t, err)

	backup, err := sql.Open("sqlite3", filepath.Join(out, "live.db"))
	require.NoError(t, err)
	defer func() { _ = backup.Close() }()

	var got string
	require.NoError(t, backup.QueryRow("SELECT v FROM t WHERE id = 1").Scan(&got))
	require.Equal(t, "committed-but-uncheckpointed", got,
		"the backup must contain commits that were still only in -wal at snapshot time")
}

// TestBackupService_Restore_RemovesStaleWAL is the reverse half of ISS-082: a
// restored archive must not sit next to a stale pre-restore -wal/-shm that
// SQLite would replay against the mismatched .db on next open.
func TestBackupService_Restore_RemovesStaleWAL(t *testing.T) {
	dir := t.TempDir()
	livePath := filepath.Join(dir, "data", "live.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(livePath), 0755))

	// A "current" live DB in WAL mode with a stale WAL file present.
	live, err := sql.Open("sqlite3", livePath)
	require.NoError(t, err)
	_, err = live.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)
	_, err = live.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	require.NoError(t, live.Close())

	// Backup source: a healthy DB with the same shape.
	srcPath := filepath.Join(dir, "archive", "live.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(srcPath), 0755))
	svcSrc := NewBackupService(map[string]string{"live": srcPath})
	// create the archive from a populated DB
	archDB, err := sql.Open("sqlite3", srcPath)
	require.NoError(t, err)
	_, err = archDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	_, err = archDB.Exec("INSERT INTO t (v) VALUES ('restored')")
	require.NoError(t, err)
	require.NoError(t, archDB.Close())

	out := filepath.Join(dir, "backup")
	_, err = svcSrc.Create(context.Background(), out)
	require.NoError(t, err)

	// Point the restore target at a WAL-mode DB that already has -wal/-shm
	// (simulate a prior live server). Keep the writer connection open so the
	// -wal stays uncheckpointed on disk at restore time. Restore must remove
	// the stale -wal so it cannot replay pre-restore frames.
	revive, err := sql.Open("sqlite3", livePath+"?_txlock=immediate")
	require.NoError(t, err)
	_, err = revive.Exec("INSERT INTO t (v) VALUES ('stale-frame')")
	require.NoError(t, err)
	require.FileExists(t, livePath+"-wal", "precondition: a live DB has a WAL file")

	svc := NewBackupService(map[string]string{"live": livePath})
	require.NoError(t, svc.Restore(context.Background(), out))

	require.NoFileExists(t, livePath+"-wal", "restore must not leave a stale -wal next to the restored file")
	require.NoFileExists(t, livePath+"-shm", "restore must not leave a stale -shm next to the restored file")
	require.NoError(t, revive.Close())

	got, err := sql.Open("sqlite3", livePath)
	require.NoError(t, err)
	defer func() { _ = got.Close() }()
	var v string
	require.NoError(t, got.QueryRow("SELECT v FROM t WHERE id = 1").Scan(&v))
	require.Equal(t, "restored", v, "the restored archive is the content served after restore")
}
