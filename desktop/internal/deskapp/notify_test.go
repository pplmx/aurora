package deskapp

import (
	"errors"
	"os"
	"path/filepath"
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
		ID: "write-done:oracle", Page: "oracle",
		Title: "Aurora finished a task", Body: "Fetching source…",
	})

	require.Len(t, *shown, 1)
	require.Equal(t, "write-done:oracle", (*shown)[0].ID,
		"the id carries the page, so a click can route without a second lookup")
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

// TestStartupWiresTheNotificationHooks holds the binary's side of the feature.
// The view decides when a notice is worth showing and the shell knows how, but
// only main's SetActions call and its click listener connect them: a missing
// field there would disable the behavior with every unit test above still green.
func TestStartupWiresTheNotificationHooks(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "desktop", "cmd", "aurora-desktop", "main.go"))
	require.NoError(t, err)
	main := string(data)

	for _, want := range []string{
		"Notify:          shell.Notify",
		"Active:          shell.Active",
		"mygo.App.OnNotificationClick(shell.OpenNotification)",
	} {
		require.Contains(t, main, want,
			"the startup wiring must connect the notification hooks (missing %q)", want)
	}
}

// TestNotificationClickOpensThePageTheWriteStartedOn makes the notice useful
// rather than merely polite. Clicking brings the window forward and shows the
// page the write started on; raising the window alone would leave the operator
// on whatever page they had moved to, looking for a result the notice named.
func TestNotificationClickOpensThePageTheWriteStartedOn(t *testing.T) {
	originalFocus := appFocus
	t.Cleanup(func() { appFocus = originalFocus })
	focused := 0
	appFocus = func() { focused++ }

	shell := NewShell(About{})
	app := nativeui.New(nil)
	shell.SetApp(app)

	shell.OpenNotification("write-done:oracle")

	require.Equal(t, 1, focused, "the window must come forward")
	require.Equal(t, "Oracle", app.PageTitle(),
		"and the page the write started on must be the visible one")
}

// TestNotificationClickWithoutAPageStillRaisesTheWindow covers the IDs this
// release does not recognise: a notice from another build, or one whose page was
// removed. Bringing the window forward is still better than a click that does
// nothing, and the page must not change on a guess.
func TestNotificationClickWithoutAPageStillRaisesTheWindow(t *testing.T) {
	originalFocus := appFocus
	t.Cleanup(func() { appFocus = originalFocus })
	focused := 0
	appFocus = func() { focused++ }

	shell := NewShell(About{})
	app := nativeui.New(nil)
	app.ShowPage("Token")
	shell.SetApp(app)

	for _, id := range []string{"", "write-done", "something-else:settings"} {
		shell.OpenNotification(id)
	}
	require.Equal(t, 3, focused, "the window comes forward for any notice")
	require.Equal(t, "Token", app.PageTitle(),
		"an unrecognised id must not navigate on a guess")
}
