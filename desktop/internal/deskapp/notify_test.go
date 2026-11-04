package deskapp

import (
	"errors"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// stubNotifications replaces the platform calls and returns the options shown,
// so a test does not need a desktop session or notification permission.
func stubNotifications(t *testing.T, supported bool, active bool) *[]mygo.NotificationOptions {
	t.Helper()
	originalSupported, originalShow, originalActive := notificationsSupported, showNotification, appIsActive
	t.Cleanup(func() {
		notificationsSupported, showNotification, appIsActive = originalSupported, originalShow, originalActive
	})
	notificationsSupported = func() bool { return supported }
	appIsActive = func() bool { return active }
	var shown []mygo.NotificationOptions
	showNotification = func(opts mygo.NotificationOptions) error {
		shown = append(shown, opts)
		return nil
	}
	return &shown
}

// TestShellShowsATaskCompletionNotice covers the display side of the decision the
// view makes: with a notification-capable platform the shell shows exactly the
// notice it was handed, keeping the id a click is routed by.
func TestShellShowsATaskCompletionNotice(t *testing.T) {
	shown := stubNotifications(t, true, false)
	shell := NewShell(About{})

	shell.Notify(nativeui.NotifyOptions{
		ID: "write-done", Title: "Aurora finished a task", Body: "Fetching source…",
	})

	require.Len(t, *shown, 1)
	require.Equal(t, "write-done", (*shown)[0].ID, "the id is how a click finds its page")
	require.Equal(t, "Aurora finished a task", (*shown)[0].Title)
	require.Equal(t, "Fetching source…", (*shown)[0].Body)
}

// TestUnsupportedPlatformSkipsTheNotice keeps a headless or unpackaged session
// quiet: the window still reports the result when the operator returns, so the
// notification is an extra channel rather than the only one, and a platform that
// cannot show one must not turn that into an error.
func TestUnsupportedPlatformSkipsTheNotice(t *testing.T) {
	shown := stubNotifications(t, false, false)
	NewShell(About{}).Notify(nativeui.NotifyOptions{ID: "write-done", Title: "t"})
	require.Empty(t, *shown)
}

// TestFailedShowIsNotFatal: a denied permission must not break the write that
// already succeeded. Nothing in the window depends on the notice having shown.
func TestFailedShowIsNotFatal(t *testing.T) {
	stubNotifications(t, true, false)
	showNotification = func(mygo.NotificationOptions) error { return errors.New("denied") }
	require.NotPanics(t, func() {
		NewShell(About{}).Notify(nativeui.NotifyOptions{ID: "write-done", Title: "t"})
	})
}

// TestActiveBeforeTheAppRunsIsConservative pins the direction of the guess. The
// view asks Active before raising a notice; before Run there is no answer, and
// reading "not active" there would raise a notification for a window that cannot
// even be on screen yet.
func TestActiveBeforeTheAppRunsIsConservative(t *testing.T) {
	stubNotifications(t, true, false)
	require.True(t, NewShell(About{}).Active(),
		"an app that has not started must read as active, suppressing the notice")
}
