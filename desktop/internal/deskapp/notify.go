package deskapp

import (
	"strings"

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
	// appFocus brings the app forward when its notice is clicked.
	appFocus = mygo.App.Focus
)

// notificationPagePrefix marks a completion notice and carries the page the
// write started on, so a click knows where to go without a second lookup that
// would have to survive a restart. MyGo hands the raw ID back on click, also
// for a notice raised by an earlier run, so the ID has to be self-describing.
const notificationPagePrefix = "write-done:"

// notificationFailurePrefix marks a notice about a page that stayed unreadable.
// It shares the page plumbing with the completion notice — a click brings the
// window forward and opens the page — but has its own ID so a failure and a
// completion for the same page do not replace each other.
const notificationFailurePrefix = "page-failed:"

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
	id := opts.ID
	if opts.Page != "" {
		// A failure notice and a completion notice for the same page are
		// different news and must not replace each other.
		prefix := notificationPagePrefix
		if strings.HasPrefix(opts.ID, notificationFailurePrefix) {
			prefix = notificationFailurePrefix
		}
		id = prefix + opts.Page
	}
	_ = showNotification(mygo.NotificationOptions{
		ID:    id,
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

// OpenNotification routes a click on a completion notice back to the record:
// the window comes forward and the page the write started on becomes the visible
// one. Raising the window alone would leave the operator on whatever page they
// had moved to, looking for a result the notice already named.
//
// A page a later release removed, or an ID from another notification, changes
// nothing: bringing the window forward is still better than a click that does
// nothing at all.
func (s *Shell) OpenNotification(id string) {
	// The application comes forward first, whether or not a window is attached:
	// a click that only navigates a hidden window would look like nothing
	// happened.
	appFocus()
	win := s.Window()
	hasWindow := win != nil && !win.IsDestroyed()
	if hasWindow {
		win.Focus()
	}
	// Both notice kinds name a page; which prefix carried it does not change
	// where the click goes.
	page, ok := strings.CutPrefix(id, notificationPagePrefix)
	if !ok {
		page, ok = strings.CutPrefix(id, notificationFailurePrefix)
	}
	if !ok || page == "" {
		return
	}
	if app := s.App(); app != nil {
		app.ShowPage(page)
	}
	if hasWindow {
		win.Invalidate()
	}
}

// SetBadgeCount mirrors a count onto the application icon. Before the app runs
// there is no icon to write to, and the call would post to a main thread that is
// not pumping yet — the same hazard the other app-level calls are guarded for.
func (s *Shell) SetBadgeCount(n int) {
	if !appReady() {
		return
	}
	mygo.App.SetBadgeCount(n)
}
