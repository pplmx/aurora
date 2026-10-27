package nativeui

import (
	"runtime"
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
// native menu is gone on Linux and Windows: the window registers them itself.
// macOS gets them from the application menu, so the test skips there.
func TestActionShortcutsSurviveTheMissingMenu(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS carries these shortcuts on the application menu")
	}
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
