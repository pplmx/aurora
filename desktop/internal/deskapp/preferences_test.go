package deskapp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
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

func TestShellPersistsThemeMode(t *testing.T) {
	original := applyNativeTheme
	t.Cleanup(func() { applyNativeTheme = original })
	var applied []nativeui.ThemeMode
	applyNativeTheme = func(mode nativeui.ThemeMode) {
		applied = append(applied, mode)
	}

	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	shell := NewShell(About{DBPath: dbPath})
	require.Equal(t, nativeui.ThemeMode(""), shell.ThemeMode(),
		"an older file has no theme field")
	require.Equal(t, mygo.ThemeDark, themeSource(shell.ThemeMode()),
		"the missing legacy value must resolve to the previous dark design")

	require.NoError(t, shell.SetThemeMode(nativeui.ThemeModeLight))
	require.Equal(t, nativeui.ThemeModeLight, shell.ThemeMode())
	require.Equal(t, []nativeui.ThemeMode{nativeui.ThemeModeLight}, applied)

	// A new shell reads the choice back, the way the next launch does.
	restarted := NewShell(About{DBPath: dbPath})
	require.Equal(t, nativeui.ThemeModeLight, restarted.ThemeMode())
	restarted.ApplyTheme()
	require.Equal(t, []nativeui.ThemeMode{
		nativeui.ThemeModeLight,
		nativeui.ThemeModeLight,
	}, applied)
}

func TestShellThemeWriteFailureStillAppliesSessionChrome(t *testing.T) {
	original := applyNativeTheme
	t.Cleanup(func() { applyNativeTheme = original })
	var applied []nativeui.ThemeMode
	applyNativeTheme = func(mode nativeui.ThemeMode) {
		applied = append(applied, mode)
	}

	shell := NewShell(About{})
	require.Error(t, shell.SetThemeMode(nativeui.ThemeModeSystem),
		"a shell without a database path cannot save")
	require.Equal(t, nativeui.ThemeMode(""), shell.ThemeMode(),
		"the failed write must not pretend the choice was saved")
	require.Equal(t, []nativeui.ThemeMode{nativeui.ThemeModeSystem}, applied,
		"the session's native chrome still follows the choice")
}

func TestSystemDarkBeforeMyGoStarts(t *testing.T) {
	isDark := func() bool { return false }
	for _, tc := range []struct {
		name   string
		ready  bool
		source mygo.ThemeSource
		want   bool
	}{
		{"light-before-run", false, mygo.ThemeLight, false},
		{"dark-before-run", false, mygo.ThemeDark, true},
		{"system-before-run", false, mygo.ThemeSystem, true},
		{"ready-uses-backend", true, mygo.ThemeDark, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, effectiveSystemDark(tc.ready, tc.source, isDark))
		})
	}
}
