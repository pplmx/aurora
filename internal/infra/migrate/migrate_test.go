package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (string, string, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migDir := filepath.Join(tmpDir, "migrations")
	err := os.MkdirAll(migDir, 0755)
	require.NoError(t, err)

	upMig := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	downMig := `DROP TABLE IF EXISTS users;`

	err = os.WriteFile(filepath.Join(migDir, "000001_init.up.sql"), []byte(upMig), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(migDir, "000001_init.down.sql"), []byte(downMig), 0644)
	require.NoError(t, err)

	v2Up := `CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL
	);`
	v2Down := `DROP TABLE IF EXISTS posts;`
	err = os.WriteFile(filepath.Join(migDir, "000002_add_posts.up.sql"), []byte(v2Up), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(migDir, "000002_add_posts.down.sql"), []byte(v2Down), 0644)
	require.NoError(t, err)

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return dbPath, migDir, cleanup
}

func TestMigrator_New(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	assert.NotNil(t, m)
}

// TestMigrator_New_DSNHardened guards the parallel-safety fix (TASK-170,
// ISS-165): New must open SQLite with the same hardened DSN every repository
// uses (_busy_timeout=5000, _txlock=immediate). Previously it opened with a
// bare `?_foreign_keys=ON`, whose default busy timeout is 0 — a migrate
// run against a DB another CLI/API process held the write lock on failed
// instantly with "database is locked" instead of waiting up to 5s. The PRAGMA
// below reads back the value the DSN param set on the real connection.
func TestMigrator_New_DSNHardened(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	var busyTimeout int
	require.NoError(t, m.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout))
	assert.Equal(t, 5000, busyTimeout,
		"migrate must wait up to 5s for a concurrent writer, not fail with SQLITE_BUSY (ISS-165)")
}

func TestMigrator_Up(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	version, err := m.Up(1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), version)

	version, err = m.Up(1)
	require.NoError(t, err)
	assert.Equal(t, uint(2), version)
}

func TestMigrator_Down(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(-1)
	require.NoError(t, err)

	version, err := m.Down(1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), version)
}

func TestMigrator_Status(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, uint(0), status.Current)
	assert.False(t, status.Dirty)
	assert.Equal(t, 2, len(status.PendingVersions))

	_, err = m.Up(1)
	require.NoError(t, err)

	status, err = m.Status()
	require.NoError(t, err)
	assert.Equal(t, uint(1), status.Current)
	assert.Equal(t, 1, len(status.PendingVersions))
	assert.Equal(t, 1, len(status.Applied))
}

func TestMigrator_Version(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	version, err := m.Version()
	require.NoError(t, err)
	assert.Equal(t, uint(0), version)

	_, err = m.Up(-1)
	require.NoError(t, err)

	version, err = m.Version()
	require.NoError(t, err)
	assert.Equal(t, uint(2), version)
}

func TestMigrator_Close(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)

	err = m.Close()
	assert.NoError(t, err)
}

func TestMigrator_AllMigrationsApplied(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(-1)
	require.NoError(t, err)

	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, 0, len(status.PendingVersions))
	assert.Equal(t, 2, len(status.Applied))
}

// TestMigrator_New_InvalidMigrationPath proves New surfaces a failure to
// open the migration source (e.g. a misconfigured migrations path) instead
// of silently constructing a broken migrator.
func TestMigrator_New_InvalidMigrationPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	m, err := New(dbPath, filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err, "New should fail when the migrations dir does not exist")
	if m != nil {
		_ = m.Close()
	}
}

// TestMigrator_Up_NoChangeExercised drives a fully-applied migration set and
// then asks for one more step. Up must treat "nothing pending" as success and
// still report the current version.
func TestMigrator_Up_NoChangeExercised(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	version, err := m.Up(-1)
	require.NoError(t, err)
	assert.Equal(t, uint(2), version)

	// No pending migrations left: Up again to hit the ErrNoChange branch.
	version, err = m.Up(-1)
	require.NoError(t, err)
	assert.Equal(t, uint(2), version, "Up on a fully-applied DB stays at the current version")
}

// TestMigrator_Down_Full exercises the steps<=0 Down path all the way back to
// schema version 0, including the ErrNilVersion handling on the final Version
// read (no migration history yet).
func TestMigrator_Down_Full(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(-1)
	require.NoError(t, err)

	version, err := m.Down(-1)
	require.NoError(t, err)
	assert.Equal(t, uint(0), version, "fully rolled back DB reports version 0")
}

// TestMigrator_Status_IgnoresUnparsableFiles proves getPendingMigrations
// skips directory entries that are not versioned .up.sql files (e.g. a README
// or a file whose leading integer fails to parse) rather than erroring out.
func TestMigrator_Status_IgnoresUnparsableFiles(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Stray files that must not be treated as pending migrations.
	require.NoError(t, os.WriteFile(filepath.Join(migPath, "README.txt"), []byte("docs"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(migPath, "migration_notes.up.sql"), []byte("--n/a"), 0644))

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, []uint{1, 2}, status.PendingVersions,
		"only the two real migrations should be pending; stray files are ignored")
}

func TestMigrator_DB(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	assert.NotNil(t, m.DB(), "DB should expose the live database handle")
}

// TestRunMigrationsIfEnabled_NoMigPathErr proves the default migrations path
// is attempted (and in this test fails loudly because ./migrations is absent)
// rather than silently skipping migration.
func TestRunMigrationsIfEnabled_NoMigPathErr(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	err := RunMigrationsIfEnabled(dbPath, MigrateConfig{AutoMigrate: true})
	require.Error(t, err, "enabled auto-migration with a missing default path must surface an error")
}

func TestRunMigrationsIfEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migDir := filepath.Join(tmpDir, "migrations")
	err := os.MkdirAll(migDir, 0755)
	require.NoError(t, err)

	upMig := `CREATE TABLE IF NOT EXISTS test_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	downMig := `DROP TABLE IF EXISTS test_table;`
	err = os.WriteFile(filepath.Join(migDir, "000001_test.up.sql"), []byte(upMig), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(migDir, "000001_test.down.sql"), []byte(downMig), 0644)
	require.NoError(t, err)

	t.Run("auto migrate disabled", func(t *testing.T) {
		err := RunMigrationsIfEnabled(dbPath, MigrateConfig{
			AutoMigrate: false,
			MigPath:     migDir,
		})
		assert.NoError(t, err)
	})

	t.Run("auto migrate enabled", func(t *testing.T) {
		dbPath2 := filepath.Join(tmpDir, "test2.db")
		err := RunMigrationsIfEnabled(dbPath2, MigrateConfig{
			AutoMigrate: true,
			MigPath:     migDir,
		})
		assert.NoError(t, err)

		m, err := New(dbPath2, migDir)
		require.NoError(t, err)
		defer func() { _ = m.Close() }()

		version, err := m.Version()
		require.NoError(t, err)
		assert.Equal(t, uint(1), version)
	})
}

// TestMigrator_RejectsDuplicateVersionFiles proves a second up/down file for
// the same version fails at construction. Otherwise map iteration order would
// silently decide which SQL runs, and the loser would never be audited.
func TestMigrator_RejectsDuplicateVersionFiles(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
	}{
		{name: "up", file: "000001_duplicate.up.sql"},
		{name: "down", file: "000001_duplicate.down.sql"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbPath, migPath, cleanup := setupTestDB(t)
			defer cleanup()
			require.NoError(t, os.WriteFile(
				filepath.Join(migPath, tc.file), []byte("SELECT 1;"), 0644,
			))

			m, err := New(dbPath, migPath)
			require.ErrorContains(t, err, "duplicate "+tc.name+" migration")
			if m != nil {
				_ = m.Close()
			}
		})
	}
}

// TestMigrator_DirtyDatabaseRefusesMigrations proves the dirty bit survives
// an interrupted migration and blocks both directions until an operator
// inspects the database. Silently retrying could apply half a schema twice.
func TestMigrator_DirtyDatabaseRefusesMigrations(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	require.NoError(t, m.setVersion(1, true))

	_, err = m.Up(0)
	require.ErrorContains(t, err, "database is dirty at version 1")
	_, err = m.Down(1)
	require.ErrorContains(t, err, "database is dirty at version 1")

	status, err := m.Status()
	require.NoError(t, err)
	assert.True(t, status.Dirty)
	assert.Equal(t, uint(1), status.Current)
	require.NoError(t, m.Close())
}

// TestMigrator_MissingDownFileFailsLoudly proves rollback never skips a
// version merely because its down file is absent: that would leave a schema
// object behind while recording an older version.
func TestMigrator_MissingDownFileFailsLoudly(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()
	require.NoError(t, os.Remove(filepath.Join(migPath, "000002_add_posts.down.sql")))

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	_, err = m.Up(0)
	require.NoError(t, err)

	_, err = m.Down(1)
	require.ErrorContains(t, err, "migration 2 has no down file")
}

// TestMigrator_ConcurrentOpenersSerialize proves two processes racing to
// migrate the same database leave exactly one schema version and apply each
// migration once. The hardened DSN's BEGIN IMMEDIATE + busy timeout is what
// turns SQLITE_BUSY from an immediate failure into a short wait.
func TestMigrator_ConcurrentOpenersSerialize(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			m, err := New(dbPath, migPath)
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = m.Close() }()
			_, err = m.Up(0)
			errs <- err
		}()
	}
	close(start)
	for range 2 {
		require.NoError(t, <-errs)
	}

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, uint(2), status.Current)
	assert.False(t, status.Dirty)
	assert.Empty(t, status.PendingVersions)
}

// TestMigrator_FailedMigrationLeavesDirtyVersion proves a failed migration
// records the target version as dirty before executing its SQL. A retry must
// refuse to run rather than pretend the half-applied schema is current.
func TestMigrator_FailedMigrationLeavesDirtyVersion(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()
	require.NoError(t, os.WriteFile(
		filepath.Join(migPath, "000001_init.up.sql"),
		[]byte("CREATE TABLE users (id INTEGER PRIMARY KEY); THIS IS NOT SQL;"),
		0644,
	))

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(1)
	require.ErrorContains(t, err, "migration 1 up failed")

	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, uint(1), status.Current)
	assert.True(t, status.Dirty, "a failed migration must be marked dirty")

	_, err = m.Up(1)
	require.ErrorContains(t, err, "database is dirty at version 1")
}
