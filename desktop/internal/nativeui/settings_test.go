package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestSettingsPageCarriesTheApplicationConfiguration covers the page that
// replaces the old Actions entries for language, data folder and About: the
// current language, the database path, the build facts, and one working
// control for each callback.
func TestSettingsPageCarriesTheApplicationConfiguration(t *testing.T) {
	app := New(&fakeServices{})
	var calls []string
	app.SetActions(Actions{
		Version:        "v1.2.3",
		BuildTime:      "2026-10-26T09:00:00Z",
		DBPath:         "/tmp/aurora/aurora.db",
		Language:       func() string { return "zh-Hans" },
		SetLanguage:    func(tag string) { calls = append(calls, "language:"+tag) },
		ShowDataFolder: func() { calls = append(calls, "folder") },
		BackUpDatabase: func() { calls = append(calls, "backup") },
		ShowAbout:      func() { calls = append(calls, "about") },
	})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Settings")
	tester.Frame()

	for _, want := range []string{"Settings", "Language", "Data", "Appearance", "About Aurora", "v1.2.3", "简体中文"} {
		require.Truef(t, tester.HasText(want), "the Settings page must show %q", want)
	}

	require.NoError(t, tester.Click("English"))
	require.NoError(t, tester.Click("Show Data Folder"))
	require.NoError(t, tester.Click("Back Up Database…"))
	require.NoError(t, tester.Click("Open About Dialog"))
	require.Equal(t, []string{"language:en", "folder", "backup", "about"}, calls)

	require.NoError(t, tester.Click("Copy Database path"))
	tester.Frame()
	require.Equal(t, "/tmp/aurora/aurora.db", tester.Clipboard())
}

// TestSettingsPageWithoutCallbacksStaysReadable keeps a window built before
// the shell is ready (and view tests) free of dead buttons: the facts render,
// the callbacks that do not exist are simply absent.
func TestSettingsPageWithoutCallbacksStaysReadable(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()

	require.True(t, tester.HasText("Language"))
	require.True(t, tester.HasText("Data"))
	require.False(t, tester.HasText("English"), "no language buttons without a shell")
	require.False(t, tester.HasText("Open About Dialog"))
}
