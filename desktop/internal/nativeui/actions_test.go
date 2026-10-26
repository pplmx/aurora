package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestSidebarActionsMenuRunsApplicationActions pins the replacement for the
// hidden window menu: the shell's file/help/language work is reachable from
// the sidebar, and a chosen item runs exactly the one callback.
func TestSidebarActionsMenuRunsApplicationActions(t *testing.T) {
	app := New(&fakeServices{})
	var calls []string
	app.SetActions(Actions{
		BackUpDatabase:  func() { calls = append(calls, "backup") },
		ExportLotteries: func() { calls = append(calls, "export") },
		ImportLotteries: func() { calls = append(calls, "import") },
		ShowDataFolder:  func() { calls = append(calls, "folder") },
		ShowAbout:       func() { calls = append(calls, "about") },
		RecentDocuments: func() []string { return []string{"/tmp/draws.json"} },
		OpenRecent:      func(path string) { calls = append(calls, "recent:"+path) },
		SetLanguage:     func(tag string) { calls = append(calls, "language:"+tag) },
		Language:        func() string { return "en" },
	})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	require.True(t, tester.HasText(i18n.GetText("desktop.menu.actions")))
	require.NoError(t, tester.Click(i18n.GetText("desktop.menu.actions")))
	for _, want := range []string{
		i18n.GetText("desktop.menu.backup"),
		i18n.GetText("desktop.menu.export"),
		i18n.GetText("desktop.menu.import"),
		i18n.GetText("desktop.menu.show_data_folder"),
		i18n.GetText("desktop.menu.open_recent"),
		i18n.GetText("desktop.menu.about"),
		i18n.GetText("desktop.menu.language"),
	} {
		require.Contains(t, tester.Menu(), want)
	}
	require.NoError(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.backup")))
	require.Equal(t, []string{"backup"}, calls)

	require.NoError(t, tester.Click(i18n.GetText("desktop.menu.actions")))
	require.NoError(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.open_recent"), "draws.json"))
	require.Equal(t, []string{"backup", "recent:/tmp/draws.json"}, calls)

	require.NoError(t, tester.Click(i18n.GetText("desktop.menu.actions")))
	require.NoError(t, tester.ChooseMenuItem(
		i18n.GetText("desktop.menu.language"),
		i18n.GetText("desktop.menu.language_zh"),
	))
	require.Equal(t, []string{"backup", "recent:/tmp/draws.json", "language:zh-Hans"}, calls)
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

	require.NoError(t, tester.Click(i18n.GetText("desktop.menu.actions")))
	require.Error(t, tester.ChooseMenuItem(i18n.GetText("desktop.menu.export")))
	require.False(t, called)
}

// TestNoActionsMeansNoActionsButton keeps view tests and a window built before
// the shell is ready free of a dead control at the bottom of the sidebar.
func TestNoActionsMeansNoActionsButton(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	require.False(t, tester.HasText(i18n.GetText("desktop.menu.actions")))
}
