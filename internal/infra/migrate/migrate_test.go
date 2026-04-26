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
		os.RemoveAll(tmpDir)
	}

	return dbPath, migDir, cleanup
}

func TestMigrator_New(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer m.Close()

	assert.NotNil(t, m)
}

func TestMigrator_Up(t *testing.T) {
	dbPath, migPath, cleanup := setupTestDB(t)
	defer cleanup()

	m, err := New(dbPath, migPath)
	require.NoError(t, err)
	defer m.Close()

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
	defer m.Close()

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
	defer m.Close()

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
	defer m.Close()

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
	defer m.Close()

	_, err = m.Up(-1)
	require.NoError(t, err)

	status, err := m.Status()
	require.NoError(t, err)
	assert.Equal(t, 0, len(status.PendingVersions))
	assert.Equal(t, 2, len(status.Applied))
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
		defer m.Close()

		version, err := m.Version()
		require.NoError(t, err)
		assert.Equal(t, uint(1), version)
	})
}