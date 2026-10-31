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
