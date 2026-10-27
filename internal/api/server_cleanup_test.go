package api

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/blockchain"
	_ "modernc.org/sqlite"
)

func TestNewServer_ClosesOpenedDatabasesWhenInitializationFails(t *testing.T) {
	resetForAPITest(t)
	t.Cleanup(blockchain.ResetForTest)
	viper.Set("api.key", "test-key")

	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	viper.Set("db.path", dbPath)

	seed, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	// The pre-existing votes table makes CREATE TABLE IF NOT EXISTS a no-op,
	// then VotingRepository's voter_pk index fails. It is the first constructor
	// after lottery, so NewServer fails with both earlier handles open.
	_, err = seed.Exec(`CREATE TABLE votes (id TEXT)`)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	srv, err := NewServer()
	require.Error(t, err)
	require.Nil(t, srv)
	require.ErrorContains(t, err, "voter_pk")

	// A fresh handle must be able to open the file after NewServer's cleanup;
	// on Linux the fd assertion below is the stronger proof that no handle
	// leaked.
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, db.Close())

	if runtime.GOOS == "linux" {
		assertNoOpenSQLiteFiles(t, dbPath)
	}
}

func assertNoOpenSQLiteFiles(t *testing.T, dbPath string) {
	t.Helper()

	entries, err := os.ReadDir("/proc/self/fd")
	require.NoError(t, err)

	paths := map[string]struct{}{
		dbPath:          {},
		dbPath + "-wal": {},
		dbPath + "-shm": {},
	}
	var open []string
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			continue
		}
		if _, ok := paths[target]; ok {
			open = append(open, target)
		}
	}
	require.Empty(t, open, "NewServer leaked SQLite file handles")
}
