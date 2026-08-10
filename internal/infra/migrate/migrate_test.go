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
// then asks for one more step, which golang-migrate reports as ErrNoChange.
// Up must swallow that sentinel and still report the current version.
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
