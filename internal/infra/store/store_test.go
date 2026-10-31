package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPath_HonorsConfiguredDbPath locks the v1.79 split-brain fix
// (TASK-102, ISS-094): a configured db.path (aurora.toml / env / viper set)
// must be honored by the single canonical resolver everywhere, and the shared
// handle must open that same file — not silently diverge back to
// ./data/aurora.db.
func TestPath_HonorsConfiguredDbPath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	custom := filepath.Join(dir, "custom", "chain.db")
	viper.Set("db.path", custom)

	got := Path()
	require.Equal(t, custom, got)

	// InitDB must open the SAME configured file (checked via
	// PRAGMA database_list), so the voting singleton and the repositories
	// can no longer target different databases.
	ResetForTest()
	db, err := OpenShared()
	require.NoError(t, err)
	// ResetForTest closes the singleton and clears the once, so the next test in
	// this package does not inherit a closed handle.
	t.Cleanup(ResetForTest)

	var seq int
	var name, filename string
	require.NoError(t, db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &filename))
	require.Equal(t, custom, filename, "InitDB must open the configured db.path, not the default")
}

// TestPath_FallsBackToDefaultWithoutConfig locks that with no configured
// db.path the resolver still returns the default; opening the database owns
// directory creation.
func TestPath_FallsBackToDefaultWithoutConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	assert.Equal(t, DefaultPath, Path())
}

// TestPath_DoesNotHideOpenErrorsAsAnEmptyPath is the regression for a silent
// data-loss path: when the configured parent could not be created, DBPath used
// to return "". Repository constructors treat "" as a request for a temporary
// SQLite database, so a command could report success while writing to storage
// that vanished when the process exited. The resolver must preserve the
// configured name and leave the real filesystem error to the open/constructor.
func TestPath_DoesNotHideOpenErrorsAsAnEmptyPath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("occupied"), 0o600))
	dbPath := filepath.Join(blocker, "aurora.db")
	viper.Set("db.path", dbPath)

	got := Path()
	require.Equal(t, dbPath, got, "a directory failure must not turn the database path into the temporary-DB sentinel")

	_, err := Open(got)
	require.Error(t, err, "the underlying open must surface the filesystem failure")
}

// TestOpen_PreservesSpecialPathCharacters keeps the ledger on the exact file
// named by configuration. Concatenating the driver parameters directly turned
// a '?' in the filename into the DSN separator and opened a different database.
func TestOpen_PreservesSpecialPathCharacters(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("'?' is not a valid Windows filename character")
	}

	dbPath := filepath.Join(t.TempDir(), "ledger?copy.db")
	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var seq int
	var name, filename string
	require.NoError(t, db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &filename))
	require.Equal(t, dbPath, filename)

	var foreignKeys int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys)
}

// TestResetForTestLeavesAConfiguredDatabaseAlone: the helper closes the shared
// handle and cleans the default ./data directory, but a test that pointed db.path
// at its own fixture expects the file to survive — the CLI's tampering tests
// reopen it after the reset. Deleting whatever db.path names broke exactly that.
func TestResetForTestLeavesAConfiguredDatabaseAlone(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	// A sibling test may have left the singleton holding a handle of its own;
	// drop it before opening this fixture.
	ResetForTest()
	t.Cleanup(ResetForTest)

	configured := filepath.Join(dir, "chain.db")
	viper.Set("db.path", configured)
	db, err := OpenShared()
	require.NoError(t, err)
	require.NoError(t, db.Close())

	ResetForTest()
	assert.FileExists(t, configured, "a configured database is not ResetForTest's to delete")

	// The default location is still cleaned, as it always was.
	viper.Set("db.path", filepath.Join("data", "aurora.db"))
	require.NoError(t, os.MkdirAll("data", 0o700))
	ResetForTest()
	_, statErr := os.Stat("data")
	assert.True(t, os.IsNotExist(statErr), "the default ./data directory is removed")
}
