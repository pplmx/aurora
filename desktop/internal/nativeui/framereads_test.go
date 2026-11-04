package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// outboxCounter counts the audit-backlog reads, which reach storage through the
// shell's Action closures rather than through the services the fake replaces. A
// per-frame audit that only watched the services would have called the backlog
// clean.
type outboxCounter struct {
	pending, oldest, list int
	actions               Actions
}

func newOutboxCounter() *outboxCounter {
	c := &outboxCounter{}
	c.actions = Actions{
		BackUpDatabase: func() {},
		OutboxPending:  func() (int, bool) { c.pending++; return 1, true },
		OutboxOldest:   func() (time.Duration, bool) { c.oldest++; return time.Minute, true },
		OutboxList: func() []PendingEvent {
			c.list++
			return []PendingEvent{{
				EventType: "token.mint", Module: "token", Aggregate: "GLD",
				ParkedAt: time.Now(), Attempts: 1,
			}}
		},
	}
	return c
}

func (c *outboxCounter) reads() int { return c.pending + c.oldest + c.list }

func (c *outboxCounter) assertUnread(t *testing.T, before, frames int) {
	t.Helper()
	require.Equalf(t, before, c.reads(),
		"the audit backlog was read %d times over %d idle frames", c.reads()-before, frames)
}

// storageReadCounters is every read the fake services count. The list is the
// contract this test holds: a frame must not reach storage on its own.
func storageReadCounters(f *fakeServices) map[string]int {
	return map[string]int{
		"history":        f.historyCalls,
		"listTokens":     f.listTokensCalls,
		"voterList":      f.voterListCalls,
		"candidateList":  f.candidateListCalls,
		"candidateName":  f.candidateNameCalls,
		"sessionList":    f.sessionListCalls,
		"sourceList":     f.sourceListCalls,
		"oracleSource":   f.oracleSourceCalls,
		"dashboard":      f.dashboardCalls,
		"blockList":      f.blockListCalls,
		"oracleHealth":   f.oracleHealthCalls,
		"tokenInfo":      f.tokenInfoCalls,
		"tokenBalance":   f.tokenBalanceCalls,
		"tokenHistory":   f.tokenHistoryCalls,
		"tokenAllowance": f.tokenAllowanceCalls,
	}
}

// TestNoPageReadsStorageWhileRepainting holds the rule a frame must obey: it
// draws what the page already read. Data arrives when the page opens, when the
// operator refreshes, and on the live-refresh tick — never because the window
// repainted, which it does for a hover or a caret blink as readily as for new
// data.
//
// The audit backlog broke this twice, in the sidebar footer and in the ledger's
// waiting list, and both were invisible to the tests that only checked what the
// screen said. Counting reads per frame is what makes the class of defect
// visible, so every page is measured here rather than only the two that broke.
func TestNoPageReadsStorageWhileRepainting(t *testing.T) {
	for _, p := range pageOrder {
		t.Run(pageTitles[p], func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			outbox := newOutboxCounter()
			app.SetActions(outbox.actions)
			tester := ui.NewTester(app.View, 1280, 2200)
			app.ShowPage(pageTitles[p])
			tester.Frame()

			before := storageReadCounters(fake)
			beforeOutbox := outbox.reads()
			const frames = 10
			for range frames {
				tester.Frame()
			}
			after := storageReadCounters(fake)
			outbox.assertUnread(t, beforeOutbox, frames)

			for name, count := range after {
				require.Equalf(t, before[name], count,
					"%s read %d times over %d idle frames; a frame draws what the page already read",
					name, count-before[name], frames)
			}
		})
	}
}

// TestNoPageReadsStorageAfterASelection covers the paths that fetch on demand.
// Selecting a token or opening a record reads once, on the click — repeating that
// read on every later frame would make an open detail card cost a query per
// repaint.
func TestNoPageReadsStorageAfterASelection(t *testing.T) {
	for _, p := range pageOrder {
		t.Run(pageTitles[p], func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			outbox := newOutboxCounter()
			app.SetActions(outbox.actions)
			tester := ui.NewTester(app.View, 1280, 2200)
			app.ShowPage(pageTitles[p])
			tester.Frame()
			for _, label := range []string{"Payload", "Inspect"} {
				if err := tester.Click(label); err == nil {
					tester.Frame()
				}
			}

			before := storageReadCounters(fake)
			beforeOutbox := outbox.reads()
			for range 10 {
				tester.Frame()
			}
			after := storageReadCounters(fake)
			outbox.assertUnread(t, beforeOutbox, 10)

			for name, count := range after {
				require.Equalf(t, before[name], count,
					"%s read %d times after its selection settled", name, count-before[name])
			}
		})
	}
}

// TestTheCounterListCoversTheFake keeps the audit honest: a counter added to the
// fake but left out of the map would make every page look clean.
func TestTheCounterListCoversTheFake(t *testing.T) {
	fake := workbenchFakes()
	fake.dashboardCalls = 101
	fake.tokenInfoCalls = 102
	counters := storageReadCounters(fake)
	require.Equal(t, 101, counters["dashboard"])
	require.Equal(t, 102, counters["tokenInfo"])
	// A read that names the page's data must be in the map. The list below is the
	// set of methods a page calls to fill itself; a new one belongs here too.
	for _, name := range []string{
		"history", "listTokens", "voterList", "candidateList", "sessionList",
		"sourceList", "dashboard", "blockList", "oracleHealth", "tokenInfo",
		"tokenBalance", "tokenHistory", "tokenAllowance",
	} {
		_, ok := counters[name]
		require.Truef(t, ok, "%s must be in the audit's counter list", name)
	}
}
