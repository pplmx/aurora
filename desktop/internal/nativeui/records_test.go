package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The ledger renders records, not payload cells. These tests pin the three
// things that make the page readable: an entry reads as a sentence, its exact
// payload is one click away, and the dashboard and the ledger page tell the
// same story about the same chain.

// ledgerRecords is the fixture the record tests share: one entry per shape the
// chain holds, already decoded the way the adapter delivers them.
func ledgerRecords() []BlockRow {
	const now = int64(1793548800)
	return []BlockRow{
		{
			Height: 4, Timestamp: now, Module: "lottery", Action: "Draw",
			Subject: "Alice, Carol", Summary: `{"participants":["Alice","Bob"],"winners":["Alice","Carol"]}`,
			Payload: `{"participants":["Alice","Bob"],"winners":["Alice","Carol"]}`,
			Facts: []BlockFact{
				{Label: "Winners", Value: "Alice, Carol"},
				{Label: "Participants", Value: "2"},
			},
		},
		{
			Height: 3, Timestamp: now - 60, Module: "token", Action: "Mint",
			Subject: "GLD", Summary: "token mint GLD owner",
			Payload: "mint|GLD|owner",
			Facts: []BlockFact{
				{Label: "Token", Value: "GLD"},
				{Label: "To", Value: "owner"},
			},
		},
	}
}

// TestLedgerReadsAsRecords pins the record sentence: the module, the action and
// the object it names are separate rendered texts, and the facts are labelled
// values beneath them.
func TestLedgerReadsAsRecords(t *testing.T) {
	app := New(&fakeServices{blocks: ledgerRecords()})
	tester := ui.NewTester(app.View, 1280, 1600)
	app.ShowPage("Blockchain")
	tester.Frame()

	for _, want := range []string{
		"Draw", "Alice, Carol",
		"Mint", "GLD",
		"Winners", "Participants", "Token", "To",
	} {
		require.Truef(t, tester.HasText(want), "the ledger must render %q", want)
	}
	// The record's height and its own time are part of the record, not a
	// separate column a reader has to line up by eye.
	require.True(t, tester.HasText("#4"), "the block height must sit on its record")
	require.True(t, tester.HasText(recordTimeLabel(1793548800)),
		"the block's local time must sit on its record")
}

// TestLedgerRecordOpensItsExactPayload keeps the decoded sentence from becoming
// the only copy of what the chain holds: the stored bytes are one click away.
func TestLedgerRecordOpensItsExactPayload(t *testing.T) {
	records := ledgerRecords()
	app := New(&fakeServices{blocks: records})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText("Payload"), "each record offers its exact payload")
	require.False(t, tester.HasText(records[0].Payload),
		"the payload is closed until it is asked for")

	require.NoError(t, tester.Click("Payload"))
	tester.Frame()
	require.True(t, tester.HasText(records[0].Payload),
		"opening the record must show the stored bytes unmodified")

	// Clicking by label works even for an element laid out off-screen, so the
	// affordance is also checked to sit inside the window.
	copyRect, ok := tester.Find("Copy payload")
	require.True(t, ok, "the copy action must be findable in the tree")
	assert.Positive(t, copyRect.W, "the copy action must have a visible width")
	assert.GreaterOrEqual(t, copyRect.X, float32(0))
	assert.LessOrEqual(t, copyRect.X+copyRect.W, float32(1280), "the copy action must stay inside the window")

	// The audit page stacks below 1240px; the action must stay reachable there
	// too, not just in the wide layout.
	tester.SetSize(900, 2000)
	tester.Frame()
	narrow, ok := tester.Find("Copy payload")
	require.True(t, ok, "the copy action must survive the stacked layout")
	assert.Positive(t, narrow.W)
	assert.LessOrEqual(t, narrow.X+narrow.W, float32(900), "the copy action must stay inside a narrow window")
	tester.SetSize(1280, 2000)
	tester.Frame()

	require.NoError(t, tester.Click("Copy payload"))
	tester.Frame()
	require.Equal(t, records[0].Payload, tester.Clipboard(),
		"the copy action must carry the exact bytes, not the shortened display")
	require.True(t, tester.HasText("Payload is on the clipboard"))

	require.NoError(t, tester.Click("Hide payload"))
	tester.Frame()
	require.False(t, tester.HasText(records[0].Payload),
		"closing the record must put the bytes away again")
	require.Equal(t, -1, app.blockchain.expanded, "closing must clear the open record")
}

// TestDashboardShowsTheSameRecordsAsTheLedger keeps the overview and the audit
// page from telling two different stories: the activity card renders the same
// decoded record.
func TestDashboardShowsTheSameRecordsAsTheLedger(t *testing.T) {
	app := New(&fakeServices{blocks: ledgerRecords()})
	tester := ui.NewTester(app.View, 1280, 2400)
	tester.Frame()

	for _, want := range []string{"Draw", "Alice, Carol", "Mint", "Winners"} {
		require.Truef(t, tester.HasText(want), "the activity card must show %q", want)
	}
}

// TestDashboardRecordOpensItsModule keeps the record a route rather than a dead
// row: choosing it lands on the page that wrote the entry.
func TestDashboardRecordOpensItsModule(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: 1, Module: "voting", Action: "Ballot", Subject: "cand-1",
		Facts: []BlockFact{{Label: "Candidate", Value: "cand-1"}},
	}}})
	tester := ui.NewTester(app.View, 1280, 2400)
	tester.Frame()

	require.NoError(t, tester.Click("cand-1"))
	tester.Frame()
	require.Equal(t, pageVoting, app.page)
}

// TestUnknownPayloadStillReads keeps an entry the decoder does not understand
// from rendering as an empty card. The stored text is what such an entry has,
// so it is what the record shows.
func TestUnknownPayloadStillReads(t *testing.T) {
	const payload = "some future module wrote this"
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: 9, Module: "", Summary: payload, Payload: payload,
	}}})
	tester := ui.NewTester(app.View, 1280, 1600)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText(payload),
		"an unrecognised payload must still read as its stored text")
	require.True(t, tester.HasText("#9"), "the block number must still identify the entry")
}
