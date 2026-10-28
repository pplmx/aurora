package backup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCopyFileAtomic_SyncFailureDoesNotPromoteATempFile proves that a restored
// database copy reaches stable storage before it can replace a live database.
// A successful close is not a durability guarantee; a power loss can expose a
// renamed but unflushed file as truncated or empty.
func TestCopyFileAtomic_SyncFailureDoesNotPromoteATempFile(t *testing.T) {
	original := syncFileFn
	t.Cleanup(func() { syncFileFn = original })

	syncErr := errors.New("sync failed")
	syncCalls := 0
	syncFileFn = func(*os.File) error {
		syncCalls++
		return syncErr
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "source.db")
	dst := filepath.Join(dir, "restore.tmp")
	require.NoError(t, os.WriteFile(src, []byte("database contents"), 0o600))

	err := copyFileAtomic(src, dst)
	require.ErrorIs(t, err, syncErr)
	require.Equal(t, 1, syncCalls, "the copied data must be synced exactly once")
	require.NoFileExists(t, dst, "a temp that was not made durable must not be left for promotion")
}
