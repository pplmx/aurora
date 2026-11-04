package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestZoomLadderMovesOneStepAtATime pins the steps a reader gets from repeated
// presses. An arithmetic 0.1 step drifts (0.8 + 0.1 seven times is 1.4999999),
// so the ratio printed at the end of the ladder was not the ratio in the list
// and the last press could stop before 200%.
func TestZoomLadderMovesOneStepAtATime(t *testing.T) {
	z := float32(1)
	var up []float32
	for range 10 {
		z = nextZoom(z, +1)
		up = append(up, z)
	}
	require.Equal(t, []float32{1.1, 1.25, 1.5, 1.75, 2, 2, 2, 2, 2, 2}, up,
		"the ladder must reach 200% and stay there")
	require.Equal(t, "200%", zoomLabel(z))

	var down []float32
	for range 10 {
		z = nextZoom(z, -1)
		down = append(down, z)
	}
	require.Equal(t, []float32{1.75, 1.5, 1.25, 1.1, 1, 0.9, 0.8, 0.8, 0.8, 0.8}, down,
		"the ladder must reach 80% and stay there")
	require.Equal(t, "80%", zoomLabel(z))
}

// TestZoomAcceptsAValueOffTheLadder keeps a ratio this release no longer lists —
// a hand-edited settings file, a ladder that changed between releases — from
// making a press do nothing. The direction pressed must still be honoured.
func TestZoomAcceptsAValueOffTheLadder(t *testing.T) {
	require.Equal(t, float32(1.25), nextZoom(1.2, +1))
	require.Equal(t, float32(1.1), nextZoom(1.2, -1))
	require.Equal(t, float32(2), nextZoom(1.99, +1))
	require.Equal(t, float32(0.9), nextZoom(0.5, +1), "below the ladder clamps, then moves up from there")
}

// TestZoomClampRejectsCorruptValues keeps a zero from reaching MyGo, which reads
// it as "no zoom" rather than as the corrupt setting it is.
func TestZoomClampRejectsCorruptValues(t *testing.T) {
	nan := float32(0)
	nan = nan / nan
	require.Equal(t, float32(1), clampZoom(nan))
	require.Equal(t, float32(1), clampZoom(0))
	require.Equal(t, float32(0.8), clampZoom(0.1))
	require.Equal(t, float32(2), clampZoom(9))
}

// TestZoomShortcutsChangeTheContentZoom drives the three chords the Settings card
// documents, and proves the ratio is acknowledged on screen: a zoom on a
// fixed-size window changes how much of the page fits, which otherwise reads as
// content that vanished.
func TestZoomShortcutsChangeTheContentZoom(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()
	require.Equal(t, float32(1), app.contentZoom)

	mod := platformRefreshModifier()
	tester.Key(mod, ui.KeyEqual)
	tester.Frame()
	require.Equal(t, float32(1.1), app.contentZoom)
	require.True(t, tester.HasText("Zoom 110%"), "the new ratio must be said on screen")

	tester.Key(mod, ui.KeyMinus)
	tester.Frame()
	require.Equal(t, float32(1), app.contentZoom)

	tester.Key(mod, ui.KeyMinus)
	tester.Frame()
	require.Equal(t, float32(0.9), app.contentZoom)

	tester.Key(mod, ui.Key0)
	tester.Frame()
	require.Equal(t, float32(1), app.contentZoom, "Ctrl/Cmd+0 returns to 100%")
}

// TestContentZoomShrinksTheContentArea proves the shortcut reaches MyGo's own
// content zoom rather than a transform: the view is laid out in a window 1/z the
// size, which is what keeps glyphs sharp at 200%.
func TestContentZoomShrinksTheContentArea(t *testing.T) {
	app := New(&fakeServices{})
	var contentWidth float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		contentWidth, _ = c.Size()
	}, 1000, 700)
	tester.Frame()
	require.Equal(t, float32(1000), contentWidth)

	app.contentZoom = 2
	tester.Frame()
	require.Equal(t, float32(500), contentWidth,
		"a 2x zoom must lay the view out in half the window")
}

// TestSavedZoomIsRestoredOnTheNextLaunch covers the persistence end: the shell's
// stored ratio reaches the window before the first frame, so the zoom does not
// visibly change after launch.
func TestSavedZoomIsRestoredOnTheNextLaunch(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{Zoom: func() float32 { return 1.5 }})
	require.Equal(t, float32(1.5), app.contentZoom)

	// A stored value outside the ladder is clamped rather than trusted.
	other := New(&fakeServices{})
	other.SetActions(Actions{Zoom: func() float32 { return 42 }})
	require.Equal(t, float32(2), other.contentZoom)
}

// TestEveryZoomChangeIsReportedToTheShell keeps the write path connected; a
// window that changed its zoom without telling the shell would look correct for
// the session and revert on the next launch.
func TestEveryZoomChangeIsReportedToTheShell(t *testing.T) {
	app := New(&fakeServices{})
	var saved []float32
	app.SetActions(Actions{SetContentZoom: func(z float32) { saved = append(saved, z) }})

	app.setContentZoom(1.25)
	app.setContentZoom(1.25) // already there: nothing to save
	app.setContentZoom(1.5)

	require.Equal(t, []float32{1.25, 1.5}, saved)
}

// TestZoomWorksOnEveryPage guards where the chord is registered. The header is
// rebuilt for every page, so a shortcut wired into one page's body would work
// exactly where its author tested and nowhere else; zoom is a window command and
// has to answer on all of them, Settings included.
func TestZoomWorksOnEveryPage(t *testing.T) {
	for _, target := range pageOrder {
		t.Run(pageTitles[target], func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, 1280, 860)
			app.ShowPage(pageTitles[target])
			tester.Frame()

			tester.Key(platformRefreshModifier(), ui.KeyEqual)
			tester.Frame()
			require.NotEqualf(t, float32(1), app.contentZoom,
				"%s: the zoom chord must work on this page too", pageTitles[target])
		})
	}
}
