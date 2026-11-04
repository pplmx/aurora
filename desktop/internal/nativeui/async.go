package nativeui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// SetBackgroundWrites enables the asynchronous write path used by the real
// desktop. It is opt-in so view tests keep the synchronous contract.
func (a *App) SetBackgroundWrites(on bool) { a.backgroundWrites = on }

// writeOrigin is where a background write began: the page whose status line will
// receive its outcome, and that line's revision before the write. The revision is
// what tells a settled write apart from a line that was already showing when it
// started.
type writeOrigin struct {
	page page
	rev  uint64
}

// runWrite runs a service write. In the desktop it leaves the frame thread
// first, so the busy banner is painted and the window stays responsive; the
// apply closure always runs on the frame thread. Tests keep the synchronous
// path and observe the same state immediately.
func (a *App) runWrite(label string, work func() func()) {
	// One write owns the database at a time, on both paths. The window disables
	// the page body while a write runs, so the synchronous path used to rely on
	// that; a caller that reached it anyway — a keyboard chord, a second click
	// in the same frame, or a test — would have started a second write under a
	// banner naming the first.
	if a.busy != "" {
		return
	}
	if !a.backgroundWrites {
		work()()
		return
	}
	ch := make(chan func(), 1)
	a.asyncCh = ch
	a.origin = writeOrigin{page: a.page, rev: a.statusOf(a.page).revision}
	a.busy = label
	a.busyLabel = label
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
		started := a.busySince
		a.busySince = time.Time{}
		label := a.busyLabel
		a.busyLabel = ""
		origin := a.origin
		a.origin = writeOrigin{}
		apply()
		// A write that ran for a while and finished while the window was in the
		// background raises a system notification: its result is only drawn in
		// a window nobody is looking at.
		a.maybeNotifyWrite(label, started, origin.page)
		// The reader may have moved while the write ran. Its outcome still
		// belongs to the page that started it, but nobody is looking at that
		// page's status line, so the window says it where they are.
		if settled := a.statusOf(origin.page); settled.revision != origin.rev {
			a.settleWrite(c, origin.page, settled)
		}
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
	// message in the same place instead of two different banners. Its wash is
	// the accent at a fraction of the verdict strength, because a running task
	// is progress, not an outcome.
	bar.Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, statusTintFill(c.Theme().Accent.Alpha(0.10)), radiusSmall)
	})
	bar.Role(ui.RoleStatus).Label(text)
	if a.busyAnnouncement != "" {
		c.Announce(a.busyAnnouncement)
		a.busyAnnouncement = ""
	}
}
