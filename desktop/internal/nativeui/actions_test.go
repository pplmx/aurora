package nativeui

import (
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestSidebarFileMenuRunsFileActions pins the replacement for the hidden
// window menu: backup, export/import and recent exports are reachable from the
// sidebar's File menu, and a chosen item runs exactly the one callback.
func TestSidebarFileMenuRunsFileActions(t *testing.T) {
	app := New(&fakeServices{})
	var calls []string
	app.SetActions(Actions{
		BackUpDatabase:  func() { calls = append(calls, "backup") },
		ExportLotteries: func() { calls = append(calls, "export") },
		ImportLotteries: func() { calls = append(calls, "import") },
		RecentDocuments: func() []string { return []string{"/tmp/draws.json"} },
		OpenRecent:      func(path string) { calls = append(calls, "recent:"+path) },
	})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	fileLabel := i18n.GetText("desktop.menu.file")
	require.True(t, tester.HasText(fileLabel))
	require.NoError(t, tester.Click(fileLabel))
	for _, want := range []string{
		i18n.GetText("desktop.menu.backup"),
		i18n.GetText("desktop.menu.export"),
		i18n.GetText("desktop.menu.import"),
		i18n.GetText("desktop.menu.open_recent"),
	} {
		require.Contains(t, tester.Menu(), want)
	}
	require.NoError(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.backup")))
	require.Equal(t, []string{"backup"}, calls)

	require.NoError(t, tester.Click(fileLabel))
	require.NoError(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.open_recent"), "draws.json"))
	require.Equal(t, []string{"backup", "recent:/tmp/draws.json"}, calls)
}

// TestSidebarActionsMenuDisablesWhatItCannotDo keeps a partially wired window
// honest: an action with no callback is disabled, not a menu item that appears
// to work and silently does nothing.
func TestSidebarActionsMenuDisablesWhatItCannotDo(t *testing.T) {
	app := New(&fakeServices{})
	called := false
	app.SetActions(Actions{BackUpDatabase: func() { called = true }})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	require.NoError(t, tester.Click(i18n.GetText("desktop.menu.file")))
	require.Error(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.export")))
	require.False(t, called)
}

// TestActionShortcutsSurviveTheMissingMenu pins Cmd/Ctrl+E/I/B after the
// native menu was reduced to platform roles: the window registers them on
// every OS, including macOS where no Aurora File items remain.
func TestActionShortcutsSurviveTheMissingMenu(t *testing.T) {
	app := New(&fakeServices{})
	var calls []string
	app.SetActions(Actions{
		ExportLotteries: func() { calls = append(calls, "export") },
		ImportLotteries: func() { calls = append(calls, "import") },
		BackUpDatabase:  func() { calls = append(calls, "backup") },
	})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyE)
	tester.Key(platformRefreshModifier(), ui.KeyI)
	tester.Key(platformRefreshModifier(), ui.KeyB)
	require.Equal(t, []string{"export", "import", "backup"}, calls)
}

// TestFileShortcutsDoNotDelegateToTheNativeMenu prevents the macOS exception
// from creeping back. Its platform File menu now owns only Close, so a runtime
// GOOS branch would silently remove all three Aurora shortcuts there.
func TestFileShortcutsDoNotDelegateToTheNativeMenu(t *testing.T) {
	source, err := os.ReadFile("actions.go")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(source), "runtime.GOOS"),
		"file shortcuts must be window-owned on every platform")
}

// TestFileActionsPauseWhileAWriteOwnsTheDatabase keeps the File menu, its
// shortcuts and the command palette from starting a repository operation while
// the page's background write is still in flight. Navigation stays available.
func TestFileActionsPauseWhileAWriteOwnsTheDatabase(t *testing.T) {
	app := New(&fakeServices{})
	var calls []string
	app.SetActions(Actions{
		BackUpDatabase:  func() { calls = append(calls, "backup") },
		ExportLotteries: func() { calls = append(calls, "export") },
		ImportLotteries: func() { calls = append(calls, "import") },
	})
	app.busy = "Minting…"
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	fileLabel := i18n.GetText("desktop.menu.file")
	require.NoError(t, tester.Click(fileLabel))
	require.Empty(t, tester.Menu(), "a write must disable the File menu")

	tester.Key(platformRefreshModifier(), ui.KeyE)
	tester.Key(platformRefreshModifier(), ui.KeyI)
	tester.Key(platformRefreshModifier(), ui.KeyB)
	require.Empty(t, calls, "file shortcuts must not bypass the busy gate")

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, app.paletteOpen)
	require.Len(t, app.paletteMatches(), len(pageOrder),
		"the palette must keep navigation but omit unavailable commands")
	require.False(t, tester.HasText("Refresh current page"))
	require.False(t, tester.HasText("Back up database…"))
	require.Empty(t, calls)
}

// TestPaletteCopiesDiagnostics: the support copy is reachable from the keyboard
// switcher, not only from the Settings card.
func TestPaletteCopiesDiagnostics(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{
		Version:     "v1.2.3",
		BuildTime:   "2026-10-26T09:00:00Z",
		DBPath:      "/tmp/aurora/aurora.db",
		LedgerFacts: func() (int, string, bool, bool) { return 5, "009c4a0820ab", true, true },
	})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	tester.Type("diagnostics")
	tester.Frame()
	matches := app.paletteMatches()
	require.Len(t, matches, 1)
	require.Equal(t, "Copy Diagnostics", matches[0].title)

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t,
		"Version: v1.2.3\nBuild time: 2026-10-26T09:00:00Z\nDatabase: /tmp/aurora/aurora.db\nLedger: 5 blocks, head 009c4a0820ab — chain valid",
		tester.Clipboard())
	require.False(t, app.paletteOpen)
}

// TestPaletteRunsShowDataFolder covers the file command the palette was
// missing: the folder that holds the database and backups is reachable from
// the keyboard switcher, not only from the File menu.
func TestPaletteRunsShowDataFolder(t *testing.T) {
	app := New(&fakeServices{})
	shown := false
	app.SetActions(Actions{ShowDataFolder: func() { shown = true }})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	tester.Type("folder")
	tester.Frame()
	matches := app.paletteMatches()
	require.Len(t, matches, 1)
	require.Equal(t, "Show data folder", matches[0].title)

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.True(t, shown)
	require.False(t, app.paletteOpen)
}

// TestFileMenuFollowsTheLanguage proves the sidebar's File label follows the
// translator. The native menu used to carry this label; now the sidebar does.
func TestFileMenuFollowsTheLanguage(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	app := New(&fakeServices{})
	app.SetActions(Actions{BackUpDatabase: func() {}})
	tester := ui.NewTester(app.View, 1280, 860)

	tr.SetLocale("en")
	tester.Frame()
	require.True(t, tester.HasText("File"))

	tr.SetLocale("zh")
	tester.Frame()
	require.True(t, tester.HasText("文件"))
	require.NoError(t, tester.Click("文件"))
	require.Contains(t, tester.Menu(), "备份数据库…")
}

// TestNoActionsMeansNoActionsButton keeps view tests and a window built before
// the shell is ready free of a dead control at the bottom of the sidebar.
func TestNoActionsMeansNoActionsButton(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	require.False(t, tester.HasText(i18n.GetText("desktop.menu.file")))
}

// TestPageTitleFollowsNavigation keeps the native window title useful in a
// task switcher even when the app is launched from a terminal.
func TestPageTitleFollowsNavigation(t *testing.T) {
	app := New(&fakeServices{})
	var titles []string
	app.SetActions(Actions{SetPageTitle: func(title string) { titles = append(titles, title) }})
	app.ShowPage("Token")
	app.ShowPage("Oracle")
	require.Equal(t, []string{"Dashboard", "Token", "Oracle"}, titles)
}
