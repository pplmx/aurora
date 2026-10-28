package nativeui

import (
	"errors"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTokenCollectionPages covers the token collection's real limit/offset
// page: only the newest hundred rows are materialised, and Next reaches the
// remainder without leaving the old page's highlight on a different token.
func TestTokenCollectionPages(t *testing.T) {
	tokens := make([]TokenRow, 150)
	for i := range tokens {
		tokens[i] = TokenRow{
			ID: "t" + strconv.Itoa(i), Name: "Token " + strconv.Itoa(i),
			Symbol: "T", TotalSupply: "1",
		}
	}
	fake := &fakeServices{tokenRows: tokens}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()

	require.Len(t, app.token.tokens, tokenPageSize)
	require.Equal(t, 0, app.token.offset)
	require.Equal(t, tokenPageSize+1, fake.lastTokenListLimit,
		"one extra row distinguishes a full page from the last page")
	require.True(t, tester.HasText("Entries 1–100"))

	app.token.chosen = 0
	tester.Frame()
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, tokenPageSize, app.token.offset)
	require.Equal(t, -1, app.token.chosen, "a new page must not reuse the old row index")
	require.Len(t, app.token.tokens, 50)
	require.False(t, app.token.hasMore)
	require.True(t, tester.HasText("Entries 101–150"))
}

// TestTokenPagerStopsAtAnExactPageBoundary proves the extra-row probe fixes
// the common "exactly 100 rows" case: Next is disabled instead of opening an
// empty page and bouncing back to the beginning.
func TestTokenPagerStopsAtAnExactPageBoundary(t *testing.T) {
	tokens := make([]TokenRow, tokenPageSize)
	for i := range tokens {
		tokens[i] = TokenRow{ID: "t" + strconv.Itoa(i), Name: "Token", Symbol: "T"}
	}
	fake := &fakeServices{tokenRows: tokens}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()
	calls := fake.listTokensCalls

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.token.offset)
	require.Equal(t, calls, fake.listTokensCalls, "a disabled Next must not reload an empty page")
}

// TestNFTPagerStopsAtAnExactPageBoundary applies the same probe and disabled
// control to the NFT collection, which previously enabled Next on a full page.
func TestNFTPagerStopsAtAnExactPageBoundary(t *testing.T) {
	nfts := make([]NFTRow, nftPageSize)
	for i := range nfts {
		nfts[i] = NFTRow{ID: "n" + strconv.Itoa(i), Name: "NFT", Owner: "owner"}
	}
	fake := &fakeServices{nftRows: nfts}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("NFT")
	tester.Frame()
	calls := fake.listAllNFTsCalls

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.nft.offset)
	require.Equal(t, calls, fake.listAllNFTsCalls, "a disabled Next must not reload an empty page")
}

// TestFailedTokenPageMoveKeepsCurrentPage pins the failure path: a Next read
// that fails must leave both the rows and the pager offset on the page those
// rows came from, with the warning explaining why they did not move.
func TestFailedTokenPageMoveKeepsCurrentPage(t *testing.T) {
	tokens := make([]TokenRow, 150)
	for i := range tokens {
		tokens[i] = TokenRow{ID: "t" + strconv.Itoa(i), Name: "Token", Symbol: "T"}
	}
	fake := &fakeServices{tokenRows: tokens}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()
	app.token.chosen = 0
	fake.loadErrs.tokens = errors.New("read failed")

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.token.offset)
	require.Len(t, app.token.tokens, tokenPageSize)
	require.Equal(t, 0, app.token.chosen, "the old page's highlight must stay with its rows")
	require.True(t, tester.HasText("Failed to load tokens: read failed"))
	require.True(t, tester.HasText("Showing the last successful load."))
}

// TestFailedNFTPageMoveKeepsCurrentPage applies the same rule to the NFT
// collection, whose pager uses the same commit-on-success path.
func TestFailedNFTPageMoveKeepsCurrentPage(t *testing.T) {
	nfts := make([]NFTRow, 150)
	for i := range nfts {
		nfts[i] = NFTRow{ID: "n" + strconv.Itoa(i), Name: "NFT", Owner: "owner"}
	}
	fake := &fakeServices{nftRows: nfts}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("NFT")
	tester.Frame()
	app.nft.selected = 0
	fake.loadErrs.nfts = errors.New("read failed")

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.nft.offset)
	require.Len(t, app.nft.list, nftPageSize)
	require.Equal(t, 0, app.nft.selected, "the old page's highlight must stay with its rows")
	require.True(t, tester.HasText("Failed to load NFTs: read failed"))
	require.True(t, tester.HasText("Showing the last successful load."))
}

// TestVotingCandidateCollectionPages applies the same storage-backed pager to
// the candidate list. Registration can outpace the screen too, and a session
// roster needs every older candidate just as much as a token form needs an
// older id.
func TestVotingCandidateCollectionPages(t *testing.T) {
	candidates := make([]CandidateRow, 150)
	for i := range candidates {
		candidates[i] = CandidateRow{
			ID: "c" + strconv.Itoa(i), Name: "Candidate " + strconv.Itoa(i),
		}
	}
	fake := &fakeServices{candidates: candidates}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	require.Len(t, app.voting.candidates, votingPageSize)
	require.Equal(t, 0, app.voting.candOffset)
	require.Equal(t, votingPageSize+1, fake.lastCandidateLimit,
		"one extra row distinguishes a full page from the last page")
	require.True(t, tester.HasText("Entries 1–100"))

	app.voting.candChoice = 0
	tester.Frame()
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, votingPageSize, app.voting.candOffset)
	require.Equal(t, -1, app.voting.candChoice, "a new page must not reuse the old row index")
	require.Len(t, app.voting.candidates, 50)
	require.False(t, app.voting.candHasMore)
	require.True(t, tester.HasText("Entries 101–150"))
}

func TestVotingCandidatePagerStopsAtAnExactPageBoundary(t *testing.T) {
	candidates := make([]CandidateRow, votingPageSize)
	for i := range candidates {
		candidates[i] = CandidateRow{ID: "c" + strconv.Itoa(i), Name: "Candidate"}
	}
	fake := &fakeServices{candidates: candidates}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()
	calls := fake.candidateListCalls

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.voting.candOffset)
	require.Equal(t, calls, fake.candidateListCalls, "a disabled Next must not reload an empty page")
}

func TestFailedCandidatePageMoveKeepsCurrentPage(t *testing.T) {
	candidates := make([]CandidateRow, 150)
	for i := range candidates {
		candidates[i] = CandidateRow{ID: "c" + strconv.Itoa(i), Name: "Candidate"}
	}
	fake := &fakeServices{candidates: candidates}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()
	app.voting.candChoice = 0
	fake.loadErrs.candidates = errors.New("read failed")

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.voting.candOffset)
	require.Len(t, app.voting.candidates, votingPageSize)
	require.Equal(t, 0, app.voting.candChoice, "the old page's highlight must stay with its rows")
	require.True(t, tester.HasText("Failed to load voting data: read failed"))
	require.True(t, tester.HasText("Showing the last successful load."))
}

func TestVotingSessionCollectionPages(t *testing.T) {
	sessions := make([]SessionRow, 150)
	for i := range sessions {
		sessions[i] = SessionRow{
			ID: "s" + strconv.Itoa(i), Title: "Session " + strconv.Itoa(i),
			Status: "draft",
		}
	}
	fake := &fakeServices{sessions: sessions}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	require.Len(t, app.voting.sessions, votingPageSize)
	require.Equal(t, 0, app.voting.sessionOffset)
	require.Equal(t, votingPageSize+1, fake.lastSessionLimit)

	app.voting.sessionChoice = 0
	tester.Frame()
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, votingPageSize, app.voting.sessionOffset)
	require.Equal(t, -1, app.voting.sessionChoice, "a new page must not reuse the old row index")
	require.Len(t, app.voting.sessions, 50)
	require.False(t, app.voting.sessionHasMore)
	require.True(t, tester.HasText("Entries 101–150"))
}

func TestFailedSessionPageMoveKeepsCurrentPage(t *testing.T) {
	sessions := make([]SessionRow, 150)
	for i := range sessions {
		sessions[i] = SessionRow{ID: "s" + strconv.Itoa(i), Title: "Session", Status: "draft"}
	}
	fake := &fakeServices{sessions: sessions}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()
	app.voting.sessionChoice = 0
	fake.loadErrs.sessions = errors.New("read failed")

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 0, app.voting.sessionOffset)
	require.Len(t, app.voting.sessions, votingPageSize)
	require.Equal(t, 0, app.voting.sessionChoice, "the old page's highlight must stay with its rows")
	require.True(t, tester.HasText("Failed to load voting data: read failed"))
	require.True(t, tester.HasText("Showing the last successful load."))
}
