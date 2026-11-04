package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// outboxActions wires the two outbox getters a footer needs.
func outboxActions(pending int, oldest time.Duration) Actions {
	return Actions{
		BackUpDatabase: func() {},
		OutboxPending:  func() (int, bool) { return pending, true },
		OutboxOldest:   func() (time.Duration, bool) { return oldest, true },
	}
}

// TestSidebarReportsAParkedAuditBacklog covers where a backlog is discovered. It
// used to appear in exactly one place — a line inside the copied diagnostics —
// which only shows if the operator opens Settings and clicks Copy. Nobody opens
// Settings to discover a problem.
func TestSidebarReportsAParkedAuditBacklog(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(3, 20*time.Second))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	require.True(t, tester.HasText("Audit outbox: 3 pending, oldest 20s"),
		"a parked backlog must be visible without opening anything")
}

// TestEmptyOutboxSaysNothing keeps a healthy store quiet. A permanent "0 pending"
// line would be one more thing to read on every launch and would train the reader
// to ignore the space where the real warning appears.
func TestEmptyOutboxSaysNothing(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(0, 0))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	require.False(t, tester.HasText("Audit outbox"), "an empty outbox is healthy and says nothing")
}

// TestAStuckBacklogIsAFailure distinguishes a backlog the drainer is still
// moving through from one it has stopped moving. The two need different words:
// a warning during a network blip is routine, while an event parked past the
// retry window means the audit chain is behind the modules with nobody
// addressing it.
func TestAStuckBacklogIsAFailure(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(4, 9*time.Minute))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	require.True(t, tester.HasText("Audit outbox: 4 pending, oldest 9m0s — not draining"),
		"an outlived retry window must say so, not merely report a count")

	line, stuck := outboxLine(OutboxState{Pending: 4, Oldest: 9 * time.Minute, OK: true})
	require.True(t, stuck, "and it must be rendered in the failure tone")
	require.Contains(t, line, "not draining")

	_, stuck = outboxLine(OutboxState{Pending: 4, Oldest: time.Second, OK: true})
	require.False(t, stuck, "a backlog inside the retry window is a warning, not a failure")
}

// TestUnreadableOutboxSaysNothing: the shell reports ok=false when it cannot
// read the outbox — no database yet, or a failed query. Showing "0 pending" for
// that case would claim the audit trail is healthy at the moment nobody can say
// it is.
func TestUnreadableOutboxSaysNothing(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{
		BackUpDatabase: func() {},
		OutboxPending:  func() (int, bool) { return 0, false },
	})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	require.False(t, tester.HasText("Audit outbox"))

	line, stuck := outboxLine(OutboxState{})
	require.Empty(t, line)
	require.False(t, stuck)
}

// TestOutboxBacklogSurvivesAPageSwitch keeps the report where a condition
// belongs. A backlog is not an event on one page: it can last minutes and it
// concerns the whole ledger, so it stays in the fixed footer while the operator
// moves between modules.
func TestOutboxBacklogSurvivesAPageSwitch(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(2, time.Minute))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	for _, target := range []page{pageToken, pageVoting, pageSettings} {
		app.ShowPage(pageTitles[target])
		tester.Frame()
		require.Truef(t, tester.HasText("Audit outbox: 2 pending"),
			"the backlog must still show on %s", pageTitles[target])
	}
}

// TestTheBacklogLineOpensTheAuditLedger makes the report actionable. A reader who
// sees a backlog wants to know what is parked, and the ledger page is where those
// events are visible; a status line nobody can click leaves them to find the page
// themselves, which is the problem this report was added to fix.
//
// It opens the page rather than starting a drain on purpose: the drainer is a
// background retry loop owned by the runtime, and a window that started a second
// pass would race it.
func TestTheBacklogLineOpensTheAuditLedger(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(2, 20*time.Second))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	const line = "Audit outbox: 2 pending, oldest 20s"
	require.True(t, tester.HasText(line))
	require.NoError(t, tester.Click(line))
	tester.Frame()

	require.Equal(t, pageBlockchain, app.page,
		"the backlog line must open the ledger that holds the parked events")
}

// TestTheBacklogLineIsKeyboardReachable covers the reader who never touches a
// pointer. The line is a button, so Tab must reach it and Enter must open the
// ledger; a report that only answers a mouse would leave exactly the user who
// relies on the keyboard unable to act on it.
func TestTheBacklogLineIsKeyboardReachable(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(outboxActions(2, 20*time.Second))
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	const line = "Audit outbox: 2 pending, oldest 20s"
	reached := false
	for range 60 {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused(line) {
			reached = true
			break
		}
	}
	require.True(t, reached, "Tab must reach the backlog line")

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageBlockchain, app.page, "Enter must open the ledger like a click does")
}
