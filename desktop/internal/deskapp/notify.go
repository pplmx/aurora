package deskapp

import (
	"github.com/egoist/mygo"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// The system-notification seam. Both are variables so a test can watch the
// decision without a desktop session: showing a notification needs the running
// app and, on macOS, a packaged build that has asked for permission.
var (
	// notificationsSupported reports whether the platform can show one at all.
	notificationsSupported = mygo.NotificationsSupported
	// showNotification creates and shows it; a failed Show is reported by the
	// caller that has something to do about it, which is nothing here.
	showNotification = func(opts mygo.NotificationOptions) error {
		return mygo.NewNotification(opts).Show()
	}
	// appIsActive is the front-ness the view asks about before raising one.
	appIsActive = mygo.App.IsActive
)

// Notify shows a task's completion while the window is in the background. The
// view decides when it is worth it; the shell owns the call because showing a
// notification needs the app handle.
//
// A platform that cannot show notifications (an unpackaged macOS build, a Linux
// session without a notification daemon) is not an error: the window still shows
// the result when the operator comes back, and the notification is the extra
// channel, not the only one.
func (s *Shell) Notify(opts nativeui.NotifyOptions) {
	if !notificationsSupported() {
		return
	}
	_ = showNotification(mygo.NotificationOptions{
		ID:    opts.ID,
		Title: opts.Title,
		Body:  opts.Body,
	})
}

// Active reports whether the app is the front one. Before the app runs there is
// no answer and the view must not raise a notification, so an unready app reads
// as active: that is the conservative direction — it suppresses the notice
// instead of showing one for a window nobody can be looking at yet.
func (s *Shell) Active() bool {
	if !mygo.App.IsReady() {
		return true
	}
	return appIsActive()
}

// OpenNotification routes a click on a completion notice back to the window,
// which is what makes the notification useful: it names the task, and clicking
// it brings the page that owns the record to the front.
func (s *Shell) OpenNotification(id string) {
	switch id {
	case "write-done":
		if win := s.Window(); win != nil && !win.IsDestroyed() {
			mygo.App.Focus()
			win.Focus()
		}
	}
}
