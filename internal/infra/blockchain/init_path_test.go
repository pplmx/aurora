package blockchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDBPath_HonorsConfiguredDbPath locks the v1.79 split-brain fix
// (TASK-102, ISS-094): a configured db.path (aurora.toml / env / viper set)
// must be honored by the single canonical resolver everywhere, and InitDB
// must open that same file — not silently diverge back to ./data/aurora.db.
func TestDBPath_HonorsConfiguredDbPath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	custom := filepath.Join(dir, "custom", "chain.db")
	viper.Set("db.path", custom)

	got := DBPath()
	require.Equal(t, custom, got)

	// InitDB must open the SAME configured file (checked via
	// PRAGMA database_list), so the voting singleton and the repositories
	// can no longer target different databases.
	ResetForTest()
	db, err := InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var seq int
	var name, filename string
	require.NoError(t, db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &filename))
	require.Equal(t, custom, filename, "InitDB must open the configured db.path, not the default")
}

// TestDBPath_FallsBackToDefaultWithoutConfig locks that with no configured
// db.path the resolver still returns the default; opening the database owns
// directory creation.
func TestDBPath_FallsBackToDefaultWithoutConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	assert.Equal(t, defaultDBPath, DBPath())
}

// TestDBPath_DoesNotHideOpenErrorsAsAnEmptyPath is the regression for a silent
// data-loss path: when the configured parent could not be created, DBPath used
// to return "". Repository constructors treat "" as a request for a temporary
// SQLite database, so a command could report success while writing to storage
// that vanished when the process exited. The resolver must preserve the
// configured name and leave the real filesystem error to the open/constructor.
func TestDBPath_DoesNotHideOpenErrorsAsAnEmptyPath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("occupied"), 0o600))
	dbPath := filepath.Join(blocker, "aurora.db")
	viper.Set("db.path", dbPath)

	got := DBPath()
	require.Equal(t, dbPath, got, "a directory failure must not turn the database path into the temporary-DB sentinel")

	_, err := OpenDB(got)
	require.Error(t, err, "the underlying open must surface the filesystem failure")
}
