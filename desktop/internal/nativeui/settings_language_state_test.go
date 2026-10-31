package nativeui

import (
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
	app.actions.SetLanguage = func(string) {}
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
	app.actions.SetLanguage = func(string) {}
	tester := ui.NewTester(app.View, 900, 500)
	app.ShowPage("Settings")
	tester.Frame()

	require.True(t, tester.HasText("简体中文"))
	require.False(t, tester.HasText("Follow System · 简体中文"),
		"without the getter the card must not guess that it follows the system")
}
