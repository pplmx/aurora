package nativeui

import (
	"testing"

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
	app.nft.historyID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint"}}

	collectionBefore, historyBefore := fake.listAllNFTsCalls, fake.nftHistoryCalls
	app.nft.from = "owner-a"
	app.nft.to = "owner-b"
	app.nft.priv = "priv"
	app.transferNFT()

	require.False(t, app.nft.status.isErr)
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

	require.Equal(t, "Ballot recorded.", app.voting.voteResult)
	require.Equal(t, before+1, fake.sessionResultsCalls,
		"the open tally for this session must be re-read after a ballot")
}
