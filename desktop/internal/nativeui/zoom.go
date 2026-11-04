package nativeui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// The content zoom the window accepts, as a ladder rather than an arithmetic
// step. Adding 0.1 repeatedly drifts (0.8 + 0.1 seven times is 1.4999999), so
// "100%" would never print and a zoom could stop one step short of its end.
// The ends are the ones a reader expects from a browser: 80% to 200%.
var zoomSteps = []float32{0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2}

// nextZoom returns the ladder step dir steps from z. A zoom off the ladder (a
// value from a release that had another one) still moves in the direction
// pressed, because the search compares against the current zoom rather than
// indexing into the list.
func nextZoom(z float32, dir int) float32 {
	z = clampZoom(z)
	if dir > 0 {
		for _, step := range zoomSteps {
			if step > z+0.001 {
				return step
			}
		}
		return zoomSteps[len(zoomSteps)-1]
	}
	for i := len(zoomSteps) - 1; i >= 0; i-- {
		if zoomSteps[i] < z-0.001 {
			return zoomSteps[i]
		}
	}
	return zoomSteps[0]
}

// clampZoom keeps a zoom inside the ladder's span and rejects the values a
// caller can only produce by mistake. MyGo reads a zero (or a NaN) zoom as "no
// zoom", which is not what a corrupt stored setting meant; one is the honest
// reading of it.
func clampZoom(z float32) float32 {
	// Zero and NaN are "no zoom given", not "as small as possible": MyGo reads a
	// zero zoom as none, and a corrupt stored value meant nothing at all.
	if z != z || z <= 0 { // NaN, or no zoom given
		return 1
	}
	low, high := zoomSteps[0], zoomSteps[len(zoomSteps)-1]
	switch {
	case z < low:
		return low
	case z > high:
		return high
	default:
		return z
	}
}

// zoomLabel is the ratio as the window prints it.
func zoomLabel(z float32) string {
	return fmt.Sprintf("%d%%", int(clampZoom(z)*100+0.5))
}

// applyZoomShortcuts handles Ctrl/Cmd + "=" / "-" / "0". Zoom is a window-wide
// shortcut rather than a Settings control because it is reached for while
// reading a page, which is also where a browser and the reference MyGo apps put
// it. KeyEqual is the unshifted key: that is what the "+" key reports on most
// layouts, and requiring Shift would make Ctrl+Plus and Ctrl+= two commands.
func (a *App) applyZoomShortcuts(c *ui.Context) {
	mod := platformRefreshModifier()
	switch {
	case c.Shortcut(mod, ui.KeyEqual):
		a.setContentZoom(nextZoom(a.contentZoom, +1))
	case c.Shortcut(mod, ui.KeyMinus):
		a.setContentZoom(nextZoom(a.contentZoom, -1))
	case c.Shortcut(mod, ui.Key0):
		a.setContentZoom(1)
	}
}

// setContentZoom records the zoom. The visible acknowledgement is left to
// showZoomNotice once per frame: a shortcut fires outside the view build, and a
// toast raised there would be raised again on every frame that re-runs the
// handler.
func (a *App) setContentZoom(z float32) {
	z = clampZoom(z)
	if z == a.contentZoom {
		return
	}
	a.contentZoom = z
	a.zoomNotice = true
	if a.actions.SetContentZoom != nil {
		a.actions.SetContentZoom(z)
	}
}

// showZoomNotice says the ratio once, above the cards, and clears it. It is a
// toast rather than a status line: the status line reports the outcome of an
// operation and the zoom is not one, and the ratio has to be visible wherever
// the reader is scrolled to.
func (a *App) showZoomNotice(c *ui.Context) {
	if !a.zoomNotice {
		return
	}
	c.Toast(fmt.Sprintf(tr("Zoom %s"), zoomLabel(a.contentZoom)))
	a.zoomNotice = false
}
