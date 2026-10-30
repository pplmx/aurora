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
		require.True(t, tester.HasText("No history loaded yet."))

		app.nft.historyID = "n1"
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
		require.True(t, tester.HasText("No tally loaded yet."))

		app.voting.resultsSession = "s1"
		app.voting.resultsLoadedID = "s1"
		tester.Frame()
		require.True(t, tester.HasText("No votes recorded for this session."))
		require.False(t, tester.HasText("No tally loaded yet."))
	})
}
