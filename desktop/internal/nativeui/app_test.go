package nativeui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

// fakeServices is a Services implementation for view tests: it returns fixed
// data and records what the view asked for, so a test can assert both what
// rendered and what the view called.
type fakeServices struct {
	draws   []DrawRow
	created *struct {
		participants []string
		seed         string
		winners      int
	}
	historyN int

	voters     int
	candidates []CandidateRow
	sessions   []SessionRow

	verifyResult VerifyResult
	verifyErr    error
}

func (f *fakeServices) LotteryHistory(limit int) ([]DrawRow, error) {
	f.historyN = limit
	return f.draws, nil
}

func (f *fakeServices) CreateDraw(participants []string, seed string, winners int) (DrawRow, error) {
	f.created = &struct {
		participants []string
		seed         string
		winners      int
	}{participants, seed, winners}
	return DrawRow{ID: "draw-1", BlockHeight: 7, Participants: participants, Winners: participants[:winners]}, nil
}

func (f *fakeServices) VerifyDraw(id string) (VerifyResult, error) {
	return f.verifyResult, f.verifyErr
}

func (f *fakeServices) Dashboard() (DashboardStats, error) {
	return DashboardStats{Lotteries: 3, Votes: 9, Candidates: 2, Sessions: 1, Integrity: "verified", Oracle: 4}, nil
}

func (f *fakeServices) LedgerStatus() (LedgerStatus, error) {
	return LedgerStatus{Valid: true, Height: 5, Blocks: 6}, nil
}

func (f *fakeServices) RegisterVoter(name string) (VoterRow, error) {
	f.voters++
	return VoterRow{PublicKey: "pk-" + name, Name: name}, nil
}
func (f *fakeServices) RegisterCandidate(name, party, program string) (CandidateRow, error) {
	return CandidateRow{ID: "c1", Name: name, Party: party, Program: program}, nil
}
func (f *fakeServices) CreateSession(title, description string, ids []string, start, end int64) (SessionRow, error) {
	return SessionRow{ID: "s1", Title: title, Status: "pending"}, nil
}
func (f *fakeServices) CastVote(sessionID, candidateID, voterPub, voterPriv string) error { return nil }
func (f *fakeServices) SessionResults(sessionID string) ([]ResultRow, error)              { return nil, nil }
func (f *fakeServices) Candidates() ([]CandidateRow, error)                               { return f.candidates, nil }
func (f *fakeServices) Sessions() ([]SessionRow, error)                                   { return f.sessions, nil }
func (f *fakeServices) StartSession(id string) error                                      { return nil }
func (f *fakeServices) EndSession(id string) error                                        { return nil }

func (f *fakeServices) CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error) {
	return TokenRow{ID: "t1", Name: name, Symbol: symbol, TotalSupply: supply, Decimals: decimals, Owner: owner}, nil
}
func (f *fakeServices) TokenInfo(id string) (TokenRow, error)                           { return TokenRow{ID: id}, nil }
func (f *fakeServices) TokenBalance(tokenID, owner string) (string, error)              { return "42", nil }
func (f *fakeServices) MintToken(tokenID, to, amount, priv string) error                { return nil }
func (f *fakeServices) TransferToken(tokenID, from, to, amount, priv string) error      { return nil }
func (f *fakeServices) ApproveToken(tokenID, owner, spender, amount, priv string) error { return nil }
func (f *fakeServices) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	return nil
}
func (f *fakeServices) BurnToken(tokenID, from, amount, priv string) error { return nil }
func (f *fakeServices) TokenAllowance(tokenID, owner, spender string) (string, error) {
	return "10", nil
}
func (f *fakeServices) TokenHistory(tokenID, owner string, limit, offset int) ([]TransferRow, error) {
	return nil, nil
}

func (f *fakeServices) OracleSources() ([]SourceRow, error)     { return nil, nil }
func (f *fakeServices) OracleTemplates() ([]TemplateRow, error) { return nil, nil }
func (f *fakeServices) AddOracleSource(in SourceInput) (SourceRow, error) {
	return SourceRow{ID: "src"}, nil
}
func (f *fakeServices) DeleteOracleSource(id string) error                       { return nil }
func (f *fakeServices) SetOracleSourceEnabled(id string, enabled bool) error     { return nil }
func (f *fakeServices) FetchOracleSource(id string) error                        { return nil }
func (f *fakeServices) OracleData(sourceID string, limit int) ([]DataRow, error) { return nil, nil }
func (f *fakeServices) OracleLatest(sourceID string) (DataRow, error)            { return DataRow{}, nil }
func (f *fakeServices) OracleHealth() ([]HealthRow, error)                       { return nil, nil }

func (f *fakeServices) MintNFT(name, description, imageURL, tokenURI, creator string) (NFTRow, error) {
	return NFTRow{ID: "n1", Name: name, Owner: creator}, nil
}
func (f *fakeServices) GetNFT(id string) (NFTRow, error)            { return NFTRow{ID: id}, nil }
func (f *fakeServices) ListNFTs(owner string) ([]NFTRow, error)     { return nil, nil }
func (f *fakeServices) TransferNFT(id, from, to, priv string) error { return nil }
func (f *fakeServices) BurnNFT(id, owner, priv string) error        { return nil }
func (f *fakeServices) NFTHistory(id string) ([]NFTOpRow, error)    { return nil, nil }

// TestAppRendersEveryPage drives the real view through MyGo's Tester, which
// builds frames without a window. It catches a panic in any page's layout and
// proves the sidebar switches pages, which is what a screenshot cannot do
// headlessly.
func TestAppRendersEveryPage(t *testing.T) {
	fake := &fakeServices{
		draws:      []DrawRow{{ID: "abc", BlockHeight: 1, Seed: "s", Participants: []string{"a", "b"}, Winners: []string{"a"}}},
		candidates: []CandidateRow{{ID: "c1", Name: "Alice"}},
		sessions:   []SessionRow{{ID: "s1", Title: "Election", Status: "active"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	for _, p := range pageOrder {
		app.ShowPage(pageTitles[p])
		require.Equal(t, p, app.page, "ShowPage must select %s", pageTitles[p])
		tester.Frame()
	}
}

// TestLotteryValidationStopsBadInput proves the field-level guards run before
// any service call, and that the error lands on the field that caused it.
func TestLotteryValidationStopsBadInput(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 860)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.participants = ""
	app.lottery.seed = "abc"
	app.lottery.count = "1"
	app.createLottery()
	require.NotEmpty(t, app.lottery.participantsErr)
	require.Nil(t, fake.created, "an empty participant list must not reach the service")

	app.lottery.participants = "a,b"
	app.lottery.seed = "ab"
	app.createLottery()
	require.NotEmpty(t, app.lottery.seedErr)
	require.Nil(t, fake.created)

	app.lottery.seed = "abcdef"
	app.lottery.count = "5"
	app.createLottery()
	require.NotEmpty(t, app.lottery.countErr)
	require.Nil(t, fake.created)

	app.lottery.count = "2"
	app.createLottery()
	require.Empty(t, app.lottery.countErr)
	require.NotNil(t, fake.created)
	require.Equal(t, 2, fake.created.winners)
}

// TestDeepLinkVerifiesTheDraw covers the aurora://lottery?id= promise: the link
// selects the Lottery page, fills the verifier and runs it.
func TestDeepLinkVerifiesTheDraw(t *testing.T) {
	fake := &fakeServices{verifyResult: VerifyResult{ID: "draw-9", Valid: true}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 860)

	app.ShowLotteryDraw("draw-9")
	require.Equal(t, pageLottery, app.page)
	require.NotNil(t, app.lottery.verifyResult)
	require.True(t, app.lottery.verifyResult.Valid)
	tester.Frame()
}

// TestSeedLabelFollowsTheDomainConstant proves the window shows the rule the
// domain enforces, not a copy of it. A hardcoded "at least 3 characters" would
// keep claiming 3 after MinSeedLength changed.
func TestSeedLabelFollowsTheDomainConstant(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	app.ShowPage("Lottery")
	tester.Frame()

	want := fmt.Sprintf("Seed (at least %d characters)", lottery.MinSeedLength)
	require.True(t, tester.HasText(want),
		"the create form must label the seed with the domain's minimum, want %q", want)
}

// TestShowIntentAppliesAPageAndADraw covers the single entry point the shell
// uses for every deep link. A cold start applies the launch argument through
// it, and so does an aurora:// link that reaches a running window; the two
// used to diverge, and a launch that named a draw opened the dashboard.
func TestShowIntentAppliesAPageAndADraw(t *testing.T) {
	fake := &fakeServices{verifyResult: VerifyResult{ID: "draw-7", Valid: true}}
	app := New(fake)

	app.ShowIntent("voting", "")
	require.Equal(t, pageVoting, app.page, "a link that names only a page must select it")

	app.ShowIntent("", "draw-7")
	require.Equal(t, pageLottery, app.page, "a link that names a draw must open the Lottery page")
	require.Equal(t, "draw-7", app.lottery.verifyID)
	require.NotNil(t, app.lottery.verifyResult, "the named draw must be verified immediately")

	app.ShowIntent("token", "draw-7")
	require.Equal(t, pageLottery, app.page, "the draw wins over the page named in the same link")

	// An unknown page must not blank the window.
	app.ShowIntent("no-such-page", "")
	require.Equal(t, pageLottery, app.page)
}

// TestWindowOpensWithTheDashboardLoaded pins the first frame. A view draws
// only what the state holds, and nothing else loaded the opening page: the
// window sat on "Loading…" until the operator switched pages or pressed
// Refresh, which a screenshot of a real launch caught.
func TestWindowOpensWithTheDashboardLoaded(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	require.False(t, tester.HasText("Loading"), "the opening frame must not be a placeholder")
	require.True(t, tester.HasText("Total Lotteries"), "the dashboard counters must be on screen")
	require.True(t, tester.HasText("9"), "the vote count must come from the services")
}

// TestAuroraThemeIsDarkAndDistinct pins the palette the window draws with. The
// build before this set a dark window background and then let MyGo draw its
// light theme on top, so every control was a white panel on a black page.
func TestAuroraThemeIsDarkAndDistinct(t *testing.T) {
	theme := auroraTheme()
	require.True(t, theme.Dark, "the window is dark by design, whatever the system appearance")
	require.Equal(t, ui.Hex(night), theme.Background)
	require.Equal(t, ui.Hex(surface), theme.Surface)
	require.Equal(t, ui.Hex(aurora), theme.Accent)
	require.Equal(t, ui.Hex(ink), theme.Text)

	// The surface has to be visible above the background, and the accent above
	// both: a theme whose three tones collapse into one is unreadable however
	// pleasant the individual colours are.
	require.NotEqual(t, theme.Background, theme.Surface)
	require.NotEqual(t, theme.Surface, theme.Accent)
	require.NotEqual(t, theme.Text, theme.Background)

	// A status line's two colours must differ, or a failure reads as success.
	require.NotEqual(t, ui.Hex(auroraRed), ui.Hex(auroraGreen))
}

// TestEveryPageRendersItsSectionsAsCards proves the layout did not drop a
// section while the pages were being restructured: each page's own headings
// must still be on screen, in a window tall enough to hold them.
func TestEveryPageRendersItsSectionsAsCards(t *testing.T) {
	want := map[page][]string{
		pageDashboard:  {"Dashboard", "Total Lotteries", "Ledger Integrity"},
		pageLottery:    {"Create Draw", "Verify a Draw", "History"},
		pageVoting:     {"Register Voter", "Register Candidate", "Create Session", "Cast Vote", "Results"},
		pageToken:      {"Create Token", "Inspect", "Mint", "Transfer", "Approve", "Burn", "Allowance", "History"},
		pageOracle:     {"Add Source", "Data Sources", "Feed Health", "Query Data", "Latest Data"},
		pageNFT:        {"Mint NFT", "Inspect", "Transfer", "Burn", "History"},
		pageBlockchain: {"Ledger"},
	}
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	// The sidebar must switch pages too: the page order and the ids the
	// Sidebar chooses by are two separate tables, and a mismatch left every
	// item selecting nothing.
	for _, p := range pageOrder {
		require.NoError(t, tester.Click(pageTitles[p]), "sidebar item %q", pageTitles[p])
		tester.Frame()
		require.Equal(t, p, app.page, "clicking %q must select that page", pageTitles[p])
	}
	for p, titles := range want {
		app.ShowPage(pageTitles[p])
		tester.Frame()
		for _, title := range titles {
			require.True(t, tester.HasText(title),
				"page %q must show the %q section", pageTitles[p], title)
		}
	}
}

// TestDrawFormOpensReady pins the winner-count default. newLotteryState set
// it, but nothing ever called it, so the field opened empty and the first
// Create click failed to parse a value the operator never saw.
func TestDrawFormOpensReady(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	require.Equal(t, "1", app.lottery.count)

	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	require.Empty(t, app.lottery.countErr, "the default count must be usable as-is")
	require.NotNil(t, fake.created, "the form must submit with only participants and seed typed")
}
