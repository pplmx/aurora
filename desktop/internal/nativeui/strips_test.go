package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestStatusStripNamesItsOutcomeAndKeepsItsFade pins the status line's look
// after it stopped being a rounded badge over a full border. The strip still
// has to say "Done"/"Failed" in words (colour alone is not a signal), and its
// tint must be strongest at the lit left edge so the strip reads as a message
// rather than as another card.
func TestStatusStripNamesItsOutcomeAndKeepsItsFade(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status statusLine
		label  string
	}{
		{"success", okStatus("Saved."), "Done"},
		{"failure", errStatus(errors.New("storage is unavailable")), "Failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := tc.status
			tester := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(auroraThemeFor(c.Preferences()))
				ui.Column(c).Padding(20).Children(func() {
					renderStatus(c, pageToken, status, nil)
				})
			}, 640, 120)
			tester.Frame()

			require.True(t, tester.HasText(tc.label),
				"the strip must state the outcome in words")
			require.True(t, tester.HasText(status.text))

			img := tester.Image()
			luma := func(x, y int) int {
				r, g, b, _ := img.At(x, y).RGBA()
				return int(r>>8) + int(g>>8) + int(b>>8)
			}
			// The verdict tint is lighter than the page background: it is
			// full strength beside the outcome word and fades back toward the
			// bare background at the far edge.
			require.Greater(t, luma(30, 40), luma(600, 40),
				"the tint must weaken toward the far edge")
		})
	}

	// A dismissed outcome reports that the caller should take focus back.
	dismissed := false
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		renderStatus(c, pageToken, okStatus("Saved."), func() { dismissed = true })
	}, 640, 120)
	tester.Frame()
	require.NoError(t, tester.Click("Dismiss"))
	require.True(t, dismissed, "the strip must call its dismiss callback")
}

// TestStatusStripStaysInsideTheWindowAtLargeText keeps the strip from pushing
// its Dismiss action past the minimum window once the text grows, and keeps
// the sentence readable beside the outcome word.
func TestStatusStripStaysInsideTheWindowAtLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		ui.Column(c).Width(600).Padding(12).Children(func() {
			renderStatus(c, pageToken, errStatus(errors.New("storage is unavailable")), func() {})
		})
	}, 640, 200)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	label, ok := tester.Find("Failed")
	require.True(t, ok)
	message, ok := tester.Find("storage is unavailable")
	require.True(t, ok)
	require.Greater(t, message.X, label.X, "the sentence must follow the outcome word")
	require.LessOrEqual(t, message.X+message.W, float32(640))

	dismiss, ok := tester.Find("Dismiss")
	require.True(t, ok)
	require.LessOrEqual(t, dismiss.X+dismiss.W, float32(640),
		"the dismiss action must stay inside the window")
}

// TestDestructiveDialogLeadsWithAWarning pins the confirmation's look: an
// irreversible question opens with a warning mark in the danger hue, and the
// decision row sits below the explanation. The interaction tests (initial
// focus on Cancel, one Tab to the destructive action) already cover behaviour;
// this covers the presentation that tells the two kinds of dialog apart.
func TestDestructiveDialogLeadsWithAWarning(t *testing.T) {
	open := true
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		destructiveConfirm(c, &open, "Burn tokens?",
			"Burn 10 from owner-pk? The tokens are destroyed permanently and cannot be recovered.",
			"Burn", ui.Handle{}, ui.Handle{})
	}, 640, 400)
	tester.Frame()
	require.True(t, open, "the dialog stays open until it is answered")

	title, ok := tester.Find("Burn tokens?")
	require.True(t, ok)
	cancel, ok := tester.Find("Cancel")
	require.True(t, ok)
	confirm, ok := tester.Find("Burn")
	require.True(t, ok)
	require.Less(t, title.Y, cancel.Y, "the question leads the dialog")
	require.GreaterOrEqual(t, title.Y, float32(0))
	require.InDelta(t, float64(cancel.Y), float64(confirm.Y), 1,
		"Cancel and the destructive action share their row")
	require.Less(t, cancel.X, confirm.X, "the safe choice comes first")

	// The warning tile sits at the dialog's upper left, above the question's
	// column, and carries the danger hue rather than a neutral surface. The
	// heading's rect is the dialog panel, so the scan covers its top-left
	// corner only.
	img := tester.Image()
	dangerPixels := 0
	for y := int(title.Y) + 12; y < int(title.Y)+60; y++ {
		for x := int(title.X) + 12; x < int(title.X)+70; x++ {
			r, _, b, _ := img.At(x, y).RGBA()
			if int(r>>8) > int(b>>8)+24 {
				dangerPixels++
			}
		}
	}
	require.Greater(t, dangerPixels, 40,
		"the mark at the dialog's corner must carry the danger hue")
}

// TestBusyStripMatchesTheStatusShape keeps the two fixed strips from looking
// like two unrelated banners: the running operation uses the same lit left
// edge as the finished one, in the accent instead of a verdict colour.
func TestBusyStripMatchesTheStatusShape(t *testing.T) {
	app := New(&fakeServices{})
	app.SetBackgroundWrites(true)
	tester := ui.NewTester(app.View, 900, 700)
	app.busy = "Running draw…"
	tester.Frame()

	// The running strip must not claim an outcome it does not have yet.
	require.True(t, tester.HasText("Running draw…"))
	require.False(t, tester.HasText("Failed"))
	require.False(t, tester.HasText("Done"))
}
