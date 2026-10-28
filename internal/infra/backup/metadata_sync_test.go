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

// TestBackupService_CreateMetadataSyncFailureRollsBackPreviousSet proves the
// metadata commit record reaches stable storage before its rename. A close
// without a sync can expose an empty or truncated metadata.json after a power
// loss, while the database snapshots have already been promoted.
func TestBackupService_CreateMetadataSyncFailureRollsBackPreviousSet(t *testing.T) {
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

	originalSync := syncFileFn
	t.Cleanup(func() { syncFileFn = originalSync })
	syncErr := errors.New("injected metadata sync failure")
	syncFileFn = func(f *os.File) error {
		if filepath.Base(f.Name()) == "metadata.json.tmp" {
			return syncErr
		}
		return f.Sync()
	}

	_, err = svc.Create(context.Background(), out)
	require.ErrorIs(t, err, syncErr)

	afterDB, err := os.ReadFile(filepath.Join(out, "blockchain.db"))
	require.NoError(t, err)
	require.Equal(t, sha256hex(oldDB), sha256hex(afterDB),
		"a metadata sync failure must restore the previous database snapshot")
	afterMeta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	require.NoError(t, err)
	require.Equal(t, oldMeta, afterMeta, "a metadata sync failure must preserve the previous commit record")
	require.NoError(t, svc.Verify(context.Background(), out))
}
