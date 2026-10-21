package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/deskapp"
)

// TestVersionPrefersThePackagedIdentity is the defect this file exists for:
// `mygo build` supplies its own -ldflags, so the justfile's -X main.Version
// never lands in a packaged binary and `aurora-desktop --version` reported the
// 0.0.1 placeholder while the .deb it came in was versioned from git
// (measured). version() must prefer what MyGo linked in.
func TestVersionPrefersThePackagedIdentity(t *testing.T) {
	original := mygo.App.Version()
	t.Cleanup(func() { mygo.App.SetVersion(original) })

	mygo.App.SetVersion("v9.9.9-packaged")
	require.Equal(t, "v9.9.9-packaged", version(), "a packaged build's version must win")

	mygo.App.SetVersion("")
	require.Equal(t, Version, version(), "an unbundled build falls back to the ldflags value")
}

// TestUserDataDirFollowsTheAppName pins the ordering bug found live: MyGo
// joins the application name into PathUserData, so resolving the data
// directory before App.SetName created ~/.config/MyGo instead of
// ~/.config/Aurora. main must name the app first; this test asserts the
// combination the shell relies on.
func TestUserDataDirFollowsTheAppName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("base directory layout differs on Windows")
	}
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	mygo.App.SetName(deskapp.AppName)
	dir, err := userDataDir()
	require.NoError(t, err)

	require.Equal(t, filepath.Join(base, deskapp.AppName), dir,
		"the data directory must be named after the app, not MyGo's default")
	require.True(t, strings.HasSuffix(dir, string(filepath.Separator)+deskapp.AppName))
	require.DirExists(t, dir, "PathUserData creates the directory on first use")

	// The window-state file MyGo keeps lives in the same directory, which is
	// why the database, the logs and the window geometry now travel together.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "aurora.db"), nil, 0o644))
	require.FileExists(t, filepath.Join(dir, "aurora.db"))
}

// TestStartupFailureDialogCarriesTheReason pins the dialog shown when Start
// fails. A packaged app has no console, so stderr is invisible; the reason and
// its severity both have to survive into the dialog, or a bad --web-root is
// indistinguishable from a double-click that did nothing.
func TestStartupFailureDialogCarriesTheReason(t *testing.T) {
	cause := errors.New(`web root "/tmp/typo" does not contain index.html`)
	opts := startupFailureDialog(cause)

	require.Equal(t, mygo.MessageError, opts.Type, "a failed start is an error, not a notice")
	require.Contains(t, opts.Detail, "/tmp/typo")
	require.Contains(t, opts.Message, "did not open",
		"the message must say what happened: Linux shows it without the title")
}
