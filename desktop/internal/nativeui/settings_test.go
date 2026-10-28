package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
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
	require.False(t, tester.HasText("Refresh"), "Settings reads no database to refresh")
	require.False(t, tester.HasText("English"), "no language buttons without a shell")
	require.False(t, tester.HasText("Open About Dialog"))
}

// TestSettingsReportsDesktopAccessibilityState replaces the old static
// "Follows system" values with what the desktop actually asked for. The
// percentage and state labels must update live when the OS preference changes.
func TestSettingsReportsDesktopAccessibilityState(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("zh")
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Settings")
	tester.SetPreferences(ui.Preferences{TextScale: 1.5, HighContrast: true, ReduceMotion: true})
	tester.Frame()

	for _, want := range []string{"150%", "已增强", "已减弱"} {
		require.Truef(t, tester.HasText(want), "Settings must report %q", want)
	}

	tester.SetPreferences(ui.Preferences{TextScale: 1.25})
	tester.Frame()
	for _, want := range []string{"125%", "标准", "已启用"} {
		require.Truef(t, tester.HasText(want), "Settings must update to report %q", want)
	}
}

func TestTextScaleLabel(t *testing.T) {
	for _, tc := range []struct {
		scale float32
		want  string
	}{
		{0, "100%"},
		{1, "100%"},
		{1.25, "125%"},
		{2, "200%"},
	} {
		require.Equal(t, tc.want, textScaleLabel(tc.scale))
	}
}

// TestSettingsStartsFromTheSavedLiveUpdatesChoice proves SetActions applies
// the persisted value before the page is drawn, rather than defaulting off and
// overwriting the file on the first click.
func TestSettingsStartsFromTheSavedLiveUpdatesChoice(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{LiveUpdates: func() bool { return true }})
	require.True(t, app.liveUpdates)
}

// TestSettingsSavesLiveUpdates pins the write path: the runtime choice and the
// persisted one change together, and a success leaves no error banner.
func TestSettingsSavesLiveUpdates(t *testing.T) {
	app := New(&fakeServices{})
	var saved bool
	app.SetActions(Actions{
		LiveUpdates:    func() bool { return false },
		SetLiveUpdates: func(on bool) error { saved = on; return nil },
	})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()

	require.NoError(t, tester.Click("Every 10 seconds"))
	tester.Frame()
	require.True(t, saved)
	require.True(t, app.liveUpdates)
	require.False(t, app.settingsStatus.isErr)
}

// TestSettingsShowsAPreferenceWriteFailure keeps a failed save from looking
// like a success that will silently reset on the next launch.
func TestSettingsShowsAPreferenceWriteFailure(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{
		LiveUpdates:    func() bool { return false },
		SetLiveUpdates: func(bool) error { return errors.New("disk full") },
	})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Settings")
	tester.Frame()

	require.NoError(t, tester.Click("Every 10 seconds"))
	tester.Frame()
	require.True(t, app.settingsStatus.isErr)
	require.Contains(t, app.settingsStatus.text, "disk full")
	require.True(t, tester.HasText("Couldn't save settings: disk full"))
}
