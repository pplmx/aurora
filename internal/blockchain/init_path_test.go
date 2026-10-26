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
	info, err := os.Stat(filepath.Dir(custom))
	require.NoError(t, err)
	require.True(t, info.IsDir(), "DBPath must create the configured path's parent dir")

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
// db.path the resolver still returns the default and creates ./data.
func TestDBPath_FallsBackToDefaultWithoutConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	assert.Equal(t, defaultDBPath, DBPath())
	assert.DirExists(t, filepath.Dir(defaultDBPath))
}
