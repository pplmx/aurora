package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// languagePage renders Settings with the two callbacks the language card reads,
// so a test can put the card in either state.
func languagePage(t *testing.T, effective string, follows bool) *ui.Tester {
	t.Helper()
	app := New(&fakeServices{})
	app.actions.Language = func() string { return effective }
	app.actions.SetLanguage = func(string) error { return nil }
	app.actions.FollowsSystem = func() bool { return follows }
	tester := ui.NewTester(app.View, 900, 500)
	app.ShowPage("Settings")
	tester.Frame()
	return tester
}

// TestLanguageCardShowsThatItFollowsTheSystem covers the state the card could
// not express before: following the system on an English machine and choosing
// English deliberately produce the same locale, so a chooser that highlighted
// English either way left the operator unable to see what a later system
// change would do.
func TestLanguageCardShowsThatItFollowsTheSystem(t *testing.T) {
	tester := languagePage(t, "en", true)

	require.True(t, tester.HasText("Follow System · English"),
		"the current row must say the language comes from the system")
	require.True(t, tester.HasText("Follow System"))
}

// TestLanguageCardShowsAnExplicitChoice is the other half: once a language is
// chosen the row names it alone, and the follow-system control is no longer
// the active policy.
func TestLanguageCardShowsAnExplicitChoice(t *testing.T) {
	tester := languagePage(t, "en", false)

	require.False(t, tester.HasText("Follow System · English"),
		"a chosen language must not be described as coming from the system")
	require.True(t, tester.HasText("English"))
}

// TestLanguageCardWithoutAChoiceCallbackStaysPlain keeps a window built before
// the shell supplies FollowsSystem from claiming a policy it cannot know.
func TestLanguageCardWithoutAChoiceCallbackStaysPlain(t *testing.T) {
	app := New(&fakeServices{})
	app.actions.Language = func() string { return "zh-Hans" }
	app.actions.SetLanguage = func(string) error { return nil }
	tester := ui.NewTester(app.View, 900, 500)
	app.ShowPage("Settings")
	tester.Frame()

	require.True(t, tester.HasText("简体中文"))
	require.False(t, tester.HasText("Follow System · 简体中文"),
		"without the getter the card must not guess that it follows the system")
}

// TestLanguageSaveFailureIsReported covers the path that used to be silent.
// The language is applied whether or not the preferences file can be written,
// so a discarded error left the operator with a changed window and no hint
// that the next launch reverts — the same failure the theme control reports.
func TestLanguageSaveFailureIsReported(t *testing.T) {
	app := New(&fakeServices{})
	app.actions.Language = func() string { return "en" }
	app.actions.SetLanguage = func(string) error {
		return errors.New("preferences file is read-only")
	}
	tester := ui.NewTester(app.View, 900, 600)
	app.ShowPage("Settings")
	tester.Frame()

	// Settings draws other controls first, so the language radios take a few
	// tabs to reach; the loop stops as soon as focus is where the test needs it
	// rather than hard-coding how many controls precede them.
	for i := 0; i < 10 && !tester.Focused("English"); i++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	require.True(t, tester.Focused("English"), "Tab must reach the current language")
	tester.Key(0, ui.KeyRight)
	tester.Frame()

	require.True(t, tester.HasText("Couldn't save settings: preferences file is read-only"),
		"a failed write must be visible, not swallowed")
}

// TestLanguageSaveSuccessStaysQuiet is the other side of the same path: a
// write that works must not paint an error, which would train the operator to
// ignore the line that matters.
func TestLanguageSaveSuccessStaysQuiet(t *testing.T) {
	app := New(&fakeServices{})
	app.actions.Language = func() string { return "en" }
	app.actions.SetLanguage = func(string) error { return nil }
	tester := ui.NewTester(app.View, 900, 600)
	app.ShowPage("Settings")
	tester.Frame()

	for i := 0; i < 10 && !tester.Focused("English"); i++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	tester.Key(0, ui.KeyRight)
	tester.Frame()

	require.False(t, tester.HasText("Couldn't save settings"),
		"a successful write must not report a failure")
}
