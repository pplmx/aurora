package events

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSQLiteStoresPreserveSpecialDatabasePath ensures the event store and
// replay protection use the escaped shared DSN rather than truncating a
// configured path at its first '?'.
func TestSQLiteStoresPreserveSpecialDatabasePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("'?' is not a valid Windows filename character")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ledger?copy.db")

	store, err := NewSQLiteEventStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	replay, err := NewSQLiteReplayProtection(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = replay.Close() })

	require.FileExists(t, dbPath)
	require.NoFileExists(t, filepath.Join(dir, "ledger"))
}
