package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEmptyModulesOpenWithTheFirstRecordPage pins the first-write hierarchy:
// after a successful empty read the create form leads the page beside a
// numbered path, and the cards that act on a record stay away until one
// exists. A column of empty cards and disabled controls taught a first-time
// operator nothing.
func TestEmptyModulesOpenWithTheFirstRecordPage(t *testing.T) {
	cases := []struct {
		page       string
		title      string
		create     string
		collection string
	}{
		{"NFT", "No NFTs yet", "Mint NFT", "Collection"},
		{"Oracle", "No sources yet", "Add Source", "Data Sources"},
		{"Token", "No tokens yet", "Create Token", "Tokens"},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, 1280, 4000)
			app.ShowPage(tc.page)
			tester.Frame()

			require.True(t, tester.HasText("First record"))
			require.True(t, tester.HasText(tc.title), "the state must be named")
			require.True(t, tester.HasText(tc.create), "the one useful form must stay")
			require.False(t, tester.HasText(tc.collection),
				"a card that acts on a record must wait for the first one")
			require.False(t, tester.HasText("How this works"),
				"the first-record panel already is the page's guide")
		})
	}
}

// TestFirstRecordPageGivesWayToTheWorkbenchAfterCreate proves the transition,
// not just the gate: the form on the first-record page lands on the full
// workbench for the record it just wrote.
func TestFirstRecordPageGivesWayToTheWorkbenchAfterCreate(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("First record"))

	app.token.name, app.token.symbol = "Gold", "GLD"
	app.token.supply, app.token.owner = "1000", "owner-pk"
	tester.Frame()
	// The card title and its submit button share the label, so drive the action
	// the button invokes rather than guessing which node a text click hits.
	app.createToken()
	tester.Frame()

	require.False(t, tester.HasText("First record"))
	require.True(t, tester.HasText("Mint"), "the write tabs follow the first token")
	require.True(t, tester.HasText("Acting on"))
}

// TestFirstRecordPageYieldsToWorkInProgress keeps the shortcut from hiding the
// full workbench: a typed subject, an applied filter or a failed read all mean
// the operator is not at the first-write state.
func TestFirstRecordPageYieldsToWorkInProgress(t *testing.T) {
	t.Run("entered subject", func(t *testing.T) {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, 1280, 4000)
		app.ShowPage("Token")
		tester.Frame()
		require.True(t, tester.HasText("No tokens yet"))

		app.token.tokenID = "t1"
		tester.Frame()
		require.False(t, tester.HasText("Fill in the create form."))
		require.True(t, tester.HasText("Tokens"))
	})

	t.Run("filter", func(t *testing.T) {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, 1280, 4000)
		app.ShowPage("NFT")
		tester.Frame()

		app.nft.filterOwner = "owner-a"
		tester.Frame()
		require.False(t, tester.HasText("First record"))
		require.True(t, tester.HasText("Collection"))
	})

	t.Run("load failure", func(t *testing.T) {
		fake := &fakeServices{loadErrs: fakeLoadErrors{nfts: errors.New("database is locked")}}
		app := New(fake)
		tester := ui.NewTester(app.View, 1280, 4000)
		app.ShowPage("NFT")
		tester.Frame()
		require.False(t, tester.HasText("First record"),
			"a failed read must not claim the collection is empty")
		require.True(t, tester.HasText("Collection"))
	})
}

// TestFirstRecordGateIgnoresOnlyUntouchedPages checks the shared gate directly
// for the inputs the rendered tests do not reach: every collection narrows by a
// search, and typing a query is work inside a collection, not a first write.
func TestFirstRecordGateIgnoresOnlyUntouchedPages(t *testing.T) {
	app := New(&fakeServices{})
	require.True(t, app.nftNeedsFirstRecord())
	require.True(t, app.oracleNeedsFirstRecord())
	require.True(t, app.tokenNeedsFirstRecord())

	app.nft.search = "star"
	require.False(t, app.nftNeedsFirstRecord())
	app.oracle.appliedSearch = "btc"
	require.False(t, app.oracleNeedsFirstRecord())
	app.token.appliedSearch = "gold"
	require.False(t, app.tokenNeedsFirstRecord())

	app.nft.list = []NFTRow{{ID: "n1"}}
	require.False(t, app.nftNeedsFirstRecord())
	app.oracle.sources = []SourceRow{{ID: "s1"}}
	require.False(t, app.oracleNeedsFirstRecord())
	app.token.tokens = []TokenRow{{ID: "t1"}}
	require.False(t, app.tokenNeedsFirstRecord())
}

// TestVotingSetupLeadsWithRegistration applies the first-write hierarchy to
// the Voting Setup tab. Three empty lists and their big empty states pushed
// the session form below the fold, while the two forms that can actually work
// on an untouched store were the only way forward.
func TestVotingSetupLeadsWithRegistration(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.True(t, tester.HasText("First record"))
	require.True(t, tester.HasText("No voters yet"))
	require.True(t, tester.HasText("Register Voter"))
	require.True(t, tester.HasText("Register Candidate"))
	require.False(t, tester.HasText("Create Session"), "a session needs a roster first")
	require.False(t, tester.HasText("How this works"),
		"the first-record panel already is the page's guide")

	// The Ballot stage keeps its own layout and stays reachable.
	app.voting.tab = 1
	tester.Frame()
	require.False(t, tester.HasText("No voters yet"))

	// The first record restores the workbench: lists, session form and guide.
	app.voting.tab = 0
	app.voting.voters = []VoterRow{{PublicKey: "voter-pk-1", Name: "Ada"}}
	tester.Frame()
	require.False(t, tester.HasText("First record"))
	require.True(t, tester.HasText("Create Session"))
	require.True(t, tester.HasText("How this works"))
}
