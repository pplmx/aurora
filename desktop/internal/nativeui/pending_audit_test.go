package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// pendingFixture wires the parked-event list the ledger reads.
func pendingFixture(events ...PendingEvent) Actions {
	return Actions{
		BackUpDatabase: func() {},
		OutboxList:     func() []PendingEvent { return events },
	}
}

// TestTheLedgerListsWhatIsWaiting covers the question the footer's count cannot
// answer. "Four pending" is not something an operator can act on; a module, the
// operation it belongs to and how many delivery attempts have failed are.
func TestTheLedgerListsWhatIsWaiting(t *testing.T) {
	now := time.Now()
	app := New(&fakeServices{blocks: []BlockRow{{Height: 2, Module: "token", Summary: "x"}}})
	app.SetActions(pendingFixture(
		PendingEvent{
			EventType: "token.mint", Module: "token", Aggregate: "GLD",
			ParkedAt: now.Add(-2 * time.Minute), Attempts: 3, NextTryAt: now.Add(30 * time.Second),
		},
		PendingEvent{
			EventType: "oracle.fetch", Module: "oracle", Aggregate: "btc",
			ParkedAt: now.Add(-20 * time.Minute), Attempts: 9, NextTryAt: now.Add(-time.Minute),
		},
	))
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Blockchain")
	tester.Frame()

	for _, want := range []string{
		"Waiting to be delivered", // the card names itself
		"GLD", "btc",              // the operations the copies belong to
		"Operation", "Attempts", "Next try",
	} {
		require.Truef(t, tester.HasText(want), "the parked list must show %q", want)
	}
}

// TestTheWaitingCardIsAbsentWhenNothingWaits keeps a healthy store quiet. The
// ledger page is read constantly; a permanent empty section would be another
// heading to skip on every visit.
func TestTheWaitingCardIsAbsentWhenNothingWaits(t *testing.T) {
	for _, actions := range []Actions{
		{BackUpDatabase: func() {}},
		pendingFixture(),
	} {
		app := New(&fakeServices{blocks: []BlockRow{{Height: 2, Module: "token", Summary: "x"}}})
		app.SetActions(actions)
		tester := ui.NewTester(app.View, 1280, 2400)
		app.ShowPage("Blockchain")
		tester.Frame()
		require.False(t, tester.HasText("Waiting to be delivered"),
			"an empty outbox must not add a section to the ledger")
	}
}

// TestAParkedCopyThatStoppedMovingIsAFailure distinguishes a copy the drainer is
// still working through from one it keeps failing on. The two need different
// words: a first failed attempt is routine, while a growing attempt count means
// the destination keeps refusing the write.
func TestAParkedCopyThatStoppedMovingIsAFailure(t *testing.T) {
	now := time.Now()
	moving := PendingEvent{ParkedAt: now.Add(-30 * time.Second), Attempts: 1}
	stuck := PendingEvent{ParkedAt: now.Add(-30 * time.Second), Attempts: pendingAttemptsStuck}

	c := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeForState(c.Preferences(), ThemeModeDark, true, pageBlockchain))
		if pendingEventTone(c, moving, now) == c.Theme().Danger {
			t.Error("a first attempt must not read as a failure")
		}
		if pendingEventTone(c, stuck, now) != c.Theme().Danger {
			t.Error("a copy past the attempt threshold must read as a failure")
		}
		// Age alone is enough: a copy parked past the retry window is stuck even
		// if the drainer never got a turn to increment its attempts.
		if pendingEventTone(c, PendingEvent{ParkedAt: now.Add(-time.Hour)}, now) != c.Theme().Danger {
			t.Error("a copy parked past the retry window must read as a failure")
		}
	}, 600, 400)
	c.Frame()
}

// TestNextTryIsAQuantity covers how the wait is spelled. A future clock time makes
// the reader subtract it against now; the question the row answers is whether the
// loop is still trying, and how soon.
func TestNextTryIsAQuantity(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	require.Equal(t, "in 30s", nextTryText(now.Add(30*time.Second), now))
	require.Equal(t, "due", nextTryText(now.Add(-time.Minute), now))
	require.Equal(t, "due", nextTryText(now, now))
}

// TestPendingEventTitleSpeaksTheWindowsLanguage: the module is the same word the
// blocks above use, and the verb comes out of the event type through the record
// vocabulary rather than being printed raw.
func TestPendingEventTitleSpeaksTheWindowsLanguage(t *testing.T) {
	event := PendingEvent{EventType: "token.mint", Module: "token"}
	require.Equal(t, "token · Mint", pendingEventTitle(event))
}
