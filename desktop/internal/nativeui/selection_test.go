package nativeui

import (
	"testing"
	"time"

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

// TestTokenSelectionPrefillsTheOwnerAndDropsOldSecrets pins the context switch:
// a token's owner public key is known from the record, so the write forms start
// with it instead of asking for the same key six times. The previous token's
// private keys and recipients are cleared, so a subject change cannot leave a
// secret or an address aimed at another token.
func TestTokenSelectionPrefillsTheOwnerAndDropsOldSecrets(t *testing.T) {
	const owner = "owner-public-key"
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", Owner: owner}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.mintPriv, app.token.xPriv, app.token.approvePriv = "p1", "p2", "p3"
	app.token.fromPriv, app.token.burnPriv = "p4", "p5"
	app.token.xTo, app.token.approveTo, app.token.fromTo, app.token.allowanceTo = "old-1", "old-2", "old-3", "old-4"
	require.NoError(t, tester.Click("t1"))
	tester.Frame()

	require.Equal(t, "t1", app.token.tokenID)
	for name, got := range map[string]string{
		"mint to":         app.token.mintTo,
		"transfer from":   app.token.xFrom,
		"approve owner":   app.token.approveOwner,
		"transfer-from":   app.token.fromOwner,
		"burn from":       app.token.burnFrom,
		"allowance owner": app.token.allowanceOwner,
		"balance account": app.token.balanceAccount,
		"history owner":   app.token.historyOwner,
	} {
		require.Equalf(t, owner, got, "%s must start from the token owner", name)
	}
	require.Empty(t, app.token.mintPriv)
	require.Empty(t, app.token.xPriv)
	require.Empty(t, app.token.approvePriv)
	require.Empty(t, app.token.fromPriv)
	require.Empty(t, app.token.burnPriv)
	require.Empty(t, app.token.xTo)
	require.Empty(t, app.token.approveTo)
	require.Empty(t, app.token.fromTo)
	require.Empty(t, app.token.allowanceTo)
}

// TestCreatingATokenPrefillsTheOwnerAndSelectsTheRow keeps the create path as
// clear as selection: the new token becomes the highlighted record and its
// owner fills the accounts, so the next action does not begin with six pastes.
func TestCreatingATokenPrefillsTheOwnerAndSelectsTheRow(t *testing.T) {
	const owner = "owner-public-key"
	fake := &fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", Owner: owner}}}
	app := New(fake)
	app.ShowPage("Token")

	app.token.name, app.token.symbol, app.token.supply, app.token.owner = "Gold", "GLD", "100", owner
	app.createToken()

	require.Equal(t, "t1", app.token.tokenID)
	require.Equal(t, "t1", app.token.ownerKeyTokenID)
	require.Equal(t, 0, app.token.chosen, "the new token must be the highlighted row")
	require.Equal(t, owner, app.token.mintTo)
	require.Equal(t, owner, app.token.burnFrom)
	require.Equal(t, owner, app.token.balanceAccount)
}

// TestVotingCreatesSelectTheirNewRecord pins the same rule for the voting
// page: a new candidate or session becomes the highlighted row, the ballot
// target, and (for a session) the controls/tally context.
func TestVotingCreatesSelectTheirNewRecord(t *testing.T) {
	fake := &fakeServices{
		candidates: []CandidateRow{{ID: "c1", Name: "Alice"}},
		sessions:   []SessionRow{{ID: "s1", Title: "Budget"}},
	}
	app := New(fake)
	app.ShowPage("Voting")

	app.voting.candSearch, app.voting.candApplied = "old", "old"
	app.voting.candName = "Alice"
	app.registerCandidate()
	require.Equal(t, 0, app.voting.candChoice, "the new candidate must be highlighted")
	require.Equal(t, "c1", app.voting.voteCandidate, "the ballot must aim at the new candidate")
	require.Empty(t, app.voting.candSearch, "a search hiding the new candidate must be cleared")

	app.voting.sessionSearch, app.voting.sessionApplied = "old", "old"
	app.voting.sessionTitle = "Budget"
	app.voting.sessionStart, app.voting.sessionEnd = time.Unix(1000, 0), time.Unix(2000, 0)
	app.createVotingSession()
	require.Equal(t, 0, app.voting.sessionChoice, "the new session must be highlighted")
	require.Equal(t, "s1", app.voting.controlID)
	require.Equal(t, "s1", app.voting.voteSession)
	require.Equal(t, "s1", app.voting.resultsSession)
	require.Empty(t, app.voting.sessionSearch)
}

// TestCreatingADrawSelectsItInTheHistory pins the lottery create path: the new
// draw is highlighted and a search that would hide it is cleared.
func TestCreatingADrawSelectsItInTheHistory(t *testing.T) {
	fake := &fakeServices{draws: []DrawRow{{ID: "draw-1", Winners: []string{"Alice"}}}}
	app := New(fake)
	app.ShowPage("Lottery")

	app.lottery.search, app.lottery.appliedSearch = "old", "old"
	app.lottery.participants, app.lottery.seed, app.lottery.count = "Alice,Bob", "seed", 1
	app.createLottery()

	require.Equal(t, "draw-1", app.lottery.verifyID)
	require.Equal(t, 0, app.lottery.selected, "the new draw must be the highlighted row")
	require.Empty(t, app.lottery.search)
}

// TestUseGeneratedOwnerKey pins the shortcut for the key the Create card just
// made: the operator should not have to copy it between two fields on the same
// page. It appears only while that key's token is the subject.
func TestUseGeneratedOwnerKey(t *testing.T) {
	const owner = "owner-public-key"
	app := New(&fakeServices{tokenRows: []TokenRow{{ID: "t1", Owner: owner}}})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.tokenID = "t1"
	app.token.ownerPriv = "generated-private-key"
	app.token.ownerKeyTokenID = "t1"
	tester.Frame()
	require.True(t, tester.HasText("Use generated owner key"))

	require.NoError(t, tester.Click("Use generated owner key"))
	tester.Frame()
	require.Equal(t, "generated-private-key", app.token.mintPriv)

	app.token.tokenID = "t2"
	tester.Frame()
	require.False(t, tester.HasText("Use generated owner key"),
		"the generated key must not be offered for another token")
}
