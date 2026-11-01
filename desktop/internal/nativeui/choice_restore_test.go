package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRestoredChoicesSurviveAFilter records what the three restore functions
// are for: a reload must not leave the highlight on a row that now means
// something else. The selection is stored as the row's id, not its index, so a
// filter that reorders or shortens the list still lands on the same record.
func TestRestoredChoicesSurviveAFilter(t *testing.T) {
	app := New(&fakeServices{})
	app.oracle.sources = []SourceRow{
		{ID: "src-1", Name: "btc-usd", URL: "https://example.test/btc"},
		{ID: "src-2", Name: "eth-usd", URL: "https://example.test/eth"},
	}
	app.voting.voters = []VoterRow{
		{PublicKey: "pk-1", Name: "Ada"},
		{PublicKey: "pk-2", Name: "Grace"},
	}
	app.voting.candidates = []CandidateRow{{ID: "cand-1", Name: "Ada"}, {ID: "cand-2", Name: "Grace"}}
	app.voting.sessions = []SessionRow{{ID: "s-1", Title: "Budget"}, {ID: "s-2", Title: "Charter"}}

	// With no filter, the id lands on its own index.
	app.restoreSourceChoice("src-2")
	require.Equal(t, 1, app.oracle.selected)
	app.restoreVoterChoice("pk-2")
	require.Equal(t, 1, app.voting.voterChoice)
	app.restoreCandidateChoice("cand-2")
	require.Equal(t, 1, app.voting.candChoice)
	app.restoreSessionChoice("s-2")
	require.Equal(t, 1, app.voting.sessionChoice)

	// Narrow the lists so the stored records sit at index 0. Restoring the old
	// index would have pointed at a different record, which is the defect this
	// design exists to prevent.
	app.oracle.search = "eth"
	app.voting.voterSearch = "Grace"
	app.voting.candSearch = "Grace"
	app.voting.sessionSearch = "Charter"

	app.restoreSourceChoice("src-2")
	require.Equal(t, 0, app.oracle.selected, "the source is found at its new position")
	app.restoreVoterChoice("pk-2")
	require.Equal(t, 0, app.voting.voterChoice)
	app.restoreCandidateChoice("cand-2")
	require.Equal(t, 0, app.voting.candChoice)
	app.restoreSessionChoice("s-2")
	require.Equal(t, 0, app.voting.sessionChoice, "the session is found at its new position")
}

// TestRestoredChoiceOfAMissingRecordIsNoSelection is the half that protects a
// write: a record that disappeared from the page (deleted, or filtered out)
// must leave nothing highlighted, so the next action cannot be applied to
// whichever row happens to occupy the old index.
func TestRestoredChoiceOfAMissingRecordIsNoSelection(t *testing.T) {
	app := New(&fakeServices{})
	app.oracle.sources = []SourceRow{{ID: "src-1", Name: "btc-usd"}}
	app.voting.voters = []VoterRow{{PublicKey: "pk-1", Name: "Ada"}}
	app.voting.candidates = []CandidateRow{{ID: "cand-1", Name: "Ada"}}
	app.voting.sessions = []SessionRow{{ID: "s-1", Title: "Budget"}}

	app.oracle.selected = 3
	app.voting.voterChoice = 3
	app.voting.candChoice = 3
	app.voting.sessionChoice = 3

	app.restoreSourceChoice("src-gone")
	app.restoreVoterChoice("pk-gone")
	app.restoreCandidateChoice("cand-gone")
	app.restoreSessionChoice("s-gone")

	require.Equal(t, -1, app.oracle.selected, "no source is highlighted")
	require.Equal(t, -1, app.voting.voterChoice, "no voter is highlighted")
	require.Equal(t, -1, app.voting.candChoice, "no candidate is highlighted")
	require.Equal(t, -1, app.voting.sessionChoice, "no session is highlighted")
}

// TestRestoringAnEmptyChoiceClearsTheSelection covers the reload path that has
// nothing to restore: it must still clear, because the list may have shrunk
// under the old index.
func TestRestoringAnEmptyChoiceClearsTheSelection(t *testing.T) {
	app := New(&fakeServices{})
	app.oracle.selected = 2
	app.voting.voterChoice = 2
	app.voting.candChoice = 2
	app.voting.sessionChoice = 2

	app.restoreSourceChoice("")
	app.restoreVoterChoice("")
	app.restoreCandidateChoice("")
	app.restoreSessionChoice("")

	require.Equal(t, -1, app.oracle.selected)
	require.Equal(t, -1, app.voting.voterChoice)
	require.Equal(t, -1, app.voting.candChoice)
	require.Equal(t, -1, app.voting.sessionChoice)
}

// TestSelectingAMissingRecordClearsTheHighlight covers the five helpers that
// highlight a record the page was just told to show — after a mint, a draw, a
// registration. They look the record up by id and used to return without
// touching the highlight when the lookup failed, leaving the previous row
// highlighted: the write cards read their subject from that highlight, so the
// next action could name a record the operator never chose.
func TestSelectingAMissingRecordClearsTheHighlight(t *testing.T) {
	app := New(&fakeServices{})
	app.oracle.sources = []SourceRow{{ID: "s1", Name: "btc"}}
	app.voting.candidates = []CandidateRow{{ID: "c1", Name: "Ada"}}
	app.voting.sessions = []SessionRow{{ID: "sess-1", Title: "Budget"}}
	app.voting.voters = []VoterRow{{PublicKey: "pk-1", Name: "Ada"}}
	app.lottery.rows = []DrawRow{{ID: "d1"}}

	app.oracle.selected = 0
	app.voting.candChoice = 0
	app.voting.sessionChoice = 0
	app.voting.voterChoice = 0
	app.lottery.selected = 0

	app.selectSourceByID("missing")
	app.selectCandidateByID("missing")
	app.selectSessionByID("missing")
	app.selectVoterByPublicKey("missing")
	app.selectLotteryByID("missing")

	require.Equal(t, -1, app.oracle.selected, "no source is highlighted")
	require.Equal(t, -1, app.voting.candChoice, "no candidate is highlighted")
	require.Equal(t, -1, app.voting.sessionChoice, "no session is highlighted")
	require.Equal(t, -1, app.voting.voterChoice, "no voter is highlighted")
	require.Equal(t, -1, app.lottery.selected, "no draw is highlighted")
}

// TestSelectingAKnownRecordHighlightsIt is the other half: the lookup that
// succeeds still lands on the record's own row.
func TestSelectingAKnownRecordHighlightsIt(t *testing.T) {
	app := New(&fakeServices{})
	app.oracle.sources = []SourceRow{{ID: "s1", Name: "btc"}, {ID: "s2", Name: "eth"}}
	app.voting.candidates = []CandidateRow{{ID: "c1", Name: "Ada"}, {ID: "c2", Name: "Grace"}}
	app.voting.sessions = []SessionRow{{ID: "sess-1", Title: "Budget"}, {ID: "sess-2", Title: "Charter"}}
	app.voting.voters = []VoterRow{{PublicKey: "pk-1", Name: "Ada"}, {PublicKey: "pk-2", Name: "Grace"}}
	app.lottery.rows = []DrawRow{{ID: "d1"}, {ID: "d2"}}

	app.selectSourceByID("s2")
	app.selectCandidateByID("c2")
	app.selectSessionByID("sess-2")
	app.selectVoterByPublicKey("pk-2")
	app.selectLotteryByID("d2")

	require.Equal(t, 1, app.oracle.selected)
	require.Equal(t, 1, app.voting.candChoice)
	require.Equal(t, 1, app.voting.sessionChoice)
	require.Equal(t, 1, app.voting.voterChoice)
	require.Equal(t, 1, app.lottery.selected)
}
