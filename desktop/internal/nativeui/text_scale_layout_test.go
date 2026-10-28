package nativeui

import (
	"fmt"
	"math"
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

// TestSidebarGivesLargeEnglishLabelsEnoughRoom keeps the scaled navigation
// from growing its type inside a fixed-width column. At 1.5x and 2x the brand
// subtitle and module names used to be cut to prefixes even though the window
// had room to widen the sidebar.
func TestSidebarGivesLargeEnglishLabelsEnoughRoom(t *testing.T) {
	for _, scale := range []float32{1.5, 2} {
		t.Run(fmt.Sprintf("scale-%g", scale), func(t *testing.T) {
			app := New(&fakeServices{})
			var brandWidth, moduleWidth float32
			tester := ui.NewTester(func(c *ui.Context) {
				app.View(c)
				brandWidth, _ = c.MeasureText(0, ui.Span{
					Text: "Verifiable ledger suite",
					Size: textSize(c, 11),
				})
				moduleWidth, _ = c.MeasureText(0, ui.Span{
					Text: "Blockchain",
					Size: c.Theme().FontSize,
				})
			}, 900, 600)
			tester.SetPreferences(ui.Preferences{TextScale: scale})
			tester.Frame()

			brand, ok := tester.Find("Verifiable ledger suite")
			require.True(t, ok)
			module, ok := tester.Find("Blockchain")
			require.True(t, ok)
			require.GreaterOrEqual(t, brand.W, brandWidth,
				"the brand subtitle needs its measured width at %.1fx", scale)
			require.GreaterOrEqual(t, module.W, moduleWidth,
				"the longest module label needs its measured width at %.1fx", scale)
		})
	}
}

// TestOracleSourceActionsWrapAtLargeText keeps the four per-source actions
// inside their card. At 2x in the minimum window, the unwrapped row pushed the
// destructive Delete button past the right edge, leaving only part of it on
// screen.
func TestOracleSourceActionsWrapAtLargeText(t *testing.T) {
	app := New(&fakeServices{sources: []SourceRow{{
		ID: "source-1", Name: "BTC feed", URL: "https://example.com/price",
		Type: "http", Enabled: true, Interval: 60,
	}}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	app.ShowPage("Oracle")
	tester.Frame()
	app.scroll.Y = math.MaxFloat32
	tester.Frame()

	enable, ok := tester.Find("Enable")
	require.True(t, ok)
	for _, label := range []string{"Enable", "Disable", "Fetch now", "Delete"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q action must be built", label)
		require.GreaterOrEqualf(t, rect.X, float32(0), "%q action starts in the window", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900), "%q action stays in the window", label)
	}
	deleteAction, ok := tester.Find("Delete")
	require.True(t, ok)
	require.Greater(t, deleteAction.Y, enable.Y,
		"the action row must wrap instead of clipping its destructive action")
}

// TestCopyRowKeepsValueVisibleAtLargeText covers one-time secrets such as the
// generated token owner key. At 2x the fixed label and full-width copy action
// left the value with no width at all, so the row offered a copy button but
// showed no trace of what it would copy.
func TestCopyRowKeepsValueVisibleAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.token.ownerPriv = "secret"
	app.ShowPage("Token")
	app.scroll.Y = 900
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	value, ok := tester.Find("secret")
	require.True(t, ok)
	require.Greater(t, value.W, float32(0),
		"the value must keep visible width beside its label and copy action")
	copyAction, ok := tester.Find("Copy Owner private key")
	require.True(t, ok)
	require.LessOrEqual(t, copyAction.X+copyAction.W, float32(900),
		"the copy action must stay inside the window")
}
