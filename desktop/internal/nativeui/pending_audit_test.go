package nativeui

import (
	"fmt"
	"strings"
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

// TestTheWaitingListIsReadWithThePageNotPerFrame keeps a body of data out of the
// paint path. The card listed the outbox on every frame, which means a database
// read on each repaint — including the idle ones the window does for a hover or a
// caret blink. Every other list on this page is read when the page loads or is
// refreshed, and this one has no reason to differ.
func TestTheWaitingListIsReadWithThePageNotPerFrame(t *testing.T) {
	reads := 0
	app := New(&fakeServices{blocks: []BlockRow{{Height: 2, Module: "token", Summary: "x"}}})
	app.SetActions(Actions{
		BackUpDatabase: func() {},
		OutboxList: func() []PendingEvent {
			reads++
			return []PendingEvent{{
				EventType: "token.mint", Module: "token", Aggregate: "GLD",
				ParkedAt: time.Now(), Attempts: 1,
			}}
		},
	})
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Blockchain")
	tester.Frame()

	afterLoad := reads
	require.Equal(t, 1, afterLoad, "opening the page reads the outbox once")

	for range 10 {
		tester.Frame()
	}
	require.Equal(t, afterLoad, reads, "repainting must not read storage")
	require.True(t, tester.HasText("Waiting to be delivered"),
		"the list is still on screen, from the rows read with the page")

	app.loadBlockchainAt("", 200, 0)
	require.Equal(t, afterLoad+1, reads, "a refresh reads it again")
}

// TestACappedWaitingListSaysHowMuchItIsHiding covers the list's own limit. The
// card draws a fixed number of rows while the footer reports the true backlog, so
// an operator counting twenty rows against a footer that says sixty would take the
// difference for a display bug rather than for the rest of the queue.
func TestACappedWaitingListSaysHowMuchItIsHiding(t *testing.T) {
	const total = PendingAuditLimit + 40
	events := make([]PendingEvent, PendingAuditLimit)
	for i := range events {
		events[i] = PendingEvent{
			EventType: "token.mint", Module: "token",
			Aggregate: fmt.Sprintf("agg-%02d", i), ParkedAt: time.Now(), Attempts: 1,
		}
	}
	app := New(&fakeServices{blocks: []BlockRow{{Height: 2, Module: "token", Summary: "x"}}})
	actions := pendingFixture(events...)
	actions.OutboxPending = func() (int, bool) { return total, true }
	actions.OutboxOldest = func() (time.Duration, bool) { return time.Minute, true }
	app.SetActions(actions)

	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.Truef(t,
		tester.HasText(fmt.Sprintf("Showing the oldest %d of %d parked copies", PendingAuditLimit, total)),
		"a capped list must say how many are left out")
}

// TestAWaitingListThatFitsKeepsThePlainDescription: when every parked copy is on
// screen there is nothing withheld, and a "showing the oldest 3 of 3" line would
// only make the reader check a number that always matches.
func TestAWaitingListThatFitsKeepsThePlainDescription(t *testing.T) {
	const total = 3
	events := make([]PendingEvent, total)
	for i := range events {
		events[i] = PendingEvent{
			EventType: "token.mint", Module: "token",
			Aggregate: fmt.Sprintf("agg-%02d", i), ParkedAt: time.Now(), Attempts: 1,
		}
	}
	app := New(&fakeServices{blocks: []BlockRow{{Height: 2, Module: "token", Summary: "x"}}})
	actions := pendingFixture(events...)
	actions.OutboxPending = func() (int, bool) { return total, true }
	app.SetActions(actions)

	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.False(t, tester.HasText("Showing the oldest"),
		"a list showing every parked copy needs no ratio")
	require.True(t, tester.HasText("Audit copies of blocks that are already on the chain"))
}

// TestAParkedCopyShowsItsPayload closes the gap the backlog card had: it named
// the operation and the retry state but never said which record was parked, so
// two mints of the same token read identically. The payload opens on request for
// the same reason the ledger's records do — JSON in every row would bury the
// retry state the rows are scanned for.
func TestAParkedCopyShowsItsPayload(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	app.SetActions(Actions{
		OutboxPending: func() (int, bool) { return 2, true },
		OutboxList: func() []PendingEvent {
			return []PendingEvent{
				{EventType: "token.Minted", Module: "token", Aggregate: "Mint GLD",
					Payload: `{"token_id":"GLD","amount":"100"}`},
				{EventType: "nft.Minted", Module: "nft", Aggregate: "Mint Star",
					Payload: `{"nft_id":"n-1"}`},
			}
		},
	})
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText("Mint GLD"), "the operation is listed")
	require.False(t, tester.HasText(`{"token_id"`),
		"the payload stays closed until asked for, so the retry facts stay readable")

	require.NoError(t, tester.Click("Payload"))
	tester.Frame()
	require.Equal(t, 0, app.blockchain.pendingExpanded, "the first row is the one opened")
	require.True(t, tester.HasText(`{"token_id":"GLD","amount":"100"}`),
		"the exact recorded bytes are what says which record is parked")

	// Only one payload is open at a time: comparing two is a toggle, not a scroll.
	require.True(t, tester.HasText("Hide payload"))
}

// TestClosingTheOpenPayloadRestoresTheRow covers the other half of the toggle,
// and the reset that keeps an index from outliving the list it points into.
func TestClosingTheOpenPayloadRestoresTheRow(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	list := []PendingEvent{
		{EventType: "token.Minted", Module: "token", Aggregate: "Mint GLD",
			Payload: `{"token_id":"GLD"}`},
	}
	app.SetActions(Actions{
		OutboxPending: func() (int, bool) { return 1, true },
		OutboxList:    func() []PendingEvent { return list },
	})
	app.ShowPage("Blockchain")
	tester.Frame()
	require.NoError(t, tester.Click("Payload"))
	tester.Frame()
	require.True(t, tester.HasText(`{"token_id":"GLD"}`))

	require.NoError(t, tester.Click("Hide payload"))
	tester.Frame()
	require.Equal(t, -1, app.blockchain.pendingExpanded)
	require.False(t, tester.HasText(`{"token_id":"GLD"}`))

	// The queue draining under an open row must close it rather than leave the
	// toggle naming a payload that is no longer there.
	require.NoError(t, tester.Click("Payload"))
	tester.Frame()
	list = nil
	app.refresh()
	tester.Frame()
	require.Equal(t, -1, app.blockchain.pendingExpanded,
		"an index past the end of the re-read list must be cleared")
}

// TestALargePayloadIsCappedAndSaysSo covers the size a real audit payload
// reaches. A lottery draw carries up to 10,000 participants, so the recorded
// bytes run to 200 KB; laying all of it out cost 131 ms per frame while the
// record was open, which is a visible stall on every repaint. The view lays out
// a prefix, says how much it left out, and Copy payload still takes everything.
func TestALargePayloadIsCappedAndSaysSo(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	big := `{"participants":["` + strings.Repeat("participant-name,", 6000) + `alice"]}`
	require.Greater(t, len(big), payloadPreviewLen, "the fixture must exceed the cap")

	app.SetActions(Actions{
		OutboxPending: func() (int, bool) { return 1, true },
		OutboxList: func() []PendingEvent {
			return []PendingEvent{{EventType: "lottery.draw", Module: "lottery",
				Aggregate: "Run draw", Payload: big}}
		},
	})
	app.ShowPage("Blockchain")
	tester.Frame()
	require.NoError(t, tester.Click("Payload"))
	tester.Frame()

	require.True(t, tester.HasText("Copy payload takes all of it"),
		"a truncated view has to say so, or the cut reads as the whole value")
	require.False(t, tester.HasText(big), "the whole 200 KB is not laid out")
}

// TestAShortPayloadIsShownWhole guards the other side: the disclosure sentence
// must stay silent when nothing was withheld.
func TestAShortPayloadIsShownWhole(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2600)
	short := `{"token_id":"GLD","amount":"100"}`
	app.SetActions(Actions{
		OutboxPending: func() (int, bool) { return 1, true },
		OutboxList: func() []PendingEvent {
			return []PendingEvent{{EventType: "token.Minted", Module: "token",
				Aggregate: "Mint GLD", Payload: short}}
		},
	})
	app.ShowPage("Blockchain")
	tester.Frame()
	require.NoError(t, tester.Click("Payload"))
	tester.Frame()

	require.True(t, tester.HasText(short), "a small payload is shown in full")
	require.False(t, tester.HasText("Copy payload takes all of it"),
		"and nothing claims bytes were withheld")
}

// TestALedgerRecordAlsoCapsItsPayload covers the other caller of the payload
// view. The backlog card and the ledger record share recordPayloadBody, so the
// cap applies to both — but "it is the same function" is the kind of reasoning
// that stops being true the day someone inlines one of the two call sites.
func TestALedgerRecordAlsoCapsItsPayload(t *testing.T) {
	big := `{"participants":["` + strings.Repeat("participant-name,", 6000) + `alice"]}`
	require.Greater(t, len(big), payloadPreviewLen, "the fixture must exceed the cap")

	fake := &fakeServices{blocks: []BlockRow{
		{Height: 1, Module: "lottery", Action: "draw", Summary: "Run draw", Payload: big},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()
	require.NoError(t, tester.Click("Payload"))
	tester.Frame()

	require.True(t, tester.HasText("Copy payload takes all of it"),
		"a ledger record with a large payload must disclose the cut too")
	require.False(t, tester.HasText(big), "the whole payload is not laid out")
}

// TestTheSidebarSaysHowFreshThePageIs covers the question a timer-driven window
// leaves open. With live updates on, the page refreshes itself, so a still page
// is ambiguous: a stalled timer, a write holding the database and a healthy
// quiet system all look the same. The stamp answers it, and stays silent when
// live updates are off, because then the page is not claiming to be fresh.
func TestTheSidebarSaysHowFreshThePageIs(t *testing.T) {
	app := New(workbenchFakes())
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app.liveNow = func() time.Time { return now }
	app.SetActions(Actions{Version: "0.0.1"})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Dashboard")
	tester.Frame()

	require.False(t, tester.HasText("Updated"),
		"with live updates off the page makes no freshness claim")

	app.liveUpdates = true
	tester.Frame()
	require.False(t, tester.HasText("Updated"),
		"and it stays silent until a read has actually completed")

	app.liveReadAt = now.Add(-3 * time.Second)
	tester.Frame()
	require.True(t, tester.HasText("Updated 3s ago"),
		"a recent read is reported in seconds")

	// A stamp that keeps counting is the point: past a minute the interval has
	// clearly stopped, and saying so is what a reader needs.
	app.liveReadAt = now.Add(-90 * time.Second)
	tester.Frame()
	require.True(t, tester.HasText("Updated 1m ago"))

	app.liveUpdates = false
	tester.Frame()
	require.False(t, tester.HasText("Updated"))
}

// TestTheFreshnessStampFollowsARealTick guards the bug the first version had:
// the stamp was written at the top of the frame routine, which runs on every
// repaint, so the sidebar claimed "0s ago" forever — including while the timer
// was stalled, which is exactly the state the stamp exists to reveal.
func TestTheFreshnessStampFollowsARealTick(t *testing.T) {
	app := New(workbenchFakes())
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app.liveNow = func() time.Time { return now }
	app.liveUpdates = true
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Dashboard")

	// A repaint that is not a tick must not stamp anything.
	tester.Frame()
	require.True(t, app.liveReadAt.IsZero(), "a plain repaint is not a read")

	// A due tick reads. The page reload runs, and only then is the time recorded.
	app.liveNext = now.Add(-time.Second)
	app.refresh()
	tester.Frame()
	require.False(t, app.liveReadAt.IsZero(), "a due tick is what stamps the read")
	require.WithinDuration(t, now, app.liveReadAt, time.Second)
}
