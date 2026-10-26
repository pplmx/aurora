package nativeui

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTokenSearchNarrowsTheList pins the collection's search field: every
// stored token is listed, and the field narrows by id, name or symbol without
// loading again. Choosing a row after filtering must still be that row.
func TestTokenSearchNarrowsTheList(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "Gold", Symbol: "AU", TotalSupply: "10"},
		{ID: "t2", Name: "Silver", Symbol: "AG", TotalSupply: "20"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("Gold"))
	require.True(t, tester.HasText("Silver"))

	app.token.search = "silver"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a name match must stay listed")
	require.False(t, tester.HasText("Gold"), "a non-match must leave the list")

	app.token.search = "AG"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a symbol match must stay listed")

	app.token.search = "nothing-like-this"
	tester.Frame()
	require.True(t, tester.HasText("No tokens match “nothing-like-this”."))

	// Choosing a row from the filtered list selects that token.
	app.token.search = "Gold"
	tester.Frame()
	require.NoError(t, tester.Click("Gold"))
	tester.Frame()
	require.Equal(t, "t1", app.token.tokenID)

	app.token.search = ""
	tester.Frame()
	require.True(t, tester.HasText("Gold"))
	require.True(t, tester.HasText("Silver"))
}

// TestNFTSearchNarrowsTheCollection covers the same field on the collection,
// which already has an owner filter: search applies on top of it.
func TestNFTSearchNarrowsTheCollection(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First Art", Owner: "owner-a"},
		{ID: "n2", Name: "Second Art", Owner: "owner-a"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()
	require.True(t, tester.HasText("First Art"))
	require.True(t, tester.HasText("Second Art"))

	app.nft.search = "second"
	tester.Frame()
	require.True(t, tester.HasText("Second Art"))
	require.False(t, tester.HasText("First Art"))

	app.nft.search = "none"
	tester.Frame()
	require.True(t, tester.HasText("No NFTs match “none”."))
}

// TestVotingListsSearchAndSelectFromTheFilteredRows covers the voting page's
// two lists. Selection indexes the filtered slice, so a row chosen after a
// search must be the row the operator sees - not the same index in the full
// list.
func TestVotingListsSearchAndSelectFromTheFilteredRows(t *testing.T) {
	fake := &fakeServices{
		candidates: []CandidateRow{
			{ID: "c1", Name: "Alice", Party: "Greens"},
			{ID: "c2", Name: "Bob", Party: "Blues"},
		},
		sessions: []SessionRow{
			{ID: "s1", Title: "Budget", Status: "active"},
			{ID: "s2", Title: "Standup", Status: "draft"},
		},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()
	require.True(t, tester.HasText("Alice"))
	require.True(t, tester.HasText("Budget"))

	app.voting.candSearch = "blues"
	tester.Frame()
	require.True(t, tester.HasText("Bob"))
	require.False(t, tester.HasText("Alice"), "the candidate search must narrow the table")

	app.voting.sessionSearch = "standup"
	tester.Frame()
	require.True(t, tester.HasText("Standup"))
	require.False(t, tester.HasText("Budget"), "the session search must narrow the table")

	// Selecting the filtered rows fills the forms with those rows' ids.
	require.NoError(t, tester.Click("s2"))
	tester.Frame()
	require.Equal(t, "s2", app.voting.controlID)

	require.NoError(t, tester.Click("c2"))
	tester.Frame()
	require.NoError(t, tester.Click("Add to session roster"))
	tester.Frame()
	require.Equal(t, "c2", app.voting.sessionIDs,
		"the filtered row must map to its own candidate, not the full list's index")
}

// TestLotteryHistorySearchAndSelection covers the draws list: id, seed,
// participant and winner are searchable, and the chosen filtered row is the
// draw whose id reaches the verifier.
func TestLotteryHistorySearchAndSelection(t *testing.T) {
	fake := &fakeServices{draws: []DrawRow{
		{ID: "draw-alpha", Seed: "spring", Participants: []string{"Alice"}, Winners: []string{"Alice"}},
		{ID: "draw-beta", Seed: "autumn", Participants: []string{"Bob"}, Winners: []string{"Bob"}},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.search = "autumn"
	tester.Frame()
	require.True(t, tester.HasText("draw-beta"))
	require.False(t, tester.HasText("draw-alpha"), "a seed search must narrow the list")

	require.NoError(t, tester.Click("draw-beta"))
	tester.Frame()
	require.Equal(t, "draw-beta", app.lottery.verifyID)

	app.lottery.search = "nobody"
	tester.Frame()
	require.True(t, tester.HasText("No draws match “nobody”."))
}

// TestOracleSourceSearchAndSelection covers the source list the same way:
// name, id, type and URL are searchable, and a filtered row loads its id into
// the action fields.
func TestOracleSourceSearchAndSelection(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{
		{ID: "src-btc", Name: "btc-price", Type: "json", URL: "https://example.com/btc"},
		{ID: "src-wx", Name: "weather", Type: "json", URL: "https://example.com/wx"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.search = "weather"
	tester.Frame()
	require.True(t, tester.HasText("weather"))
	require.False(t, tester.HasText("btc-price"))

	require.NoError(t, tester.Click("src-wx"))
	tester.Frame()
	require.Equal(t, "src-wx", app.oracle.fetchID)
	require.Equal(t, "src-wx", app.oracle.queryID)
	require.Equal(t, "src-wx", app.oracle.latestID)
}

// TestSearchingClearsTheRowHighlight pins the selection after a query change.
// The table's selection is an index into the filtered rows, so keeping an old
// index after a new query would highlight a different token than the one the
// forms still hold; the operator must pick from the list they are looking at.
func TestSearchingClearsTheRowHighlight(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "Gold", Symbol: "AU", TotalSupply: "10"},
		{ID: "t2", Name: "Silver", Symbol: "AG", TotalSupply: "20"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()

	require.NoError(t, tester.Click("Gold"))
	tester.Frame()
	require.Equal(t, 0, app.token.chosen, "clicking a row chooses it")

	app.token.search = "silver"
	tester.Frame()
	require.Equal(t, "silver", app.token.search)
	require.Equal(t, -1, app.token.chosen,
		"a new query must clear the highlight that indexed the old rows")
}

// TestLedgerSearchAndRowLimit covers the chain page. The ledger keeps every
// entry forever, so the table must be searchable and bounded instead of
// building every row each frame; the counters above it still describe the
// whole chain.
func TestLedgerSearchAndRowLimit(t *testing.T) {
	blocks := make([]BlockRow, 30)
	for i := range blocks {
		blocks[i] = BlockRow{
			Height:  int64(len(blocks) - i),
			Module:  "module-" + strconv.Itoa(i),
			Summary: "payload-" + strconv.Itoa(i),
		}
	}
	fake := &fakeServices{blocks: blocks}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	require.True(t, tester.HasText("payload-0"), "the newest block is listed")
	require.True(t, tester.HasText("payload-29"), "all thirty fit at the default limit")

	app.blockchain.blockLimit = 10
	tester.Frame()
	require.True(t, tester.HasText("payload-9"))
	require.False(t, tester.HasText("payload-10"), "the limit must cut the table")
	require.True(t, tester.HasText("Showing the newest 10 of 30 matching entries"))

	app.blockchain.search = "payload-29"
	tester.Frame()
	require.True(t, tester.HasText("payload-29"), "the search must find the oldest entry too")
	require.False(t, tester.HasText("payload-0"))
	require.False(t, tester.HasText("Showing the newest"), "one match fits the limit")

	app.blockchain.search = "nothing-like-this"
	tester.Frame()
	require.True(t, tester.HasText("No blocks match “nothing-like-this”."))
}
