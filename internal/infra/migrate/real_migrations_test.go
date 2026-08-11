package migrate

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realMigrationsDir resolves the repository's migrations/ directory from
// this source file's location (tests must not depend on the process cwd).
func realMigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations"))
}

// TestRealMigrations_Apply is the regression test for the latent migration
// bug: migrations/000001 began with `PRAGMA journal_mode=WAL;`, which
// golang-migrate's sqlite3 driver executes inside an explicit transaction
// where WAL mode cannot be switched → every real migration run failed with
// "cannot change into wal mode from within a transaction". The migrate
// package's own tests used synthetic migrations, so the real files were
// never exercised.
//
// This test applies the ACTUAL checkout migrations and asserts the tables
// they promise exist. It must stay green as long as the migrations apply.
func TestRealMigrations_Apply(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err, "New with real migrations dir")
	defer func() { _ = m.Close() }()

	// Up(0) applies every pending migration (steps<=0 means "all"); Up(N)
	// with N larger than the number of migrations errors with "limit short".
	version, err := m.Up(0)
	require.NoError(t, err, "applying real migrations must not fail")
	require.NotZero(t, version, "expected at least one migration applied")

	t.Logf("migrations applied to version %d", version)

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	for _, table := range []string{
		"blocks", "lottery_records",
		"votes", "voters", "candidates", "voting_sessions",
		"nfts", "nft_operations",
		"tokens", "accounts", "allowances",
		"data_sources", "oracle_data",
	} {
		assertTableExists(t, db, table)
	}
}

// TestRealMigrations_UpTwice_NoChange hammers the idempotency contract:
// running Up a second time must be a no-op (ErrNoChange), not an error.
func TestRealMigrations_UpTwice_NoChange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(0)
	require.NoError(t, err)

	_, err = m.Up(0)
	require.NoError(t, err, "second Up must be a no-op, not an error")
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var n int
	err := db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&n)
	require.NoError(t, err, "query sqlite_master for %s", table)
	assert.Equal(t, 1, n, "migration must create table %s", table)
}
