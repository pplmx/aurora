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
		// Three states, as the NFT history and the voting tally have: an empty
		// field asks for a subject, a chosen-but-unread token says it has not
		// been loaded, and a read that returned nothing says so. The card used
		// to print the middle answer for the first state too, telling the reader
		// to press a button that is disabled until an ID is filled in.
		require.True(t, tester.HasText("Choose a token, or enter its ID first."),
			"an empty token field is a missing first step, not an empty read")
		require.False(t, tester.HasText("No transfers loaded yet."))

		app.token.tokenID = "t1"
		tester.Frame()
		require.True(t, tester.HasText("No transfers loaded yet."),
			"a token named but not read is not an empty history")
		require.False(t, tester.HasText("Choose a token, or enter its ID first."))

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

	t.Run("oracle readings", func(t *testing.T) {
		app := New(&fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Oracle")
		app.oracle.tab = 2
		app.oracle.queryID = "s1"
		tester.Frame()
		// A read that came back empty. The card used to draw no rows at all,
		// which reads as "this card has no content" rather than "this source has
		// no readings yet".
		app.oracle.dataLoadedSource = "s1"
		app.oracle.dataLoadedLimit = app.oracleQueryLimit()
		tester.Frame()
		require.True(t, tester.HasText("No readings recorded for this source yet."))

		app.oracle.data = []DataRow{{Value: "42", BlockHeight: 7}}
		tester.Frame()
		require.True(t, tester.HasText("42"))
		require.False(t, tester.HasText("No readings recorded for this source yet."))
	})

	t.Run("oracle latest", func(t *testing.T) {
		app := New(&fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Oracle")
		app.oracle.tab = 2
		app.oracle.latestID = "s1"
		// Opening the tab reads the selected source, so the card is answered by
		// the frame below; the "nothing read yet" state is the one before it.
		require.False(t, tester.HasText("No reading recorded for this source yet."))

		app.oracle.latestLoadedSource = "s1"
		tester.Frame()
		require.True(t, tester.HasText("No reading recorded for this source yet."),
			"an answered read with no value says so")

		app.oracle.latest = &DataRow{Value: "42", BlockHeight: 7}
		tester.Frame()
		require.True(t, tester.HasText("42"))
		require.False(t, tester.HasText("No reading recorded for this source yet."))
	})

	t.Run("voting tally", func(t *testing.T) {
		app := New(&fakeServices{sessions: []SessionRow{{ID: "s0", Title: "Budget", Status: "active"}}})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Voting")
		app.voting.tab = 2
		tester.Frame()
		const prompt = "Choose a stored session in Results, or paste its ID first."
		require.True(t, tester.HasText(prompt),
			"an empty id is a missing first step, not an empty read")

		// Naming a session loads it, and an empty tally then says so.
		app.voting.resultsSession = "s1"
		tester.Frame()
		require.True(t, tester.HasText("No votes recorded for this session."))
		require.False(t, tester.HasText(prompt))
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

// TestValueCardsDrawNoValueRowWithoutAValue covers the other half of the empty-read
// contract. The list cards say "this subject has no records yet"; the value cards
// must not, because they read something that exists by definition — a token's
// supply, an account's balance, an asset's holder — so a blank means "not read
// yet", and a row whose label is followed by an empty value claims the value is
// empty.
//
// readout draws each row as two text nodes, the label and the value, so the check
// is on pairs rather than on a joined string.
func TestValueCardsDrawNoValueRowWithoutAValue(t *testing.T) {
	emptyValueRows := func(tester *ui.Tester) []string {
		texts := tester.Texts()
		var found []string
		for i := 0; i+1 < len(texts); i++ {
			if texts[i+1] == "" {
				found = append(found, texts[i])
			}
		}
		return found
	}

	t.Run("token", func(t *testing.T) {
		app := New(workbenchFakes())
		tester := ui.NewTester(app.View, 1400, 6000)
		app.ShowPage("Token")
		tester.Frame()
		app.token.tokenID = "t1"
		tester.Frame()
		// Inputs filled and the read answered, but no value came back.
		app.token.balanceAccount = "account-a"
		app.token.balanceLoadedToken, app.token.balanceLoadedOwner = "t1", "account-a"
		app.token.allowanceOwner, app.token.allowanceTo = "account-a", "spender-a"
		app.token.allowanceLoadedToken = "t1"
		app.token.allowanceLoadedOwner = "account-a"
		app.token.allowanceLoadedTo = "spender-a"
		tester.Frame()

		rows := emptyValueRows(tester)
		require.NotContains(t, rows, "Balance",
			"a balance that was not read must not draw an empty row")
		require.NotContains(t, rows, "Allowance",
			"an allowance that was not read must not draw an empty row")
	})

	t.Run("nft", func(t *testing.T) {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, 1400, 6000)
		app.ShowPage("NFT")
		tester.Frame()
		app.nft.id = "n1"
		app.nft.nftLoadedID = "n1"
		tester.Frame()

		rows := emptyValueRows(tester)
		for _, label := range []string{"Holder", "Creator", "Block"} {
			require.NotContains(t, rows, label,
				"an asset that was not read must not draw empty rows")
		}
	})
}
