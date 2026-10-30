package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEmptyReadoutCopyDistinguishesLoadedFromNotLoaded keeps a successful
// empty read from sounding like no read happened at all. The load key tells
// the card which sentence is true.
func TestEmptyReadoutCopyDistinguishesLoadedFromNotLoaded(t *testing.T) {
	t.Run("token history", func(t *testing.T) {
		app := New(workbenchFakes())
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Token")
		tester.Frame()
		require.True(t, tester.HasText("No transfers loaded yet."))

		app.token.tokenID = "t1"
		app.token.historyLoadedToken = "t1"
		app.token.historyLoadedLimit, app.token.historyLoadedOffset = 50, 0
		tester.Frame()
		require.True(t, tester.HasText("No transfers found for this query."))
		require.False(t, tester.HasText("No transfers loaded yet."))
	})

	t.Run("NFT history", func(t *testing.T) {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("NFT")
		app.nft.opTab = 2
		tester.Frame()
		require.True(t, tester.HasText("Choose an asset in Collection or enter its ID first."),
			"an empty id is a missing first step, not an empty read")

		app.nft.historyID = "n1"
		tester.Frame()
		require.True(t, tester.HasText("No history loaded yet."),
			"an id typed but not loaded is not an empty history")

		app.nft.historyLoadedID = "n1"
		tester.Frame()
		require.True(t, tester.HasText("No operations recorded for this NFT."))
		require.False(t, tester.HasText("No history loaded yet."))
	})

	t.Run("voting tally", func(t *testing.T) {
		app := New(&fakeServices{sessions: []SessionRow{{ID: "s0", Title: "Budget", Status: "active"}}})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Voting")
		app.voting.tab = 2
		tester.Frame()
		require.True(t, tester.HasText("Choose a session in Setup or enter its ID first."),
			"an empty id is a missing first step, not an empty read")

		// Naming a session loads it, and an empty tally then says so.
		app.voting.resultsSession = "s1"
		tester.Frame()
		require.True(t, tester.HasText("No votes recorded for this session."))
		require.False(t, tester.HasText("Choose a session in Setup or enter its ID first."))
	})
}

// TestEmptyHealthNamesTheMissingFeed pins the Health stage's empty panel: an
// empty store is a missing feed, not a feed that has not been polled yet.
func TestEmptyHealthNamesTheMissingFeed(t *testing.T) {
	showHealth := func(t *testing.T, fake *fakeServices) *ui.Tester {
		t.Helper()
		app := New(fake)
		tester := ui.NewTester(app.View, 1280, 2000)
		app.ShowPage("Oracle")
		app.oracle.tab = 1
		tester.Frame()
		return tester
	}

	t.Run("no feeds", func(t *testing.T) {
		tester := showHealth(t, &fakeServices{})
		require.True(t, tester.HasText("No feeds yet. Add a source to start polling."))
	})

	t.Run("feeds without attempts", func(t *testing.T) {
		tester := showHealth(t, &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}})
		require.True(t, tester.HasText("No poll attempts recorded yet."))
		require.False(t, tester.HasText("No feeds yet. Add a source to start polling."))
	})
}
