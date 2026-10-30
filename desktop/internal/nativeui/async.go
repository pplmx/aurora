package nativeui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// SetBackgroundWrites enables the asynchronous write path used by the real
// desktop. It is opt-in so view tests keep the synchronous contract.
func (a *App) SetBackgroundWrites(on bool) { a.backgroundWrites = on }

// runWrite runs a service write. In the desktop it leaves the frame thread
// first, so the busy banner is painted and the window stays responsive; the
// apply closure always runs on the frame thread. Tests keep the synchronous
// path and observe the same state immediately.
func (a *App) runWrite(label string, work func() func()) {
	if !a.backgroundWrites {
		work()()
		return
	}
	if a.asyncCh != nil {
		return // one write at a time; the banner names the running one
	}
	ch := make(chan func(), 1)
	a.asyncCh = ch
	a.busy = label
	a.busySince = time.Now()
	a.busyAnnouncement = tr(label)
	a.busySlowAnnounced = false
	if a.invalidate != nil {
		a.invalidate()
	}
	go func() {
		ch <- work()
	}()
}

// pollAsync applies a completed write and schedules the next poll while one
// runs. It is called at the top of View, on the frame thread.
func (a *App) pollAsync(c *ui.Context) {
	if a.asyncCh == nil {
		return
	}
	select {
	case apply := <-a.asyncCh:
		a.asyncCh = nil
		a.busy = ""
		a.busySince = time.Time{}
		apply()
	default:
		c.After(40 * time.Millisecond)
	}
}

// busyBar is the fixed feedback for a write that has left the frame thread.
// It sits beside the page status so it is visible from any scroll position.
func (a *App) busyBar(c *ui.Context) {
	if a.busy == "" {
		return
	}
	text := tr(a.busy)
	if !a.busySince.IsZero() && time.Since(a.busySince) > 20*time.Second {
		text = tr("Still working…") + " " + tr("This can take a while; the operation will finish on its own.")
		if !a.busySlowAnnounced {
			a.busyAnnouncement, a.busySlowAnnounced = text, true
		}
	}
	bar := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(space3).
		Padding(space2+2, space3+2).Radius(radiusSmall).
		BorderWidth(0, 0, 0, 2).BorderColor(c.Theme().Accent.Alpha(0.72)).Children(func() {
		ui.Spinner(c)
		ui.Text(c, text).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
			Grow(1).Shrink(1).MinWidth(0)
	})
	// The strip shares the status line's shape: a wash that fades to the right
	// and a lit left edge, so "working" and "finished" read as the same kind of
	// message in the same place instead of two different banners.
	bar.Draw(func(p *ui.Painter, r ui.Rect) {
		tint := c.Theme().Accent.Alpha(0.10)
		p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
			{Color: tint, At: 0},
			{Color: tint.Alpha(0.25), At: 0.9},
		}}, radiusSmall)
	})
	bar.Role(ui.RoleStatus).Label(text)
	if a.busyAnnouncement != "" {
		c.Announce(a.busyAnnouncement)
		a.busyAnnouncement = ""
	}
}
