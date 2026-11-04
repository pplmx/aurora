package nativeui

import "time"

// Notifying a write that finished while the window was in the background.
//
// The window is the only place a result shows, and a desktop app is routinely
// behind something else: an import of a large file, a network fetch or a slow
// signature can run for seconds after the operator switched to a browser. When
// they come back, the busy bar is gone and the status line says nothing about
// how long it took — and if they never come back, they never learn the write
// finished at all.
//
// The reference apps built on MyGo treat this as part of running work in the
// background at all: a session that needs you raises a system notification. The
// same rule applied here: a write that took long enough to be worth waiting for,
// and finished while the window was not in front, raises one.

// writeNoticeAfter is how long a write must run before its completion is worth a
// system notification. Below it the write finished while the operator was still
// watching — a notification for a two-second mint would be noise, and noise is
// what makes people turn notifications off.
const writeNoticeAfter = 10 * time.Second

// NotifyOptions is one system notification the shell should show. The view
// names it rather than calling MyGo directly: the view package has no window or
// app handle, and a test can then observe the decision without a desktop.
type NotifyOptions struct {
	// ID identifies the notification on the platform, so a second write from
	// the same page replaces the first notice instead of stacking.
	ID string
	// Page is the page the write started on, as its stable id. A click brings
	// that page forward: a notification that only raises the window leaves the
	// operator to remember which page the record went to.
	Page  string
	Title string
	Body  string
}

// maybeNotifyWrite raises the completion notice when the write ran long enough
// and the window is not in front. It is a no-op without a notify callback, which
// is how every view test runs and how the app behaves before the shell wires it.
func (a *App) maybeNotifyWrite(label string, started time.Time, origin page) {
	if a.actions.Notify == nil || a.actions.Active == nil {
		return
	}
	if started.IsZero() || time.Since(started) < writeNoticeAfter {
		return
	}
	// The app is in front: the busy bar and the status line already say this,
	// and a notification on top of the window it belongs to is noise.
	if a.actions.Active() {
		return
	}
	a.actions.Notify(NotifyOptions{
		ID:    "write-done:" + pageIDs[origin],
		Page:  pageIDs[origin],
		Title: tr("Aurora finished a task"),
		Body:  tr(label),
	})
}
