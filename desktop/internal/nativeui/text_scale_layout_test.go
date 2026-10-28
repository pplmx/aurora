package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMetricCardKeepsValueAndNoteApartAtLargeText covers the dashboard's
// fixed-height value slot. The value font scales with the OS text setting but
// the slot used not to, so at 2x the large number painted over its label and
// the sentence explaining what it counts.
func TestMetricCardKeepsValueAndNoteApartAtLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(360).Padding(24).Children(func() {
			metricCard(c, "Records", "123", "Exact stored rows", ink, true, pageDashboard)
		})
	}, 420, 300)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	label, ok := tester.Find("Records")
	require.True(t, ok)
	value, ok := tester.Find("123")
	require.True(t, ok)
	note, ok := tester.Find("Exact stored rows")
	require.True(t, ok)

	require.LessOrEqual(t, label.Y+label.H, value.Y,
		"the enlarged value must start below its label")
	require.LessOrEqual(t, value.Y+value.H, note.Y,
		"the note must start below the enlarged value")
}

// TestSidebarScrollsAtLargeText keeps every module reachable in the shortest
// supported window. At 2x the navigation is taller than the fixed sidebar, so
// the Settings row and File action used to paint below the window edge with no
// way to scroll them into view.
func TestSidebarScrollsAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{BackUpDatabase: func() {}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	file, ok := tester.Find(i18n.GetText("desktop.menu.file"))
	require.True(t, ok)
	require.LessOrEqual(t, file.Y+file.H, float32(600),
		"the fixed File action must stay inside the window")

	tester.Scroll(90, 300, 0, 600)
	tester.Frame()
	settings, ok := tester.Find(tr(pageTitles[pageSettings]))
	require.True(t, ok)
	require.GreaterOrEqual(t, settings.Y, float32(0))
	require.LessOrEqual(t, settings.Y+settings.H, float32(600),
		"scrolling the module list must reveal Settings inside the sidebar")

	require.NoError(t, tester.Click(tr(pageTitles[pageSettings])))
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
}

// TestSidebarRevealsKeyboardPageChoiceAtLargeText covers keyboard navigation:
// changing the page must reveal the newly selected row instead of changing an
// item that is still below the visible scroll area.
func TestSidebarRevealsKeyboardPageChoiceAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyComma)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)

	settings, ok := tester.Find(tr(pageTitles[pageSettings]))
	require.True(t, ok)
	require.Less(t, settings.X, sidebarWidthFor(900),
		"the assertion must inspect the sidebar row, not the page heading")
	require.GreaterOrEqual(t, settings.Y, float32(0))
	require.LessOrEqual(t, settings.Y+settings.H, float32(600),
		"the selected module must be revealed in the sidebar")
}
