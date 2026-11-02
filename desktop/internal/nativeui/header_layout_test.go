package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPageHeaderKeepsActionUsable is the regression for the narrow-window
// header. A plain row squeezed the purpose sentence into a four-line column and
// clipped the action's chord to two characters: the button still said Refresh,
// but the part that teaches the shortcut was unreadable.
func TestPageHeaderKeepsActionUsable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		width  int
		scale  float32
		stacks bool
	}{
		{"wide, default text", 1280, 1, false},
		{"narrow, default text", 900, 1, false},
		{"wide, large text", 1280, 2, false},
		{"narrow, large text", 760, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, tc.width, 1200)
			tester.SetPreferences(ui.Preferences{TextScale: tc.scale})
			app.ShowPage("Token")
			tester.Frame()

			purpose, ok := tester.Find("Create a token, then mint, send, approve or burn it")
			require.True(t, ok, "the page purpose must be drawn")
			button, ok := tester.Find("Refresh")
			require.True(t, ok, "the header action must be drawn")
			chord, ok := tester.Find("Ctrl+R")
			require.True(t, ok, "the action must show its chord")

			// The chord is the widest part of the action, so it is what a
			// squeezed button loses first. Its measured width has to survive
			// whole, whatever the arrangement.
			require.Greaterf(t, chord.W, float32(30)*tc.scale,
				"the chord was clipped to %.0fpx", chord.W)

			// Nothing may be pushed past the right edge.
			require.LessOrEqual(t, button.X+button.W, float32(tc.width),
				"the action must stay inside the window")

			stacked := button.Y >= purpose.Y+purpose.H
			require.Equalf(t, tc.stacks, stacked,
				"header arrangement at %dpx, %.0fx text", tc.width, tc.scale)
		})
	}
}

// TestPageHeaderStackDoesNotStretchAction guards the arrangement itself: a
// column stretches its children, which turned the button into a page-wide bar
// on the first attempt.
func TestPageHeaderStackDoesNotStretchAction(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 760, 1200)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	app.ShowPage("Token")
	tester.Frame()

	button, ok := tester.Find("Refresh")
	require.True(t, ok)
	require.Lessf(t, button.W, float32(400), "the stacked action must keep its own measure, got %.0fpx", button.W)
}

// TestDisabledButtonDropsHintWhenItCannotFit keeps the hint honest. Both runs
// in the button are single-line, so a row with no room clips them: a 760px
// window at 2x text showed "Complete the re" and lost the rest. The disabled
// fill and muted label already say the action is not ready, so the sentence is
// the first thing to drop.
func TestDisabledButtonDropsHintWhenItCannotFit(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    int
		sc   float32
		want bool
	}{
		{"wide, default text", 1280, 1, true},
		{"narrow, default text", 760, 1, true},
		{"wide, large text", 1280, 2, true},
		{"narrow, large text", 760, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, tc.w, 1200)
			tester.SetPreferences(ui.Preferences{TextScale: tc.sc})
			app.ShowPage("Token")
			tester.Frame()

			hint, ok := tester.Find("Complete the required fields")
			require.Equalf(t, tc.want, ok, "hint presence at %dpx, %.0fx text", tc.w, tc.sc)
			if !ok {
				return
			}
			// A drawn hint must be inside the window, not half of one.
			require.LessOrEqual(t, hint.X+hint.W, float32(tc.w),
				"the hint must not run past the window edge")
		})
	}
}

// TestTabRowWrapsOnlyWhenItMust pins where the tab list stops fitting on one
// row. The active tab's underline is drawn to join the tab to the list's own
// hairline; once the labels take two rows, that underline lands on the first
// row and reads as a divider cutting the remaining tabs off, so the view skips
// it there. What this test protects is the measurement behind that decision:
// if it disagrees with the layout, the underline lands on a wrapped row.
func TestTabRowWrapsOnlyWhenItMust(t *testing.T) {
	labels := []string{"Transfer NFT", "Burn", "History"}
	for _, tc := range []struct {
		name string
		w    int
		sc   float32
		want bool
	}{
		{"narrow, default text", 760, 1, false},
		{"wide, large text", 1280, 2, false},
		{"narrow, large text", 760, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{nftRows: []NFTRow{{ID: "n", Name: "S"}}})
			tester := ui.NewTester(app.View, tc.w, 1700)
			tester.SetPreferences(ui.Preferences{TextScale: tc.sc})
			app.ShowPage("NFT")
			tester.Frame()

			rows := map[float32]bool{}
			for _, label := range labels {
				r, ok := tester.Find(label)
				require.Truef(t, ok, "%q must be drawn", label)
				rows[r.Y] = true
			}
			require.Equalf(t, tc.want, len(rows) > 1,
				"tab rows at %dpx, %.0fx text", tc.w, tc.sc)
		})
	}
}

// TestTabRowWrapsAgreesWithTheLayout is the one that matters: the underline's
// placement depends on tabRowWraps predicting the layout, so the prediction is
// compared against the rows the list actually produced.
func TestTabRowWrapsAgreesWithTheLayout(t *testing.T) {
	labels := []string{"Transfer NFT", "Burn", "History"}
	for _, tc := range []struct {
		name string
		w    int
		sc   float32
	}{
		{"narrow, default text", 760, 1},
		{"wide, large text", 1280, 2},
		{"narrow, large text", 760, 2},
		{"wide, default text", 1280, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{nftRows: []NFTRow{{ID: "n", Name: "S"}}})
			var predicted bool
			tester := ui.NewTester(func(c *ui.Context) {
				predicted = tabRowWraps(c, labels)
				app.View(c)
			}, tc.w, 1700)
			tester.SetPreferences(ui.Preferences{TextScale: tc.sc})
			app.ShowPage("NFT")
			tester.Frame()

			rows := map[float32]bool{}
			for _, label := range labels {
				r, ok := tester.Find(label)
				require.Truef(t, ok, "%q must be drawn", label)
				rows[r.Y] = true
			}
			require.Equalf(t, len(rows) > 1, predicted,
				"tabRowWraps predicted %v but the list drew %d row(s) at %dpx, %.0fx text",
				predicted, len(rows), tc.w, tc.sc)
		})
	}
}

// TestEveryPageHeaderStaysInsideTheWindowAtItsTightest covers the other seven
// pages. The existing check exercises Token, whose purpose sentence is one of the
// shortest, at the widths where the arrangement changes. Settings has no action at
// all and the longest purpose line, so it takes the other branch of header() and
// was never measured at any size.
//
// Every page is checked at the smallest window and the largest text the window
// offers, which is where a header runs out of room first.
func TestEveryPageHeaderStaysInsideTheWindowAtItsTightest(t *testing.T) {
	const width, scale = 900, float32(2)

	for _, target := range pageOrder {
		target := target
		t.Run(pageTitles[target], func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, width, 900)
			tester.SetPreferences(ui.Preferences{TextScale: scale})
			app.ShowPage(pageTitles[target])
			tester.Frame()

			title, ok := tester.Find(pageTitles[target])
			require.True(t, ok, "the page title must be drawn")
			purpose, ok := tester.Find(pagePurpose[target])
			require.True(t, ok, "the page purpose must be drawn")
			for _, rect := range []struct {
				what string
				r    ui.Rect
			}{{"title", title}, {"purpose", purpose}} {
				require.GreaterOrEqualf(t, rect.r.X, float32(0),
					"%s: the %s must not start left of the window", pageTitles[target], rect.what)
				require.LessOrEqualf(t, rect.r.X+rect.r.W, float32(width)+1,
					"%s: the %s must not run past the window", pageTitles[target], rect.what)
			}

			// A page with an action keeps its chord whole; Settings has none.
			// Its Shortcuts card prints the same chord as documentation, so the
			// absence is checked on the header's own button, not on the text.
			if target == pageSettings {
				_, hasRefresh := tester.Find("Refresh")
				require.False(t, hasRefresh,
					"Settings has no stored data to refresh")
				return
			}
			button, ok := tester.Find("Refresh")
			require.True(t, ok, "%s must offer Refresh", pageTitles[target])
			require.LessOrEqualf(t, button.X+button.W, float32(width)+1,
				"%s: the action must stay inside the window", pageTitles[target])
			require.Greaterf(t, button.W, float32(80),
				"%s: the action must keep a usable measure", pageTitles[target])
		})
	}
}
