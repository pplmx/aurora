package nativeui

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// noticeRecorder installs the Notify/Active hooks and records what the view
// decided, so the test is about the decision rather than about the platform.
type noticeRecorder struct {
	shown  []NotifyOptions
	active bool
}

func (r *noticeRecorder) actions() Actions {
	return Actions{
		Active: func() bool { return r.active },
		Notify: func(opts NotifyOptions) { r.shown = append(r.shown, opts) },
	}
}

// finishWrite settles the running background write, reporting how long it ran.
func finishWrite(t *testing.T, app *App, tester *ui.Tester, ran time.Duration) {
	t.Helper()
	app.busySince = time.Now().Add(-ran)
	for i := 0; i < 400 && app.asyncCh != nil; i++ {
		tester.Advance(50 * time.Millisecond)
		tester.Frame()
	}
	require.Nil(t, app.asyncCh, "the write must settle")
}

// TestSlowWriteInTheBackgroundRaisesANotification covers the case the notice
// exists for: a write long enough to be worth waiting for finished while the
// window was behind something else. The result is only drawn in a window nobody
// is looking at, so the operator would otherwise never learn that it finished.
func TestSlowWriteInTheBackgroundRaisesANotification(t *testing.T) {
	recorder := &noticeRecorder{active: false}
	fake := &fakeServices{}
	app := New(fake)
	app.SetActions(recorder.actions())
	tester := ui.NewTester(app.View, 1280, 1400)
	startTokenCreate(t, app, tester)

	finishWrite(t, app, tester, writeNoticeAfter+time.Second)

	require.Len(t, recorder.shown, 1, "a slow write finishing in the background must notify")
	require.Equal(t, "write-done", recorder.shown[0].ID)
	require.NotEmpty(t, recorder.shown[0].Body, "the notice says which task finished")
	require.Contains(t, recorder.shown[0].Body, "Creating token",
		"the task's own label is what tells the operator what was done")
}

// TestFrontWindowSuppressesTheNotification keeps the notice from competing with
// the window it belongs to: the busy bar and the status line already report the
// result there, and a notification over the app it was raised for is noise.
func TestFrontWindowSuppressesTheNotification(t *testing.T) {
	recorder := &noticeRecorder{active: true}
	app := New(&fakeServices{})
	app.SetActions(recorder.actions())
	tester := ui.NewTester(app.View, 1280, 1400)
	startTokenCreate(t, app, tester)

	finishWrite(t, app, tester, writeNoticeAfter+time.Second)

	require.Empty(t, recorder.shown, "the front window already shows the result")
}

// TestQuickWriteDoesNotNotify guards the other half of the threshold. A
// notification for a two-second mint is noise, and noise is what makes people
// turn notifications off, which loses the ones that matter.
func TestQuickWriteDoesNotNotify(t *testing.T) {
	recorder := &noticeRecorder{active: false}
	app := New(&fakeServices{})
	app.SetActions(recorder.actions())
	tester := ui.NewTester(app.View, 1280, 1400)
	startTokenCreate(t, app, tester)

	finishWrite(t, app, tester, time.Second)

	require.Empty(t, recorder.shown, "a write the operator watched must stay quiet")
}

// TestNotificationNeedsTheShell keeps the view usable on its own. Both hooks are
// optional, and a view test (or a window built before the shell is wired) must
// not raise one or panic without them.
func TestNotificationNeedsTheShell(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1400)
	startTokenCreate(t, app, tester)
	require.NotPanics(t, func() { finishWrite(t, app, tester, writeNoticeAfter+time.Second) })
}

// TestNotificationNamesTheFailedWrite keeps a completion notice from describing a
// rejection as a success. The body carries the task label, and the failure keeps
// its own status line, so the two channels agree about what happened.
func TestNotificationNamesTheFailedWrite(t *testing.T) {
	recorder := &noticeRecorder{active: false}
	fake := &fakeServices{createErr: errors.New("storage is unavailable")}
	app := New(fake)
	app.SetActions(recorder.actions())
	tester := ui.NewTester(app.View, 1280, 1400)
	app.SetBackgroundWrites(true)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	tester.Frame()
	require.NotEmpty(t, app.busy)

	finishWrite(t, app, tester, writeNoticeAfter+time.Second)

	require.Len(t, recorder.shown, 1)
	require.Contains(t, recorder.shown[0].Body, "Running draw",
		"the notice names the task that failed")
	require.True(t, app.lottery.status.isErr, "and the page still records the failure")
}
