package sqlite

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestDSN_PreservesSpecialPathCharacters locks the database identity contract:
// a configured filename is not a string to concatenate a query onto. A literal
// '?' previously became the query separator, so Aurora opened a different file
// with the rest of the path treated as connection parameters.
func TestDSN_PreservesSpecialPathCharacters(t *testing.T) {
	names := []string{"ledger%3Fcopy.db", "ledger#copy.db", "ledger name.db"}
	if runtime.GOOS != "windows" {
		names = append(names, "ledger?copy.db")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), name)
			db, err := sql.Open("sqlite", DSN(dbPath))
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			var seq int
			var databaseName, filename string
			require.NoError(t, db.QueryRow("PRAGMA database_list").Scan(&seq, &databaseName, &filename))
			require.Equal(t, dbPath, filename)

			var foreignKeys int
			require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
			require.Equal(t, 1, foreignKeys, "the encoded path must keep the write DSN parameters")
		})
	}
}

func TestDSN_HandlesRepeatedLeadingSlashInAbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("leading '//' denotes a Windows UNC path")
	}

	dbPath := "/" + filepath.Join(t.TempDir(), "ledger.db")
	db, err := sql.Open("sqlite", DSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var seq int
	var databaseName, filename string
	require.NoError(t, db.QueryRow("PRAGMA database_list").Scan(&seq, &databaseName, &filename))
	require.Equal(t, filepath.Clean(dbPath), filename)

	var foreignKeys int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys)
}

func TestDSN_KeepsMemoryAndExplicitFileDSNs(t *testing.T) {
	require.Equal(t, ":memory:?_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000", DSN(":memory:"))
	require.Equal(t, "file::memory:?_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000", DSN("file::memory:"))
}
