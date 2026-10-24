package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/deskapp"
	"github.com/pplmx/aurora/internal/i18n"
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

// TestBuildTimePrefersTheLinkedValue pins the `just desktop` path: a build
// with -X main.BuildTime must keep exactly what the linker supplied.
func TestBuildTimePrefersTheLinkedValue(t *testing.T) {
	linked := "2026-10-21T11:00:00Z"
	got := resolveBuildTime(linked,
		func() (string, error) { return "", errors.New("must not read resources") },
		func() (string, error) { return "", errors.New("must not stat") },
		func(string) (fs.FileInfo, error) { return nil, errors.New("must not stat") })
	require.Equal(t, linked, got)
}

// TestBuildTimeUsesThePackagedStamp is the packaged-build fix: `mygo build`
// replaces the justfile's ldflags, so BuildTime stays "unknown" and the
// resource written before packaging carries the build identity.
func TestBuildTimeUsesThePackagedStamp(t *testing.T) {
	dir := t.TempDir()
	mygo.App.SetPath(mygo.PathResources, dir)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathResources, "") })
	require.NoError(t, os.WriteFile(filepath.Join(dir, "build-time.txt"),
		[]byte("2026-10-21T11:22:33.456Z\n"), 0o644))

	original := BuildTime
	t.Cleanup(func() { BuildTime = original })
	BuildTime = unknownBuildTime

	require.Equal(t, "2026-10-21T11:22:33Z", buildTime(),
		"the packaged resource must win over the executable timestamp")
}

// TestStampedBuildTimeRejectsAMalformedResource keeps a corrupt stamp from
// reaching --version: the resolver falls back rather than printing junk.
func TestStampedBuildTimeRejectsAMalformedResource(t *testing.T) {
	dir := t.TempDir()
	mygo.App.SetPath(mygo.PathResources, dir)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathResources, "") })

	require.NoError(t, os.WriteFile(filepath.Join(dir, "build-time.txt"), []byte("not a timestamp\n"), 0o644))
	_, err := stampedBuildTime()
	require.Error(t, err, "a malformed stamp must fall through, not reach --version")
}

// TestBuildTimeFallsBackToTheExecutable covers a plain `go build` without
// ldflags: there is no linked value and no packaged resource.
func TestBuildTimeFallsBackToTheExecutable(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "aurora")
	require.NoError(t, os.WriteFile(exe, []byte("binary"), 0o755))
	built := time.Date(2026, 10, 21, 11, 22, 33, 987, time.UTC)
	require.NoError(t, os.Chtimes(exe, built, built))

	got := resolveBuildTime(unknownBuildTime,
		func() (string, error) { return "", os.ErrNotExist },
		func() (string, error) { return exe, nil },
		os.Stat)
	require.Equal(t, "2026-10-21T11:22:33Z", got)
}

// TestBuildTimeStaysUnknownWithoutAnExecutable keeps the fallback honest: a
// failed lookup must not invent a timestamp.
func TestBuildTimeStaysUnknownWithoutAnExecutable(t *testing.T) {
	got := resolveBuildTime(unknownBuildTime,
		func() (string, error) { return "", os.ErrNotExist },
		func() (string, error) { return "", errors.New("no executable") },
		func(string) (fs.FileInfo, error) { return nil, errors.New("must not stat") })
	require.Equal(t, unknownBuildTime, got)
}

// TestGenerationModeSkipsTheShell pins the package-build contract. `mygo
// build` runs the app with MYGO_GENERATE to write a TypeScript client; Aurora
// has no bindings, so the run must return before Start opens the database and
// logs. That failure blocked a real package build on a read-only HOME.
func TestGenerationModeSkipsTheShell(t *testing.T) {
	// MYGO_GENERATE is a path, so this is the same invocation mygo build makes.
	t.Setenv("MYGO_GENERATE", filepath.Join(t.TempDir(), "mygo.ts"))
	require.True(t, generationRequested())

	t.Setenv("MYGO_GENERATE", "")
	require.False(t, generationRequested())
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
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })
	tr.SetLocale("en")

	cause := errors.New(`web root "/tmp/typo" does not contain index.html`)
	opts := startupFailureDialog(cause)

	require.Equal(t, mygo.MessageError, opts.Type, "a failed start is an error, not a notice")
	require.Contains(t, opts.Detail, "/tmp/typo")
	require.Contains(t, opts.Message, "did not open",
		"the message must say what happened: Linux shows it without the title")
}

// TestStartupFailureDialogFollowsTheLocale is the half a failed start is most
// likely to show: the window never opens, so this dialog is the only thing the
// user sees, and it was hardcoded English even when MyGo's role labels were
// Chinese. internal/i18n is initialized in main before Start can fail, so the
// dialog can read it.
func TestStartupFailureDialogFollowsTheLocale(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	opts := startupFailureDialog(errors.New("boom"))
	require.Equal(t, "Aurora 无法启动", opts.Title)
	require.Equal(t, "Aurora 未能打开窗口。", opts.Message)
}

// TestQuitOnLastWindowClosed pins the platform split. MyGo quits when the last
// window closes unless a listener says otherwise, and on macOS that is wrong:
// Cmd+W should close the window and leave the app in the Dock, where clicking
// it reopens the window. A single wrong answer here either kills a macOS app
// on Cmd+W or leaves a windowless process running on Linux and Windows.
func TestQuitOnLastWindowClosed(t *testing.T) {
	assert.True(t, quitsOnLastWindowClosed("linux"))
	assert.True(t, quitsOnLastWindowClosed("windows"))
	assert.False(t, quitsOnLastWindowClosed("darwin"),
		"macOS keeps the app alive after Cmd+W and reopens the window from the Dock")
}

type recordingMainWindow struct {
	icon     []byte
	onDrop   func(*mygo.FileDropEvent)
	focused  bool
	iconCall int
	dropCall int
}

func (w *recordingMainWindow) SetIcon(icon []byte) error {
	w.icon = append([]byte(nil), icon...)
	w.iconCall++
	return nil
}

func (w *recordingMainWindow) OnFileDrop(fn func(*mygo.FileDropEvent)) func() {
	w.onDrop = fn
	w.dropCall++
	return func() {}
}

func (w *recordingMainWindow) Focus() { w.focused = true }

// TestMainWindowRecreationKeepsStateAndWiring pins the lifecycle the macOS
// Dock and second-launch paths use: both windows come from the same
// specification and receive the same bindings. StateKey is what restores the
// saved geometry; losing it made a reopened window forget its size and place.
func TestMainWindowRecreationKeepsStateAndWiring(t *testing.T) {
	intent, ok := deskapp.ParseIntent("aurora://lottery?id=abc")
	require.True(t, ok)
	opts := deskapp.Flags{Intent: intent}

	originalMenu := mygo.App.Menu()
	t.Cleanup(func() { mygo.App.SetMenu(originalMenu) })
	menu := mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}})
	mygo.App.SetMenu(menu)

	shell := deskapp.NewShell(deskapp.About{})
	for i := range 2 {
		win := &mygo.Window{}
		hooks := &recordingMainWindow{}
		var got mygo.WindowOptions

		gotWin := buildMainWindow(shell, opts, func(options mygo.WindowOptions) (*mygo.Window, mainWindowHooks) {
			got = options
			return win, hooks
		})

		require.Same(t, win, gotWin)
		assert.Equal(t, "main", got.StateKey,
			"window %d must use the stable state key", i+1)
		// A native window has no URL: the launch intent selects the view the
		// App renders, and Content is what carries it.
		assert.Empty(t, got.URL, "window %d must not host a web page", i+1)
		require.NotNil(t, got.Content, "window %d must render the native view", i+1)
		assert.Equal(t, deskapp.AppName, got.Title)
		assert.Equal(t, 1280, got.Width)
		assert.Equal(t, 860, got.Height)
		assert.Equal(t, 900, got.MinWidth)
		assert.Equal(t, 600, got.MinHeight)
		assert.Equal(t, "#0c2228", got.BackgroundColor)

		assert.Equal(t, 1, hooks.iconCall, "window %d must get the app icon", i+1)
		assert.NotEmpty(t, hooks.icon)
		assert.Equal(t, 1, hooks.dropCall, "window %d must accept dropped exports", i+1)
		require.NotNil(t, hooks.onDrop)
		hooks.onDrop(&mygo.FileDropEvent{})
		assert.True(t, hooks.focused, "window %d must be focused", i+1)
		assert.Same(t, win, shell.Window())
		assert.Same(t, menu, mygo.App.Menu(),
			"recreating window %d must not replace the application menu", i+1)
	}
}

// TestStartupRequiresWebViewBeforeDatabase pins the order that matters when a
// user launches a package on a machine without WebKitGTK: the app must report
// the missing runtime without first creating a database or logs. A packaged
// launch from a desktop entry has no console, so the only useful report is the
// startup dialog, and it must not leave partial state behind.
func TestStartupRequiresWebViewBeforeDatabase(t *testing.T) {
	var calls []string
	err := startDesktop(
		func() bool { calls = append(calls, "webview"); return false },
		func() error { calls = append(calls, "start"); return nil },
	)
	require.Error(t, err)
	require.Equal(t, []string{"webview"}, calls,
		"a missing webview must stop startup before the database opens")
	require.Contains(t, err.Error(), "no system webview")
}

// TestStartupRunsAfterWebView confirms the successful path still starts the
// shell once the runtime check passes; otherwise the early return would turn
// a missing webview into a reason to skip startup on every machine.
func TestStartupRunsAfterWebView(t *testing.T) {
	var calls []string
	err := startDesktop(
		func() bool { calls = append(calls, "webview"); return true },
		func() error { calls = append(calls, "start"); return nil },
	)
	require.NoError(t, err)
	require.Equal(t, []string{"webview", "start"}, calls)
}
