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

// TestSidebarFileActionScalesWithLargeText keeps the pinned File control from
// squeezing its scaled label into fixed chrome. It shares the action button's
// 34px base height, so at 2x it must grow just as the card actions do.
func TestSidebarFileActionScalesWithLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{BackUpDatabase: func() {}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	file, ok := tester.Find(i18n.GetText("desktop.menu.file"))
	require.True(t, ok)
	require.GreaterOrEqual(t, file.H, float32(68),
		"the pinned File action must scale its full 34px height at 2x")
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

// TestTokenWriteTabsWrapAtLargeText covers the five write tabs at the minimum
// window width. At 2x the built-in single-line tab row placed Burn beyond the
// right edge, so the destructive form had no visible entry point.
func TestTokenWriteTabsWrapAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.ShowPage("Token")
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	rects := make(map[string]ui.Rect)
	for _, label := range []string{"Mint", "Transfer", "Approve", "Transfer From", "Burn"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q tab must be built", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q tab must stay inside the window", label)
		rects[label] = rect
	}
	require.Greater(t, rects["Burn"].Y, rects["Mint"].Y,
		"the tab row must wrap instead of pushing Burn off screen")

	app.scroll.Y = rects["Burn"].Y - 300
	tester.Frame()
	require.NoError(t, tester.Click("Burn"))
	tester.Frame()
	require.Equal(t, 4, app.token.opTab, "the wrapped Burn tab must remain operable")
}

// TestLotteryHistoryStacksAtLargeText keeps every draw fact readable without
// sideways scrolling. At 2x the four-column table put Winners at x=901 in the
// 900px window, and the row itself had no visible action for loading its full
// id into Verify.
func TestLotteryHistoryStacksAtLargeText(t *testing.T) {
	drawID := "draw-0123456789abcdef"
	app := New(&fakeServices{draws: []DrawRow{{
		ID: drawID, BlockHeight: 42,
		Participants: []string{"A", "B", "C"}, Winners: []string{"A"},
	}}})
	app.ShowPage("Lottery")
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, label := range []string{"Participants", "Winners"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q field must be shown", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q must stay inside the window", label)
	}
	action, ok := tester.Find("Use this draw for Verify")
	require.True(t, ok)
	app.scroll.Y = action.Y - 300
	tester.Frame()
	require.NoError(t, tester.Click("Use this draw for Verify"))
	tester.Frame()
	require.Equal(t, drawID, app.lottery.verifyID,
		"the stacked record must still load the full id into Verify")
}

// TestOracleHealthStacksAtLargeText replaces the five-column health table when
// enlarged type no longer fits it. At 2x the Last error header began at x=931,
// outside the 900px window, so the most actionable symptom was inaccessible.
func TestOracleHealthStacksAtLargeText(t *testing.T) {
	app := New(&fakeServices{
		sources: []SourceRow{{
			ID: "source-0123456789abcdef", Name: "BTC feed",
		}},
		healthRows: []HealthRow{{
			SourceID: "source-0123456789abcdef", Name: "BTC feed",
			Attempts: 10, Successes: 9, Failures: 1,
			LastError: "temporary upstream error",
		}},
	})
	app.ShowPage("Oracle")
	app.oracle.tab = 1
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, text := range []string{"Last error", "temporary upstream error"} {
		rect, ok := tester.Find(text)
		require.Truef(t, ok, "%q must be shown", text)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q must stay inside the window", text)
	}
}

// TestQuietButtonsGrowWithLargeText covers copy, dismiss and other low-chrome
// actions. They kept a 26px height while their 12pt label measured 33px high
// at 2x, so the text was vertically clipped inside the hit target.
func TestQuietButtonsGrowWithLargeText(t *testing.T) {
	var textHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		quietButton(c, "Dismiss key")
		_, textHeight = c.MeasureText(0, ui.Span{
			Text: "Dismiss key", Size: textSize(c, 12),
		})
	}, 320, 120)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	button, ok := tester.Find("Dismiss key")
	require.True(t, ok)
	require.GreaterOrEqual(t, button.H, textHeight,
		"the quiet button must be at least as tall as its scaled label")
}

// TestActionButtonsScaleWithLargeText keeps the three button styles from
// packing a 33px label into a 34px box at 2x. The base control is 34px tall,
// so scaling the whole action preserves its normal breathing room instead of
// only preventing a one-pixel clip.
func TestActionButtonsScaleWithLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			primaryButton(c, "Create")
			secondaryButton(c, "Refresh")
			dangerButton(c, "Delete")
		})
	}, 320, 260)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, label := range []string{"Create", "Refresh", "Delete"} {
		button, ok := tester.Find(label)
		require.Truef(t, ok, "%q button must be built", label)
		require.GreaterOrEqualf(t, button.H, float32(68),
			"%q must scale its full 34px height at 2x", label)
	}
}

// TestTextAreasScaleWithLargeText keeps multiline fields showing several lines
// after the system enlarges their type. The fixed 64px box held barely one
// scaled line at 2x, so editing JSON headers or a long description meant
// scrolling through a viewport the height of the caret.
func TestTextAreasScaleWithLargeText(t *testing.T) {
	var area ui.Handle
	var ctx *ui.Context
	value := "{}"
	tester := ui.NewTester(func(c *ui.Context) {
		ctx = c
		textAreaControl(c, &value).Ref(&area)
	}, 420, 360)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	bounds := area.Bounds(ctx)
	require.GreaterOrEqual(t, bounds.H, float32(128),
		"a multiline field must scale its full 64px height at 2x")
}
