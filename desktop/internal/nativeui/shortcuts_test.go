package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPageShortcutsFollowTheSidebarOrder holds the promise the Settings page makes
// when it lists "Pages 1…8": the number a reader learns from the sidebar is the one
// that opens that page. The two lists are separate — pageOrder is the keyboard
// order and moduleOrder the sidebar's module section — so a page inserted into one
// without the other would send the shortcut somewhere the reader can see it is not.
func TestPageShortcutsFollowTheSidebarOrder(t *testing.T) {
	require.Len(t, pageOrder, 8, "the shortcut range is 1…8, so the order must have eight pages")
	require.LessOrEqual(t, len(pageOrder), 9, "digits 1 through 9 are the whole range")

	for i, want := range pageOrder {
		app := New(workbenchFakes())
		tester := ui.NewTester(app.View, 1280, 2400)
		// Start somewhere that is not the target, so a shortcut that did nothing
		// could not pass.
		start := pageSettings
		if want == pageSettings {
			start = pageDashboard
		}
		app.ShowPage(pageTitles[start])
		tester.Frame()

		tester.Key(platformRefreshModifier(), ui.Key(int(ui.Key1)+i))
		tester.Frame()

		require.Equalf(t, want, app.page,
			"shortcut %d must open %s, the page in position %d of the sidebar",
			i+1, pageTitles[want], i+1)
	}
}

// TestPageShortcutsAndTheCommaKeyAgree covers the settings shortcut, which has a
// key of its own because the Application section sits outside the numbered range a
// reader counts through.
func TestPageShortcutsAndTheCommaKeyAgree(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyComma)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
}
