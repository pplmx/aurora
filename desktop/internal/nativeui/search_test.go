package nativeui

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
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
	tester.Announcements() // the initial page title

	app.token.search = "silver"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a name match must stay listed")
	require.False(t, tester.HasText("Gold"), "a non-match must leave the list")
	require.Equal(t, []string{"1 result."}, tester.Announcements())

	app.token.search = "AG"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a symbol match must stay listed")
	require.Equal(t, []string{"1 result."}, tester.Announcements())

	app.token.search = "nothing-like-this"
	tester.Frame()
	require.True(t, tester.HasText("No tokens match “nothing-like-this”."))
	require.Equal(t, []string{fmt.Sprintf(tr("%d results."), 0)}, tester.Announcements())

	// Choosing a row from the filtered list selects that token, and the choice
	// is spoken because it refills the page's forms.
	app.token.search = "Gold"
	tester.Frame()
	tester.Announcements()
	require.NoError(t, tester.Click("Gold"))
	tester.Frame()
	require.Equal(t, "t1", app.token.tokenID)
	require.Equal(t, []string{tr("Selected Gold")}, tester.Announcements())

	app.token.search = ""
	tester.Frame()
	require.True(t, tester.HasText("Gold"))
	require.True(t, tester.HasText("Silver"))
	require.Equal(t, []string{fmt.Sprintf(tr("%d results."), 2)}, tester.Announcements())
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
		voterRows: []VoterRow{
			{PublicKey: "pk-vera", Name: "Vera", HasVoted: false},
			{PublicKey: "pk-walt", Name: "Walt", HasVoted: true},
		},
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
	require.True(t, tester.HasText("Vera"))

	app.voting.voterSearch = "walt"
	tester.Frame()
	require.True(t, tester.HasText("Walt"))
	require.False(t, tester.HasText("Vera"), "the voter search must narrow the table")

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

	require.NoError(t, tester.Click("pk-walt"))
	tester.Frame()
	require.Equal(t, "pk-walt", app.voting.voteVoterPub,
		"the filtered voter row must fill the ballot with its own public key")
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

// TestLotteryHistorySearchesBeyondTheLoadedPage: the draws search runs in
// storage, so an old draw that is not on the loaded page is still found. A
// page-local filter would have reported "no draws match" instead.
func TestLotteryHistorySearchesBeyondTheLoadedPage(t *testing.T) {
	draws := make([]DrawRow, 0, lotteryHistoryLimit+5)
	for i := 0; i < lotteryHistoryLimit+5; i++ {
		draws = append(draws, DrawRow{ID: fmt.Sprintf("draw-%03d", i), Seed: fmt.Sprintf("seed-%03d", i)})
	}
	oldest := len(draws) - 1
	draws[oldest].Seed = "needle-seed"
	app := New(&fakeServices{draws: draws})
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Lottery")
	tester.Frame()

	require.False(t, tester.HasText("draw-104"), "the oldest draw starts outside the loaded page")

	app.lottery.search = "needle"
	tester.Frame()
	require.True(t, tester.HasText("draw-104"),
		"the search must reach an old draw outside the loaded page")
	require.Contains(t, tester.Announcements(), "1 result.",
		"the repaint is visual; the count must also be spoken")
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

// TestClearingTheSearchRestoresTheWholePage covers the other half of a filter:
// a search narrows the rows already loaded, so emptying the field must bring
// them all back. Every filter does this through containsFold's empty-query
// branch, and a filter written as a one-way narrowing would leave the list
// permanently short with no way to widen it again.
func TestClearingTheSearchRestoresTheWholePage(t *testing.T) {
	fake := &fakeServices{
		sources: []SourceRow{{ID: "s1", Name: "btc"}, {ID: "s2", Name: "weather"}},
		tokenRows: []TokenRow{
			{ID: "t1", Name: "Gold", Symbol: "AU"},
			{ID: "t2", Name: "Silver", Symbol: "AG"},
		},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.search = "weather"
	tester.Frame()
	require.Len(t, app.filteredSources(), 1, "the query narrows the page")
	require.False(t, tester.HasText("btc"))

	app.oracle.search = ""
	tester.Frame()
	require.Len(t, app.filteredSources(), 2, "emptying the field restores the page")
	require.True(t, tester.HasText("btc"))
	require.True(t, tester.HasText("weather"))

	app.ShowPage("Token")
	tester.Frame()
	app.token.search = "gold"
	tester.Frame()
	require.False(t, tester.HasText("Silver"))

	app.token.search = ""
	tester.Frame()
	require.True(t, tester.HasText("Gold"), "emptying restores the token list too")
	require.True(t, tester.HasText("Silver"))
}

// TestClearingTheVotingSearchesRestoresTheirLists covers the three voting
// lists, whose filters live behind filteredVoters, filteredCandidates and
// filteredSessions rather than inline in the view.
func TestClearingTheVotingSearchesRestoresTheirLists(t *testing.T) {
	app := New(&fakeServices{})
	app.voting.voters = []VoterRow{
		{Name: "Ada", PublicKey: "k1"}, {Name: "Bob", PublicKey: "k2"},
	}
	app.voting.candidates = []CandidateRow{
		{ID: "c1", Name: "Ada"}, {ID: "c2", Name: "Bob"},
	}
	app.voting.sessions = []SessionRow{
		{ID: "s1", Title: "Budget"}, {ID: "s2", Title: "Charter"},
	}

	for _, c := range []struct {
		name  string
		set   *string
		count func() int
		query string
	}{
		{"voters", &app.voting.voterSearch, func() int { return len(app.filteredVoters()) }, "ada"},
		{"candidates", &app.voting.candSearch, func() int { return len(app.filteredCandidates()) }, "bob"},
		{"sessions", &app.voting.sessionSearch, func() int { return len(app.filteredSessions()) }, "budget"},
	} {
		t.Run(c.name, func(t *testing.T) {
			*c.set = c.query
			require.Equal(t, 1, c.count(), "the query narrows the page")
			*c.set = ""
			require.Equal(t, 2, c.count(), "emptying the field restores the page")
		})
	}
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
	require.True(t, tester.HasText("Entries 1–10 of 30"))
	require.Equal(t, 10, fake.lastBlockLimit, "an unfiltered page stays bounded")

	app.blockchain.search = "  payload-29  "
	tester.Frame()
	require.True(t, tester.HasText("payload-29"), "the search must find the oldest entry too")
	require.False(t, tester.HasText("payload-0"))
	require.Equal(t, "payload-29", fake.lastBlockSearch)
	require.Equal(t, 10, fake.lastBlockLimit, "search must request only one bounded page")
	require.Equal(t, 0, fake.lastBlockOffset)
	require.Equal(t, 1, app.blockchain.blockTotal)
	calls := fake.blockListCalls
	tester.Frame()
	require.Equal(t, calls, fake.blockListCalls, "surrounding spaces must not repeat the same query")

	app.blockchain.search = "nothing-like-this"
	tester.Frame()
	require.True(t, tester.HasText("No blocks match “nothing-like-this”."))
}

// TestLedgerSearchPagesEveryMatchWithoutLoadingWholeChain keeps search global
// while bounding both the rows returned and the page walk. The old view asked
// for every block on each search and sliced the matches itself.
func TestLedgerSearchPagesEveryMatchWithoutLoadingWholeChain(t *testing.T) {
	blocks := make([]BlockRow, 40)
	for i := range blocks {
		summary := "other-" + strconv.Itoa(i)
		if i%2 == 0 {
			summary = "needle-" + strconv.Itoa(i)
		}
		blocks[i] = BlockRow{
			Height:  int64(len(blocks) - i),
			Module:  "lottery",
			Summary: summary,
		}
	}
	fake := &fakeServices{blocks: blocks}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	app.blockchain.blockLimit = 10
	tester.Frame()

	app.blockchain.search = "needle"
	tester.Frame()
	require.Equal(t, 10, fake.lastBlockLimit, "search must not ask for the whole chain")
	require.Equal(t, 0, fake.lastBlockOffset)
	require.Equal(t, 20, app.blockchain.blockTotal)
	require.True(t, tester.HasText("needle-0"))
	require.True(t, tester.HasText("needle-18"))
	require.False(t, tester.HasText("needle-20"))
	require.True(t, tester.HasText("Entries 1–10 of 20"))

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, "needle", fake.lastBlockSearch)
	require.Equal(t, 10, fake.lastBlockLimit)
	require.Equal(t, 10, fake.lastBlockOffset)
	require.True(t, tester.HasText("needle-20"))
	require.True(t, tester.HasText("needle-38"))
	require.False(t, tester.HasText("needle-18"))
	require.True(t, tester.HasText("Entries 11–20 of 20"))
}

// TestLedgerPagesMoveThroughOlderEntries covers the paging controls: the page
// query is bounded, Next moves the offset, and Previous returns.
func TestLedgerPagesMoveThroughOlderEntries(t *testing.T) {
	blocks := make([]BlockRow, 12)
	for i := range blocks {
		blocks[i] = BlockRow{Height: int64(12 - i), Module: "lottery", Summary: "payload-" + strconv.Itoa(i)}
	}
	app := New(&fakeServices{blocks: blocks})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	app.blockchain.blockLimit = 10
	tester.Frame()

	require.Equal(t, 0, app.blockchain.blockOffset)
	require.True(t, tester.HasText("Entries 1–10 of 12"))
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 10, app.blockchain.blockOffset)
	require.True(t, tester.HasText("Entries 11–12 of 12"))
	require.NoError(t, tester.Click("Previous"))
	tester.Frame()
	require.Equal(t, 0, app.blockchain.blockOffset)
}

// TestFailedLedgerPageMoveKeepsCurrentPage mirrors the other paged lists: Next
// commits its offset only after the new rows arrive. A failed move used to
// advance the pager while the table still held the previous page.
func TestFailedLedgerPageMoveKeepsCurrentPage(t *testing.T) {
	blocks := make([]BlockRow, 150)
	for i := range blocks {
		blocks[i] = BlockRow{Height: int64(len(blocks) - i), Module: "lottery", Summary: "row-" + strconv.Itoa(i)}
	}
	fake := &fakeServices{blocks: blocks}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()

	app.blockchain.blockLimit = 100
	tester.Frame()
	require.Equal(t, 0, app.blockchain.blockOffset)
	require.True(t, tester.HasText("Entries 1–100 of 150"))

	fake.loadErrs.blocks = errors.New("database is locked")
	require.NoError(t, tester.Click("Next"))
	tester.Frame()

	require.Equal(t, 0, app.blockchain.blockOffset,
		"a failed Next must leave the offset with the rows it actually loaded")
	require.True(t, tester.HasText("Entries 1–100 of 150"))
	require.True(t, tester.HasText("Couldn't read chain blocks"))
}

// TestSearchHintsNameTheirCollection pins the hint MyGo draws in an empty
// search field. Its own placeholder is the untranslated word "Search"; each
// field now names the collection it narrows, in the window's language.
func TestSearchHintsNameTheirCollection(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	app := New(&fakeServices{
		draws:     []DrawRow{{ID: "d1"}},
		tokenRows: []TokenRow{{ID: "t1", Name: "Gold"}},
		blocks:    []BlockRow{{Height: 1, Module: "lottery"}},
	})
	tester := ui.NewTester(app.View, 1280, 3000)

	app.ShowPage("Lottery")
	tester.Frame()
	require.True(t, tester.HasText("Search draws"), "the hint must name the collection")

	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("Search tokens"))

	app.ShowPage("Blockchain")
	tester.Frame()
	require.True(t, tester.HasText("Search blocks"))

	translator.SetLocale("zh")
	app.ShowPage("Lottery")
	tester.Frame()
	require.True(t, tester.HasText("搜索抽奖"), "the hint must follow the window's language")
}

// TestEmptyChainHidesItsSearch pins the first-run shape of the Ledger list:
// with only the genesis block there is nothing to search or page through, so
// the controls wait for the first module record. A leftover query still shows
// the field, because the field is what explains its empty result.
func TestEmptyChainHidesItsSearch(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText("No module records yet; the first write adds one."))
	require.False(t, tester.HasText("Search blocks"))
	require.False(t, tester.HasText("Show"))

	app.blockchain.search = "draw"
	tester.Frame()
	require.True(t, tester.HasText("Search blocks"),
		"a query left from another session keeps the field that explains it")
	require.True(t, tester.HasText("No blocks match “draw”."))
}

// TestClearingAQueryWhileTheListIsEmptyIsRecorded covers the voter, candidate
// and session lists while their store is empty. Their applied-query field was
// updated after the empty-list return, so a query cleared at that moment kept
// looking applied: the first-record guide stayed hidden — the page showed a work
// bench over an empty store — and the next read announced itself as a search
// result.
func TestClearingAQueryWhileTheListIsEmptyIsRecorded(t *testing.T) {
	fake := &fakeServices{voterRows: []VoterRow{{Name: "Ada", PublicKey: "k1"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()
	require.False(t, app.votingNeedsFirstRecord())

	app.voting.voterSearch = "ada"
	tester.Frame()
	require.Equal(t, "ada", app.voting.voterApplied)

	// The store empties under the still-applied query, then the reader clears it.
	fake.voterRows = nil
	app.loadVoting()
	tester.Frame()
	app.voting.voterSearch = ""
	tester.Frame()

	require.Empty(t, app.voting.voterApplied,
		"a cleared query must be recorded even when no row survived the read")
	require.True(t, app.votingNeedsFirstRecord(),
		"an empty store shows the first-record guide, not a work bench")
}

// TestClearingAQueryWhileTheStoreIsEmpty covers the token, NFT and oracle lists,
// which had the same ordering as the voter list: the applied-query update sat
// after the empty-list return, so a query cleared at that moment was never
// recorded. Each of those pages gates its first-record panel on the applied
// query, so the stale value showed a work bench over an empty store.
func TestClearingAQueryWhileTheStoreIsEmpty(t *testing.T) {
	t.Run("tokens", func(t *testing.T) {
		fake := &fakeServices{tokenRows: []TokenRow{{ID: "GLD", Name: "Gold", Symbol: "GLD"}}}
		app := New(fake)
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Token")
		tester.Frame()
		require.False(t, app.tokenNeedsFirstRecord())

		app.token.search = "gold"
		tester.Frame()
		require.Equal(t, "gold", app.token.appliedSearch)

		fake.tokenRows = nil
		app.loadTokenPage(0)
		tester.Frame()
		app.token.search = ""
		tester.Frame()
		require.Empty(t, app.token.appliedSearch,
			"a cleared query is recorded even when no token survived the read")
		require.True(t, app.tokenNeedsFirstRecord())
	})

	t.Run("nfts", func(t *testing.T) {
		fake := &fakeServices{nftRows: []NFTRow{{ID: "n1", Name: "Star"}}}
		app := New(fake)
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("NFT")
		tester.Frame()
		require.False(t, app.nftNeedsFirstRecord())

		app.nft.search = "star"
		tester.Frame()
		require.Equal(t, "star", app.nft.appliedSearch)

		fake.nftRows = nil
		app.loadNFTPage(0)
		tester.Frame()
		app.nft.search = ""
		tester.Frame()
		require.Empty(t, app.nft.appliedSearch,
			"a cleared query is recorded even when no asset survived the read")
		require.True(t, app.nftNeedsFirstRecord())
	})

	t.Run("oracle sources", func(t *testing.T) {
		fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}}
		app := New(fake)
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Oracle")
		tester.Frame()

		app.oracle.search = "btc"
		tester.Frame()
		require.Equal(t, "btc", app.oracle.appliedSearch)

		fake.sources = nil
		app.loadOracleSourcePage(0)
		tester.Frame()
		app.oracle.search = ""
		tester.Frame()
		require.Empty(t, app.oracle.appliedSearch,
			"a cleared query is recorded even when no source survived the read")
	})
}

// TestTheTokenSearchSpansTheWholeStore covers what the hint promises. The list is
// paged, so a token on an older page is not in memory; the search has to be a
// storage query, or the reader concludes the token does not exist.
func TestTheTokenSearchSpansTheWholeStore(t *testing.T) {
	rows := make([]TokenRow, 150)
	for i := range rows {
		rows[i] = TokenRow{ID: "t" + strconv.Itoa(i), Name: "Token " + strconv.Itoa(i), Symbol: "T"}
	}
	// The row that matters sits on page 2, well away from the loaded page.
	rows[120].Name = "Needle"
	fake := &fakeServices{tokenRows: rows}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()
	require.Len(t, app.token.tokens, tokenPageSize, "page one is loaded")
	require.False(t, tester.HasText("Needle"), "the row is not on the loaded page")

	app.token.search = "needle"
	tester.Frame()

	require.True(t, tester.HasText("Needle"), "the search must find a row outside the loaded page")
	require.Equal(t, "needle", fake.lastTokenSearch, "and it must be the term storage was asked for")
	require.Len(t, app.token.tokens, 1)
}

// TestASearchThatMatchesNothingIsNotAnEmptyStore keeps the two sentences apart.
// The store's own state decides which is true: an empty result from a search over
// a populated store needs "no match", because clearing the field is the fix.
func TestASearchThatMatchesNothingIsNotAnEmptyStore(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "Gold", Symbol: "AU", TotalSupply: "10"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.search = "nothing-like-this"
	tester.Frame()

	require.True(t, tester.HasText("No tokens match “nothing-like-this”."))
	require.False(t, tester.HasText("No tokens yet"),
		"a populated store must not be described as empty")
}

// TestTheNFTSearchSpansTheWholeStore is the NFT half of the token rule: the
// collection is paged, so an asset on an older page is not in memory and a
// page-local filter could not find it.
func TestTheNFTSearchSpansTheWholeStore(t *testing.T) {
	rows := make([]NFTRow, 150)
	for i := range rows {
		rows[i] = NFTRow{ID: "n" + strconv.Itoa(i), Name: "NFT " + strconv.Itoa(i), Owner: "owner"}
	}
	rows[130].Name = "Needle"
	fake := &fakeServices{nftRows: rows}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()
	require.False(t, tester.HasText("Needle"), "the row is not on the loaded page")

	app.nft.search = "needle"
	tester.Frame()

	require.True(t, tester.HasText("Needle"), "the search must find a row outside the loaded page")
	require.Equal(t, "needle", fake.lastNFTSearch, "and it must be the term storage was asked for")
	require.Len(t, app.nft.list, 1)
}

// TestAnNFTSearchThatMatchesNothingIsNotAnEmptyCollection keeps the two sentences
// apart, as the token list does: clearing the field is the fix for one and
// minting the first asset is the fix for the other.
func TestAnNFTSearchThatMatchesNothingIsNotAnEmptyCollection(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{{ID: "n1", Name: "Star", Owner: "owner"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.search = "nothing-like-this"
	tester.Frame()

	require.True(t, tester.HasText("No NFTs match “nothing-like-this”."))
	require.False(t, tester.HasText("No NFTs yet"),
		"a populated collection must not be described as empty")
}

// TestTheSourceSearchSpansTheWholeStore completes the rule for the last paged
// collection. A registered feed whose row is on an older page was unreachable by
// name, which for a source — an id nobody reads in full — meant the only way back
// to it was to remember its URL.
func TestTheSourceSearchSpansTheWholeStore(t *testing.T) {
	rows := make([]SourceRow, 120)
	for i := range rows {
		rows[i] = SourceRow{
			ID: "s" + strconv.Itoa(i), Name: "feed " + strconv.Itoa(i),
			URL: "https://example.com/" + strconv.Itoa(i), Type: "json",
		}
	}
	rows[110].Name = "needle"
	fake := &fakeServices{sources: rows}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Oracle")
	tester.Frame()
	require.False(t, tester.HasText("needle"), "the row is not on the loaded page")

	app.oracle.search = "needle"
	tester.Frame()

	require.True(t, tester.HasText("needle"), "the search must find a row outside the loaded page")
	require.Len(t, app.oracle.sources, 1)
}

// TestANewSourceAppearsEvenWithASearchOpen covers the write that follows a search.
// The read after a create must not carry the old term: a source the term does not
// match would come back missing, so the write would report success while the list
// showed nothing new — and the highlight would point at nothing.
func TestANewSourceAppearsEvenWithASearchOpen(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2600)
	app.ShowPage("Oracle")
	tester.Frame()
	app.oracle.search, app.oracle.appliedSearch = "nothing-matches", "nothing-matches"
	tester.Frame()

	fake.addSource = func(SourceInput) (SourceRow, error) {
		row := SourceRow{ID: "new", Name: "New source", Type: "json", URL: "https://example.com/new"}
		fake.sources = append([]SourceRow{row}, fake.sources...)
		return row, nil
	}
	app.oracle.addName, app.oracle.addURL = "New source", "https://example.com/new"
	app.addOracleSource()
	tester.Frame()

	require.Zero(t, app.oracle.offset, "the new row is newest-first, so it is on page zero")
	require.Equal(t, 0, app.oracle.selected, "and it is the chosen row")
	require.Empty(t, app.oracle.appliedSearch, "the search was cleared by the write")
	require.True(t, tester.HasText("New source"))
}
