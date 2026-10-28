package deskapp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreferencesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesName)
	require.NoError(t, savePreferences(path, Preferences{LiveUpdates: true}))

	got := loadPreferences(path)
	require.True(t, got.LiveUpdates)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "a preferences file must stay private")
	}
}

// TestLoadPreferencesIgnoresMalformedFile keeps a cosmetic setting from
// preventing the window from opening; the next save replaces the bad file.
func TestLoadPreferencesIgnoresMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesName)
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	require.False(t, loadPreferences(path).LiveUpdates)
}

func TestSavePreferencesHardensExistingMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows protects the per-user AppData tree with ACLs, not Unix mode bits")
	}

	path := filepath.Join(t.TempDir(), preferencesName)
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
	require.NoError(t, os.Chmod(path, 0o644))

	require.NoError(t, savePreferences(path, Preferences{LiveUpdates: true}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
		"saving preferences must repair a permissive mode left by an older release")
}

func TestShellPersistsLiveUpdates(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	shell := NewShell(About{DBPath: dbPath})
	require.False(t, shell.LiveUpdates())

	require.NoError(t, shell.SetLiveUpdates(true))
	require.True(t, shell.LiveUpdates())

	// A new shell reads the choice back, the way the next launch does.
	require.True(t, NewShell(About{DBPath: dbPath}).LiveUpdates())
}

func TestShellRequiresADatabasePathToSave(t *testing.T) {
	shell := NewShell(About{})
	require.Error(t, shell.SetLiveUpdates(true))
}
