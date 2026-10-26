package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestSuccessfulTokenWriteRefreshesWhatThePageShows pins the post-write
// refresh. The collection rows show total supply and the Inspect card shows
// the token's metadata; leaving either at its pre-write value beside a "Done"
// status read as if the write had changed nothing.
func TestSuccessfulTokenWriteRefreshesWhatThePageShows(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", TotalSupply: "100"}}}
	app := New(fake)
	app.token.tokenID = "t1"
	app.token.info = &TokenRow{ID: "t1", Name: "Gold", TotalSupply: "100"}

	tokensBefore, infoBefore := fake.listTokensCalls, fake.tokenInfoCalls
	app.token.mintTo = "owner-pk"
	app.token.mintAmount = "5"
	app.token.mintPriv = "priv"
	app.mintToken()

	require.False(t, app.token.status.isErr)
	require.Empty(t, app.token.mintPriv, "a used signing key must not stay on screen")
	require.Equal(t, tokensBefore+1, fake.listTokensCalls, "the collection must be re-read after a mint")
	require.Equal(t, infoBefore+1, fake.tokenInfoCalls, "an open Inspect readout must be re-read after a mint")
}

// TestSuccessfulNFTActionRefreshesCollectionAndHistory covers the same rule on
// the NFT page: a transfer changes the owner shown in the collection, and the
// history gains the transfer that was just recorded.
func TestSuccessfulNFTActionRefreshesCollectionAndHistory(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{{ID: "n1", Name: "Star", Owner: "owner-a"}}}
	app := New(fake)
	app.nft.transferID = "n1"
	app.nft.nft = &NFTRow{ID: "n1", Name: "Star", Owner: "owner-a"}
	app.nft.nftLoadedID = "n1"
	app.nft.historyID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint"}}
	app.nft.historyLoadedID = "n1"

	collectionBefore, historyBefore := fake.listAllNFTsCalls, fake.nftHistoryCalls
	app.nft.from = "owner-a"
	app.nft.to = "owner-b"
	app.nft.priv = "priv"
	app.transferNFT()

	require.False(t, app.nft.status.isErr)
	require.Empty(t, app.nft.priv, "a used signing key must not stay on screen")
	require.Equal(t, collectionBefore+1, fake.listAllNFTsCalls, "the collection must be re-read after a transfer")
	require.Equal(t, historyBefore+1, fake.nftHistoryCalls, "an open history must be re-read after a transfer")
}

// TestCastVoteRefreshesAnOpenTally pins the ballot path: a tally already on
// screen for the session that just accepted the ballot must include it, or the
// unchanged numbers contradict "Ballot recorded."
func TestCastVoteRefreshesAnOpenTally(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	app.voting.voteSession = "s1"
	app.voting.voteCandidate = "c1"
	app.voting.voteVoterPub = "pk"
	app.voting.votePriv = "sk"
	app.voting.resultsSession = "s1"
	app.voting.results = []ResultRow{{CandidateID: "c1", Votes: 1}}

	before := fake.sessionResultsCalls
	app.castVote()

	require.Equal(t, "Ballot recorded.", app.voting.voteResult.text)
	require.False(t, app.voting.voteResult.isErr)
	require.Equal(t, before+1, fake.sessionResultsCalls,
		"the open tally for this session must be re-read after a ballot")
}

// TestOracleFetchRefreshesTheReadCardsItFeeds pins the fetch path: a reading
// just stored must appear in the Query and Latest cards when they are showing
// that source, or "Done" sits beside the value from before the fetch.
func TestOracleFetchRefreshesTheReadCardsItFeeds(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3200)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.fetchID = "s1"
	app.oracle.queryID = "s1"
	app.oracle.data = []DataRow{{Value: "old"}}
	app.oracle.latestID = "s1"
	app.oracle.latest = &DataRow{Value: "old"}
	tester.Frame()

	dataBefore, latestBefore := fake.oracleDataCalls, fake.oracleLatestCalls
	require.NoError(t, tester.Click("Fetch now"))
	tester.Frame()

	require.False(t, app.oracle.status.isErr)
	require.Equal(t, dataBefore+1, fake.oracleDataCalls, "the open Query card must be re-read after a fetch")
	require.Equal(t, latestBefore+1, fake.oracleLatestCalls, "the open Latest card must be re-read after a fetch")

	// A card showing another source is left alone.
	app.oracle.queryID = "other"
	app.oracle.data = []DataRow{{Value: "old"}}
	dataBefore = fake.oracleDataCalls
	require.NoError(t, tester.Click("Fetch now"))
	tester.Frame()
	require.Equal(t, dataBefore, fake.oracleDataCalls, "a card for another source must not be re-read")
}

// TestDeletingASourceClearsItsReadings covers the other direction: the cards
// that were showing the deleted source no longer describe a live feed.
func TestDeletingASourceClearsItsReadings(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}}
	app := New(fake)
	app.oracle.fetchID = "s1"
	app.oracle.queryID = "s1"
	app.oracle.data = []DataRow{{Value: "old"}}
	app.oracle.latestID = "s1"
	app.oracle.latest = &DataRow{Value: "old"}

	app.deleteOracleSource()

	require.False(t, app.oracle.status.isErr)
	require.Empty(t, app.oracle.data)
	require.Nil(t, app.oracle.latest)
}

// TestNumericFormLimitsAreBounded pins the number inputs that feed list
// limits: the defaults match the page, and a value past the ceiling is capped
// rather than sent to the repository.
func TestNumericFormLimitsAreBounded(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 6000)
	app.ShowPage("Token")
	tester.Frame()

	require.NoError(t, tester.Click("Load History"))
	tester.Frame()
	require.Equal(t, 50, fake.tokenHistoryLimit, "the default page size is 50")

	app.token.historyLimit = 5000
	require.NoError(t, tester.Click("Load History"))
	tester.Frame()
	require.Equal(t, maxListLimit, fake.tokenHistoryLimit, "an oversized limit must be capped")

	app.ShowPage("Oracle")
	tester.Frame()
	require.NoError(t, tester.Click("Query"))
	tester.Frame()
	require.Equal(t, 20, fake.oracleDataLimit, "the default query limit is 20")

	app.oracle.queryLim = 5000
	require.NoError(t, tester.Click("Query"))
	tester.Frame()
	require.Equal(t, maxListLimit, fake.oracleDataLimit)
}

// TestPageRefreshReReadsOpenReadouts pins the Refresh action beyond the lists:
// a readout that is on screen (token info/balance/history, NFT details/history,
// oracle Query/Latest, voting tally) must follow the database, or Ctrl+R
// refreshes the page around stale answers.
func TestPageRefreshReReadsOpenReadouts(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)

	app.ShowPage("Token")
	app.token.tokenID = "t1"
	app.token.balanceAccount = "owner"
	app.token.historyOwner = "owner"
	app.token.info = &TokenRow{ID: "t1"}
	app.token.balance = "42"
	app.token.history = []TransferRow{{Amount: "1"}}
	beforeInfo, beforeBalance, beforeHistory := fake.tokenInfoCalls, fake.tokenBalanceCalls, fake.tokenHistoryCalls
	app.refresh()
	require.Equal(t, beforeInfo+1, fake.tokenInfoCalls, "an open Inspect readout must refresh")
	require.Equal(t, beforeBalance+1, fake.tokenBalanceCalls, "a shown balance must refresh")
	require.Equal(t, beforeHistory+1, fake.tokenHistoryCalls, "a loaded history must refresh")

	app.ShowPage("NFT")
	app.nft.id = "n1"
	app.nft.nft = &NFTRow{ID: "n1"}
	app.nft.nftLoadedID = "n1"
	app.nft.historyID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint"}}
	app.nft.historyLoadedID = "n1"
	beforeDetails, beforeHistory := fake.getNFTCalls, fake.nftHistoryCalls
	app.refresh()
	require.Equal(t, beforeDetails+1, fake.getNFTCalls, "an open NFT readout must refresh")
	require.Equal(t, beforeHistory+1, fake.nftHistoryCalls, "a loaded NFT history must refresh")

	app.ShowPage("Oracle")
	app.oracle.data = []DataRow{{Value: "1"}}
	app.oracle.latest = &DataRow{Value: "1"}
	beforeData, beforeLatest := fake.oracleDataCalls, fake.oracleLatestCalls
	app.refresh()
	require.Equal(t, beforeData+1, fake.oracleDataCalls, "an open Query card must refresh")
	require.Equal(t, beforeLatest+1, fake.oracleLatestCalls, "an open Latest card must refresh")

	app.ShowPage("Voting")
	app.voting.resultsSession = "s1"
	app.voting.results = []ResultRow{{CandidateID: "c1", Votes: 1}}
	beforeResults := fake.sessionResultsCalls
	app.refresh()
	require.Equal(t, beforeResults+1, fake.sessionResultsCalls, "an open tally must refresh")
}

// TestBalanceAndHistoryUseTheirOwnAccount pins the two read inputs apart: they
// used to share one field, so typing a history owner silently changed the
// account the Balance card read, under a readout fetched for someone else.
func TestBalanceAndHistoryUseTheirOwnAccount(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.balanceAccount = "alice"
	app.token.historyOwner = "bob"

	require.NoError(t, tester.Click("Get Balance"))
	require.NoError(t, tester.Click("Load History"))
	require.Equal(t, "alice", fake.lastBalanceOwner, "the Balance card reads its own account")
	require.Equal(t, "bob", fake.lastHistoryOwner, "the History card reads its own owner")
}

// TestTokenHistoryPagesWithItsOffset pins the History card's offset control:
// the repository and API page a long transfer stream, and without an offset the
// window could only ever show the first page.
func TestTokenHistoryPagesWithItsOffset(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.historyLimit = 10
	app.token.historyOffset = 5

	require.NoError(t, tester.Click("Load History"))
	tester.Frame()
	require.Equal(t, 10, fake.tokenHistoryLimit)
	require.Equal(t, 5, fake.lastHistoryOffset, "the History card must page from its offset")
}

// TestReloadRefreshesTheCurrentPage pins the entry point an import uses: a
// change made outside the view (a menu/drag/forwarded import) re-reads the page
// it affects instead of leaving the pre-import history on screen.
func TestReloadRefreshesTheCurrentPage(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold"}}}
	app := New(fake)
	app.ShowPage("Token")
	before := fake.listTokensCalls

	app.Reload()
	require.Equal(t, before+1, fake.listTokensCalls, "Reload must re-read the current page")
}

// TestTokenReadoutFailureKeepsSameSubjectRows applies the stale-load policy to
// the three Token readouts: a failed refresh of the same token/account/query
// keeps the last successful answer and marks it stale instead of blanking it
// or hiding the error.
func TestTokenReadoutFailureKeepsSameSubjectRows(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.info = &TokenRow{ID: "t1", Name: "Gold", TotalSupply: "100"}
	app.token.infoLoadedID = "t1"
	app.token.balanceAccount = "owner"
	app.token.balance = "42"
	app.token.balanceLoadedToken, app.token.balanceLoadedOwner = "t1", "owner"
	app.token.historyOwner = "owner"
	app.token.history = []TransferRow{{From: "alice", To: "bob", Amount: "1", Nonce: 1}}
	app.token.historyLoadedToken, app.token.historyLoadedOwner = "t1", "owner"
	app.token.historyLoadedLimit, app.token.historyLoadedOffset = 50, 0
	tester.Frame()

	boom := errors.New("database is locked")
	fake.loadErrs = fakeLoadErrors{tokenInfo: boom, tokenBalance: boom, tokenHistory: boom}
	app.refresh()
	tester.Frame()

	require.Equal(t, "Gold", app.token.info.Name)
	require.Equal(t, "42", app.token.balance)
	require.Len(t, app.token.history, 1)
	require.ErrorIs(t, app.token.infoErr, boom)
	require.ErrorIs(t, app.token.balanceErr, boom)
	require.ErrorIs(t, app.token.historyErr, boom)
	for _, warning := range []string{
		"Couldn't load token info", "Couldn't load balance", "Couldn't load history",
	} {
		require.Truef(t, tester.HasText(warning), "%s must be visible", warning)
	}
	require.True(t, tester.HasText("Showing the last successful load."))
	require.False(t, tester.HasText("No transfers loaded yet."),
		"a failed read must not claim there is no history")
}

// TestTokenSelectionDropsReadoutsWhenTheNewReadFails pins the key side: the
// previous token's metadata, balance and history must be gone before the new
// token is read, so a failure cannot leave them under the new id.
func TestTokenSelectionDropsReadoutsWhenTheNewReadFails(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "FirstToken", TotalSupply: "1"},
		{ID: "t2", Name: "SecondToken", TotalSupply: "2"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.info = &TokenRow{ID: "t1", Name: "OldToken"}
	app.token.infoLoadedID = "t1"
	app.token.balanceAccount = "owner"
	app.token.balance = "42"
	app.token.balanceLoadedToken, app.token.balanceLoadedOwner = "t1", "owner"
	app.token.history = []TransferRow{{From: "old-from", Amount: "1"}}
	app.token.historyLoadedToken, app.token.historyLoadedOwner = "t1", "owner"
	app.token.historyLoadedLimit, app.token.historyLoadedOffset = 50, 0
	tester.Frame()

	boom := errors.New("database is locked")
	fake.loadErrs = fakeLoadErrors{tokenInfo: boom, tokenBalance: boom, tokenHistory: boom}
	require.NoError(t, tester.Click("t2"))
	tester.Frame()

	require.Equal(t, "t2", app.token.tokenID)
	require.Nil(t, app.token.info, "the old token's metadata must be gone")
	require.Empty(t, app.token.balance, "the old token's balance must be gone")
	require.Empty(t, app.token.history, "the old token's history must be gone")
	require.True(t, tester.HasText("Couldn't load token info"))
	require.True(t, tester.HasText("Couldn't load balance"))
	require.True(t, tester.HasText("Couldn't load history"))
	require.False(t, tester.HasText("OldToken"))
	require.False(t, tester.HasText("No transfers loaded yet."))
}

// TestTokenHistoryQueryChangeInvalidatesRows pins the other half of the
// history key: changing Limit or Offset makes the old rows answer a different
// question, so a failed read must not leave them in place.
func TestTokenHistoryQueryChangeInvalidatesRows(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.history = []TransferRow{{From: "old-page", Amount: "1"}}
	app.token.historyLoadedToken, app.token.historyLoadedOwner = "t1", ""
	app.token.historyLoadedLimit, app.token.historyLoadedOffset = 50, 0
	tester.Frame()
	require.True(t, tester.HasText("old-page"))

	app.token.historyOffset = 50
	fake.loadErrs.tokenHistory = errors.New("database is locked")
	require.NoError(t, tester.Click("Load History"))
	tester.Frame()

	require.Empty(t, app.token.history, "rows from the previous offset must be dropped")
	require.True(t, tester.HasText("Couldn't load history"))
	require.False(t, tester.HasText("old-page"))
	require.False(t, tester.HasText("No transfers loaded yet."))
}

// TestOracleReadoutFailureKeepsSameSourceRows applies the stale-load policy to
// the Query and Latest cards: a failed refresh of the same source keeps the
// last successful answer and marks it stale instead of blanking it or hiding
// the error.
func TestOracleReadoutFailureKeepsSameSourceRows(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.queryID = "s1"
	app.oracle.data = []DataRow{{Value: "old-reading"}}
	app.oracle.dataLoadedSource, app.oracle.dataLoadedLimit = "s1", 20
	app.oracle.latestID = "s1"
	app.oracle.latest = &DataRow{Value: "old-latest"}
	app.oracle.latestLoadedSource = "s1"
	tester.Frame()

	boom := errors.New("database is locked")
	fake.loadErrs = fakeLoadErrors{oracleData: boom, oracleLatest: boom}
	app.refresh()
	tester.Frame()

	require.Len(t, app.oracle.data, 1)
	require.NotNil(t, app.oracle.latest)
	require.ErrorIs(t, app.oracle.dataErr, boom)
	require.ErrorIs(t, app.oracle.latestErr, boom)
	require.True(t, tester.HasText("Couldn't load readings"))
	require.True(t, tester.HasText("Couldn't load the latest value"))
	require.True(t, tester.HasText("Showing the last successful load."))
}

// TestOracleSelectionDropsReadingsWhenTheNewReadFails pins the key side: the
// previous source's Query and Latest values must be gone before the new source
// is read, so a failure cannot show them under the new id.
func TestOracleSelectionDropsReadingsWhenTheNewReadFails(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{
		{ID: "s1", Name: "btc"},
		{ID: "s2", Name: "eth"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.data = []DataRow{{Value: "old-reading"}}
	app.oracle.dataLoadedSource, app.oracle.dataLoadedLimit = "s1", 20
	app.oracle.latest = &DataRow{Value: "old-latest"}
	app.oracle.latestLoadedSource = "s1"
	tester.Frame()
	require.True(t, tester.HasText("old-reading"))

	boom := errors.New("database is locked")
	fake.loadErrs = fakeLoadErrors{oracleData: boom, oracleLatest: boom}
	require.NoError(t, tester.Click("s2"))
	tester.Frame()

	require.Equal(t, "s2", app.oracle.queryID)
	require.Equal(t, "s2", app.oracle.latestID)
	require.Empty(t, app.oracle.data, "the old source's readings must be gone")
	require.Nil(t, app.oracle.latest, "the old source's latest value must be gone")
	require.True(t, tester.HasText("Couldn't load readings"))
	require.True(t, tester.HasText("Couldn't load the latest value"))
	require.False(t, tester.HasText("old-reading"))
	require.False(t, tester.HasText("old-latest"))
}

// TestOracleQueryChangeInvalidatesRows pins the limit side of the Query key: a
// different page size makes the old rows answer a different question, so a
// failed read must not leave them in place.
func TestOracleQueryChangeInvalidatesRows(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.queryID = "s1"
	app.oracle.data = []DataRow{{Value: "old-page"}}
	app.oracle.dataLoadedSource, app.oracle.dataLoadedLimit = "s1", 20
	tester.Frame()
	require.True(t, tester.HasText("old-page"))

	app.oracle.queryLim = 50
	fake.loadErrs.oracleData = errors.New("database is locked")
	require.NoError(t, tester.Click("Query"))
	tester.Frame()

	require.Empty(t, app.oracle.data, "rows from the previous limit must be dropped")
	require.True(t, tester.HasText("Couldn't load readings"))
	require.False(t, tester.HasText("old-page"))
}
