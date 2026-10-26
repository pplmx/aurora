package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTokenSelectionRefreshesOpenReadouts pins that choosing another token
// cannot leave the previous token's metadata, balance or history on screen:
// the readouts do not name their token, so stale answers would read as the
// chosen one's.
func TestTokenSelectionRefreshesOpenReadouts(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "First", TotalSupply: "1"},
		{ID: "t2", Name: "Second", TotalSupply: "2"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.info = &TokenRow{ID: "t1"}
	app.token.balanceAccount = "owner"
	app.token.historyOwner = "owner"
	app.token.balance = "42"
	app.token.history = []TransferRow{{Amount: "1"}}

	require.NoError(t, tester.Click("t2"))
	tester.Frame()

	require.Equal(t, "t2", app.token.tokenID)
	require.Equal(t, "t2", fake.lastTokenInfoID, "the Inspect card must follow the selection")
	require.Equal(t, "t2", fake.lastBalanceToken, "a shown balance must follow the selection")
	require.Equal(t, "t2", fake.lastHistoryToken, "a loaded history must follow the selection")
}

// TestNFTSelectionRefreshesOpenReadouts covers the same rule for the asset
// detail and history cards.
func TestNFTSelectionRefreshesOpenReadouts(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First", Owner: "owner-a"},
		{ID: "n2", Name: "Second", Owner: "owner-b"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.nft = &NFTRow{ID: "n1"}
	app.nft.history = []NFTOpRow{{Type: "mint"}}

	require.NoError(t, tester.Click("n2"))
	tester.Frame()

	require.Equal(t, "n2", app.nft.id)
	require.Equal(t, "owner-b", app.nft.from, "the chosen asset's holder fills Transfer")
	require.Equal(t, "owner-b", app.nft.burnOwner, "the chosen asset's holder fills Burn")
	require.Equal(t, "n2", fake.lastNFTID, "the Inspect card must follow the selection")
	require.Equal(t, "n2", fake.lastNFTHistoryID, "a loaded history must follow the selection")
}

// TestOracleSelectionRefreshesReadings covers the Query and Latest cards: they
// are keyed by the id field the row fills, so they must re-read it rather than
// keep the previous source's values.
func TestOracleSelectionRefreshesReadings(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{
		{ID: "s1", Name: "btc"},
		{ID: "s2", Name: "eth"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3200)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.data = []DataRow{{Value: "1"}}
	app.oracle.latest = &DataRow{Value: "1"}

	require.NoError(t, tester.Click("s2"))
	tester.Frame()

	require.Equal(t, "s2", app.oracle.queryID)
	require.Equal(t, "s2", fake.lastOracleDataSource, "the Query card must follow the selection")
	require.Equal(t, "s2", fake.lastOracleLatestSource, "the Latest card must follow the selection")
}

// TestLotterySelectionClearsTheOldVerdict keeps a verdict from a previous draw
// from sitting under a newly chosen one; the next Examine fills it again.
func TestLotterySelectionClearsTheOldVerdict(t *testing.T) {
	fake := &fakeServices{draws: []DrawRow{
		{ID: "draw-a", Winners: []string{"Alice"}},
		{ID: "draw-b", Winners: []string{"Bob"}},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.verifyResult = &VerifyResult{ID: "draw-a", Valid: true}

	require.NoError(t, tester.Click("draw-b"))
	tester.Frame()

	require.Equal(t, "draw-b", app.lottery.verifyID)
	require.Nil(t, app.lottery.verifyResult, "the previous draw's verdict must not stay on screen")
}

// TestSessionSelectionMovesTheOpenTally extends the same rule to the voting
// Results card: selecting another session must not leave the previous
// session's tally under the new session id.
func TestSessionSelectionMovesTheOpenTally(t *testing.T) {
	fake := &fakeServices{sessions: []SessionRow{
		{ID: "s1", Title: "Budget", Status: "active"},
		{ID: "s2", Title: "Standup", Status: "active"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	app.voting.resultsSession = "s1"
	app.voting.results = []ResultRow{{CandidateID: "c1", Votes: 1}}

	require.NoError(t, tester.Click("s2"))
	tester.Frame()

	require.Equal(t, "s2", app.voting.resultsSession)
	require.Equal(t, "s2", fake.lastSessionResultsID, "the open tally must follow the selection")
}

// TestCreateMovesOpenReadoutsToTheNewRecord covers the create paths: a new
// token, NFT or draw becomes the subject of the page, so any readout already
// on screen must move with it instead of describing the previous record.
func TestCreateMovesOpenReadoutsToTheNewRecord(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)

	app.ShowPage("Token")
	app.token.info = &TokenRow{ID: "old"}
	app.token.balanceAccount = "owner"
	app.token.historyOwner = "owner"
	app.token.balance = "1"
	app.token.history = []TransferRow{{Amount: "1"}}
	app.token.name, app.token.symbol, app.token.supply, app.token.owner = "New", "N", "10", "pk"
	app.createToken()
	require.Equal(t, "t1", app.token.tokenID)
	require.Equal(t, "t1", fake.lastTokenInfoID, "the Inspect card must move to the new token")
	require.Equal(t, "t1", fake.lastBalanceToken, "a shown balance must move to the new token")
	require.Equal(t, "t1", fake.lastHistoryToken, "a loaded history must move to the new token")

	app.ShowPage("NFT")
	app.nft.nft = &NFTRow{ID: "old"}
	app.nft.history = []NFTOpRow{{Type: "mint"}}
	app.nft.name, app.nft.creator = "New", "pk"
	app.mintNFT()
	require.Equal(t, "n1", app.nft.id)
	require.Equal(t, "n1", fake.lastNFTID, "the Inspect card must move to the new asset")
	require.Equal(t, "n1", fake.lastNFTHistoryID, "a loaded history must move to the new asset")

	app.ShowPage("Lottery")
	app.lottery.verifyResult = &VerifyResult{ID: "old", Valid: true}
	app.lottery.participants, app.lottery.seed, app.lottery.count = "Alice,Bob", "seed", 1
	app.createLottery()
	require.Nil(t, app.lottery.verifyResult, "a new draw must not inherit the previous verdict")
}
