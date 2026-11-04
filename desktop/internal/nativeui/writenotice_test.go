package nativeui

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// settleOnPage runs a background write to completion with the reader on target.
// It returns the tester so a test can inspect what the window shows.
func settleOnPage(t *testing.T, app *App, tester *ui.Tester, target page) {
	t.Helper()
	for i := 0; i < 400 && app.asyncCh != nil; i++ {
		tester.Advance(50 * time.Millisecond)
		tester.Frame()
	}
	require.Nil(t, app.asyncCh, "the write must settle")
}

// startTokenCreate begins the token create write in the background and returns
// once its busy bar is showing.
//
// The write is held open until the returned release function is called. A fake
// service finishes instantly, so a frame could settle the write while the window
// was still on the origin page — and settleWrite deliberately reports nothing
// there, because the status line already says it. The off-page tests then found
// no outcome to look at, and the failure depended on how much work a frame
// happened to do first, which the rest of the suite running in parallel changes.
// Holding the write makes "still running when the reader moves" a fact of the
// test rather than a race it usually wins.
func startTokenCreate(t *testing.T, app *App, tester *ui.Tester) (release func()) {
	t.Helper()
	app.SetBackgroundWrites(true)
	app.ShowPage("Token")
	tester.Frame()

	started := make(chan struct{})
	hold := make(chan struct{})
	app.runWrite("Creating token…", func() func() {
		close(started)
		<-hold
		return func() { app.token.status = okStatus("Created T (T) with supply 100. Owner: owner") }
	})
	<-started
	require.NotEmpty(t, app.busy, "the write must be running in the background")
	tester.Frame()
	return func() { close(hold) }
}

// TestWriteOffPageReportsWhereTheReaderIs covers the gap a background write
// leaves: the page body is disabled but the sidebar is not, so the operator can
// move away while a slow write runs. The outcome then landed in the status line
// of the page they left, which nobody was looking at, and the busy bar simply
// vanished — a write that takes seconds and then reports nowhere reads as one
// that was dropped.
func TestWriteOffPageReportsWhereTheReaderIs(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	release := startTokenCreate(t, app, tester)

	app.ShowPage("Dashboard")
	tester.Frame()
	release()
	settleOnPage(t, app, tester, pageDashboard)

	require.Equal(t, pageDashboard, app.page, "the reader stays where they moved to")
	status := app.token.status.spoken()
	require.NotEmpty(t, status)
	require.True(t, tester.HasText(status),
		"the outcome of a write that finished off-page must be visible where the reader is")
	require.True(t, tester.HasText("Open Token"),
		"and the message must offer the page the record went to")
}

// TestWriteOnItsOwnPageIsNotDuplicated keeps the status strip authoritative where
// it is already visible. A toast there would say the same sentence a second
// time, over the form the operator is reading.
func TestWriteOnItsOwnPageIsNotDuplicated(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	release := startTokenCreate(t, app, tester)
	release()
	settleOnPage(t, app, tester, pageToken)

	status := app.token.status.spoken()
	require.NotEmpty(t, status)
	require.True(t, tester.HasText(status), "the page's own status line reports it")
	require.False(t, tester.HasText("Open Token"),
		"the same page must not also raise a toast about itself")
}

// TestWriteNoticeReturnsToThePage proves the offered action is real: the record
// the operator waited for must be one click away, not merely named.
func TestWriteNoticeReturnsToThePage(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	release := startTokenCreate(t, app, tester)

	app.ShowPage("Dashboard")
	tester.Frame()
	release()
	settleOnPage(t, app, tester, pageDashboard)

	// Click the real button rather than the callback behind it: the action is
	// only useful if it is reachable, and a toast drawn outside the window or
	// covered by one of its siblings would pass a callback-only test.
	require.NoError(t, tester.Click("Open Token"))
	tester.Frame()
	require.Equal(t, pageToken, app.page, "the offered action must open the page")
}

// TestFailedWriteOffPageSaysItFailed keeps a failure from looking like a success
// once it is repeated away from its page. The status line carries the outcome in
// colour and in a word; a notice that dropped that distinction would turn a
// rejected write into an announcement.
func TestFailedWriteOffPageSaysItFailed(t *testing.T) {
	fake := &fakeServices{createErr: errors.New("storage is unavailable")}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	app.SetBackgroundWrites(true)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	tester.Frame()
	require.NotEmpty(t, app.busy)

	app.ShowPage("Dashboard")
	tester.Frame()
	settleOnPage(t, app, tester, pageDashboard)

	require.True(t, app.lottery.status.isErr)
	require.True(t, tester.HasText(app.lottery.status.spoken()),
		"a failure that finished off-page must still reach the reader")
	require.True(t, tester.HasText("Open Lottery"),
		"and the message still names the page the write belongs to")
}

// TestOffPageReportingWorksFromEveryPage generalises the rule to where the reader
// can actually end up. The existing case moves Token → Dashboard; the sidebar
// offers seven destinations, and a toast that only appeared on one of them would
// leave the operator who moved elsewhere with a vanished busy bar and no outcome.
func TestOffPageReportingWorksFromEveryPage(t *testing.T) {
	for _, target := range pageOrder {
		if target == pageToken {
			continue
		}
		t.Run(pageTitles[target], func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, 1280, 1400)
			release := startTokenCreate(t, app, tester)

			app.ShowPage(pageTitles[target])
			tester.Frame()
			release()
			settleOnPage(t, app, tester, target)

			require.Equal(t, target, app.page, "the reader stays where they moved to")
			status := app.token.status.spoken()
			require.NotEmpty(t, status)
			require.Truef(t, tester.HasText(status),
				"the outcome must be visible on %s", pageTitles[target])
			require.Truef(t, tester.HasText("Open Token"),
				"and %s must offer the page the record went to", pageTitles[target])
		})
	}
}
