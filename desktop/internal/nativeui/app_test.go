package nativeui

import (
	"errors"
	"fmt"
	"math"
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
	historyN      int
	historyOffset int
	historyCalls  int

	voters     int
	voterRows  []VoterRow
	candidates []CandidateRow
	// candidateNames supplies roster names beyond the current candidate page.
	candidateNames map[string]string
	sessions       []SessionRow

	verifyResult VerifyResult
	verifyErr    error

	// createErr makes CreateDraw fail, so a test can see what a failed action
	// reports.
	createErr error

	// templates is what OracleTemplates returns.
	templates []TemplateRow

	// tokenRows is what ListTokens returns.
	tokenRows []TokenRow
	// tokenInfoRows supplies full metadata for an id that may be outside the
	// current list page. Ids without an entry still return the minimal row that
	// most selection tests need.
	tokenInfoRows map[string]TokenRow

	// nftRows is what ListAllNFTs returns.
	nftRows []NFTRow
	// nftByID supplies full metadata when Inspect names an asset outside the
	// current collection page.
	nftByID map[string]NFTRow
	// ownerNFTs is what ListNFTs returns for an owner filter.
	ownerNFTs []NFTRow
	// nftHistoryRows is the operation trail returned for an NFT.
	nftHistoryRows []NFTOpRow

	// sources is what OracleSources returns.
	sources []SourceRow
	// addSource overrides the add result for tests that need the new row to
	// enter the next source page or fail that refresh.
	addSource func(SourceInput) (SourceRow, error)

	// blocks is what LedgerBlocks returns.
	blocks []BlockRow

	// loadErrs fail one read each, so a test can prove a page keeps the
	// content of its last successful load when a refresh fails.
	loadErrs fakeLoadErrors

	// healthRows is what OracleHealth returns, so a refresh test can keep rows
	// across a failed health read.
	healthRows []HealthRow

	// Counters for the reads a successful write is expected to repeat, so a
	// test can prove the page refreshes what it shows after a write.
	listTokensCalls     int
	lastTokenListLimit  int
	lastTokenListOffset int
	voterListCalls      int
	lastVoterLimit      int
	lastVoterOffset     int
	candidateListCalls  int
	lastCandidateLimit  int
	lastCandidateOffset int
	candidateNameCalls  int
	lastRosterIDs       []string
	sessionListCalls    int
	lastSessionLimit    int
	lastSessionOffset   int
	startSessionCalls   int
	endSessionCalls     int
	sourceListCalls     int
	oracleSourceCalls   int
	lastSourceLimit     int
	lastSourceOffset    int
	dashboardCalls      int
	blockListCalls      int
	lastBlockSearch     string
	lastBlockLimit      int
	lastBlockOffset     int
	oracleHealthCalls   int
	tokenInfoCalls      int
	tokenBalanceCalls   int
	tokenHistoryCalls   int
	tokenAllowanceCalls int
	getNFTCalls         int

	// The last subject each read was asked for, so a selection can be proven
	// to re-read its own record rather than the previous one.
	lastTokenInfoID        string
	lastBalanceToken       string
	lastAllowanceToken     string
	lastHistoryToken       string
	lastBalanceOwner       string
	lastAllowanceOwner     string
	lastAllowanceSpender   string
	lastHistoryOwner       string
	lastHistoryOffset      int
	lastNFTID              string
	lastNFTHistoryID       string
	lastNFTHistoryLimit    int
	lastNFTHistoryOffset   int
	lastOracleDataSource   string
	lastOracleLatestSource string
	lastSessionResultsID   string
	listAllNFTsCalls       int
	nftHistoryCalls        int
	sessionResultsCalls    int
	oracleDataCalls        int
	oracleLatestCalls      int
	// resultsByID is what SessionResults returns for a session; an entry may
	// be absent (empty tally). sessionResultsErr fails every tally read, so a
	// test can prove what stays on screen after a failed refresh.
	resultsByID       map[string][]ResultRow
	sessionResultsErr error

	// The last list limits the forms sent.
	tokenHistoryLimit int
	oracleDataLimit   int
}

// fakeLoadErrors is one failure hook per read a page performs.
type fakeLoadErrors struct {
	history       error
	health        error
	templates     error
	blocks        error
	tokens        error
	nfts          error
	sources       error
	voters        error
	candidates    error
	sessionRoster error
	sessions      error
	chain         error
	stats         error
	// Per-readout failures: the list-level hooks above must not be the only
	// way to exercise a failed detail read.
	tokenInfo    error
	tokenBalance error
	tokenHistory error
	nftDetails   error
	nftHistory   error
	oracleData   error
	oracleLatest error
}

func (f *fakeServices) LotteryHistory(limit, offset int) ([]DrawRow, error) {
	f.historyCalls++
	f.historyN = limit
	f.historyOffset = offset
	if f.loadErrs.history != nil {
		return nil, f.loadErrs.history
	}
	if offset >= len(f.draws) {
		return nil, nil
	}
	rows := f.draws[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeServices) CreateDraw(participants []string, seed string, winners int) (DrawRow, error) {
	if f.createErr != nil {
		return DrawRow{}, f.createErr
	}
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
	f.dashboardCalls++
	if f.loadErrs.stats != nil {
		return DashboardStats{}, f.loadErrs.stats
	}
	return DashboardStats{Lotteries: 3, Votes: 9, Candidates: 2, Sessions: 1, Integrity: "verified", Oracle: 4}, nil
}

func (f *fakeServices) LedgerBlocks(limit int) ([]BlockRow, error) {
	page, err := f.LedgerBlocksPage("", limit, 0)
	return page.Rows, err
}

func (f *fakeServices) LedgerBlocksPage(search string, limit, offset int) (LedgerBlockPage, error) {
	f.blockListCalls++
	f.lastBlockSearch = search
	f.lastBlockLimit = limit
	f.lastBlockOffset = offset
	if f.loadErrs.blocks != nil {
		return LedgerBlockPage{}, f.loadErrs.blocks
	}
	matching := make([]BlockRow, 0, len(f.blocks))
	for _, block := range f.blocks {
		if containsFold(search, block.Module, block.Summary, fmt.Sprintf("%d", block.Height)) {
			matching = append(matching, block)
		}
	}
	if offset > len(matching) {
		return LedgerBlockPage{Total: len(matching)}, nil
	}
	rows := matching[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return LedgerBlockPage{Rows: rows, Total: len(matching)}, nil
}
func (f *fakeServices) LedgerStatus() (LedgerStatus, error) {
	if f.loadErrs.chain != nil {
		return LedgerStatus{}, f.loadErrs.chain
	}
	height := 5
	if len(f.blocks) > 0 {
		height = len(f.blocks)
	}
	return LedgerStatus{Valid: true, Height: height, Blocks: height + 1}, nil
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
func (f *fakeServices) SessionResults(sessionID string) ([]ResultRow, error) {
	f.sessionResultsCalls++
	f.lastSessionResultsID = sessionID
	if f.sessionResultsErr != nil {
		return nil, f.sessionResultsErr
	}
	return f.resultsByID[sessionID], nil
}
func (f *fakeServices) Voters(limit, offset int) ([]VoterRow, error) {
	f.voterListCalls++
	f.lastVoterLimit = limit
	f.lastVoterOffset = offset
	if f.loadErrs.voters != nil {
		return nil, f.loadErrs.voters
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.voterRows) {
		return nil, nil
	}
	rows := f.voterRows[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
func (f *fakeServices) Candidates(limit, offset int) ([]CandidateRow, error) {
	f.candidateListCalls++
	f.lastCandidateLimit = limit
	f.lastCandidateOffset = offset
	if f.loadErrs.candidates != nil {
		return nil, f.loadErrs.candidates
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.candidates) {
		return nil, nil
	}
	rows := f.candidates[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
func (f *fakeServices) CandidateNames(ids []string) (map[string]string, error) {
	f.candidateNameCalls++
	f.lastRosterIDs = append([]string(nil), ids...)
	if f.loadErrs.sessionRoster != nil {
		return nil, f.loadErrs.sessionRoster
	}
	names := make(map[string]string, len(ids))
	for _, candidate := range f.candidates {
		names[candidate.ID] = candidate.Name
	}
	for id, name := range f.candidateNames {
		names[id] = name
	}
	return names, nil
}
func (f *fakeServices) Sessions(limit, offset int) ([]SessionRow, error) {
	f.sessionListCalls++
	f.lastSessionLimit = limit
	f.lastSessionOffset = offset
	if f.loadErrs.sessions != nil {
		return nil, f.loadErrs.sessions
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.sessions) {
		return nil, nil
	}
	rows := f.sessions[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
func (f *fakeServices) StartSession(id string) error {
	f.startSessionCalls++
	return nil
}
func (f *fakeServices) EndSession(id string) error {
	f.endSessionCalls++
	return nil
}

func (f *fakeServices) GenerateKeyPair() (string, string, error) {
	return "pk-generated", "sk-generated", nil
}

func (f *fakeServices) CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error) {
	return TokenRow{ID: "t1", Name: name, Symbol: symbol, TotalSupply: supply, Decimals: decimals, Owner: owner}, nil
}
func (f *fakeServices) TokenInfo(id string) (TokenRow, error) {
	f.tokenInfoCalls++
	f.lastTokenInfoID = id
	if f.loadErrs.tokenInfo != nil {
		return TokenRow{}, f.loadErrs.tokenInfo
	}
	if info, ok := f.tokenInfoRows[id]; ok {
		return info, nil
	}
	return TokenRow{ID: id}, nil
}
func (f *fakeServices) ListTokens(limit, offset int) ([]TokenRow, error) {
	f.listTokensCalls++
	f.lastTokenListLimit = limit
	f.lastTokenListOffset = offset
	if f.loadErrs.tokens != nil {
		return nil, f.loadErrs.tokens
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.tokenRows) {
		return nil, nil
	}
	rows := f.tokenRows[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
func (f *fakeServices) TokenBalance(tokenID, owner string) (string, error) {
	f.tokenBalanceCalls++
	f.lastBalanceToken = tokenID
	f.lastBalanceOwner = owner
	if f.loadErrs.tokenBalance != nil {
		return "", f.loadErrs.tokenBalance
	}
	return "42", nil
}
func (f *fakeServices) MintToken(tokenID, to, amount, priv string) error                { return nil }
func (f *fakeServices) TransferToken(tokenID, from, to, amount, priv string) error      { return nil }
func (f *fakeServices) ApproveToken(tokenID, owner, spender, amount, priv string) error { return nil }
func (f *fakeServices) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	return nil
}
func (f *fakeServices) BurnToken(tokenID, from, amount, priv string) error { return nil }
func (f *fakeServices) TokenAllowance(tokenID, owner, spender string) (string, error) {
	f.tokenAllowanceCalls++
	f.lastAllowanceToken = tokenID
	f.lastAllowanceOwner = owner
	f.lastAllowanceSpender = spender
	return "10", nil
}
func (f *fakeServices) TokenHistory(tokenID, owner string, limit, offset int) ([]TransferRow, error) {
	f.tokenHistoryLimit = limit
	f.tokenHistoryCalls++
	f.lastHistoryToken = tokenID
	f.lastHistoryOwner = owner
	f.lastHistoryOffset = offset
	if f.loadErrs.tokenHistory != nil {
		return nil, f.loadErrs.tokenHistory
	}
	return nil, nil
}

func (f *fakeServices) OracleSources(limit, offset int) ([]SourceRow, error) {
	f.sourceListCalls++
	f.lastSourceLimit = limit
	f.lastSourceOffset = offset
	if f.loadErrs.sources != nil {
		return nil, f.loadErrs.sources
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.sources) {
		return nil, nil
	}
	rows := f.sources[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeServices) OracleSource(id string) (SourceRow, error) {
	f.oracleSourceCalls++
	for _, source := range f.sources {
		if source.ID == id {
			return source, nil
		}
	}
	return SourceRow{}, errors.New("source not found")
}

func (f *fakeServices) OracleTemplates() ([]TemplateRow, error) {
	if f.loadErrs.templates != nil {
		return nil, f.loadErrs.templates
	}
	return f.templates, nil
}
func (f *fakeServices) AddOracleSource(in SourceInput) (SourceRow, error) {
	if f.addSource != nil {
		return f.addSource(in)
	}
	return SourceRow{ID: "src"}, nil
}
func (f *fakeServices) DeleteOracleSource(id string) error                   { return nil }
func (f *fakeServices) SetOracleSourceEnabled(id string, enabled bool) error { return nil }
func (f *fakeServices) FetchOracleSource(id string) error                    { return nil }
func (f *fakeServices) OracleData(sourceID string, limit int) ([]DataRow, error) {
	f.oracleDataCalls++
	f.lastOracleDataSource = sourceID
	f.oracleDataLimit = limit
	if f.loadErrs.oracleData != nil {
		return nil, f.loadErrs.oracleData
	}
	return nil, nil
}
func (f *fakeServices) OracleLatest(sourceID string) (DataRow, error) {
	f.oracleLatestCalls++
	f.lastOracleLatestSource = sourceID
	if f.loadErrs.oracleLatest != nil {
		return DataRow{}, f.loadErrs.oracleLatest
	}
	return DataRow{}, nil
}
func (f *fakeServices) OracleHealth() ([]HealthRow, error) {
	f.oracleHealthCalls++
	if f.loadErrs.health != nil {
		return nil, f.loadErrs.health
	}
	return f.healthRows, nil
}

func (f *fakeServices) MintNFT(name, description, imageURL, tokenURI, creator string) (NFTRow, error) {
	return NFTRow{ID: "n1", Name: name, Owner: creator}, nil
}
func (f *fakeServices) GetNFT(id string) (NFTRow, error) {
	f.getNFTCalls++
	f.lastNFTID = id
	if f.loadErrs.nftDetails != nil {
		return NFTRow{}, f.loadErrs.nftDetails
	}
	if n, ok := f.nftByID[id]; ok {
		return n, nil
	}
	return NFTRow{ID: id}, nil
}
func (f *fakeServices) ListNFTsPage(owner string, limit, offset int) ([]NFTRow, error) {
	if f.loadErrs.nfts != nil {
		return nil, f.loadErrs.nfts
	}
	rows := f.ownerNFTs
	if owner == "" {
		rows = f.nftRows
	}
	if offset > len(rows) {
		return nil, nil
	}
	rows = rows[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeServices) ListAllNFTsPage(limit, offset int) ([]NFTRow, error) {
	f.listAllNFTsCalls++
	return f.ListNFTsPage("", limit, offset)
}

func (f *fakeServices) ListNFTs(owner string) ([]NFTRow, error) {
	if f.loadErrs.nfts != nil {
		return nil, f.loadErrs.nfts
	}
	return f.ownerNFTs, nil
}
func (f *fakeServices) ListAllNFTs() ([]NFTRow, error) {
	f.listAllNFTsCalls++
	if f.loadErrs.nfts != nil {
		return nil, f.loadErrs.nfts
	}
	return f.nftRows, nil
}
func (f *fakeServices) TransferNFT(id, from, to, priv string) error { return nil }
func (f *fakeServices) BurnNFT(id, owner, priv string) error        { return nil }
func (f *fakeServices) NFTHistory(id string, limit, offset int) ([]NFTOpRow, error) {
	f.nftHistoryCalls++
	f.lastNFTHistoryID = id
	f.lastNFTHistoryLimit = limit
	f.lastNFTHistoryOffset = offset
	if f.loadErrs.nftHistory != nil {
		return nil, f.loadErrs.nftHistory
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.nftHistoryRows) {
		return nil, nil
	}
	rows := f.nftHistoryRows[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

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
	app.lottery.count = 1
	app.createLottery()
	require.NotEmpty(t, app.lottery.participantsErr)
	require.Nil(t, fake.created, "an empty participant list must not reach the service")

	app.lottery.participants = "a,b"
	app.lottery.seed = "ab"
	app.createLottery()
	require.NotEmpty(t, app.lottery.seedErr)
	require.Nil(t, fake.created)

	app.lottery.seed = "abcdef"
	app.lottery.count = 5
	app.createLottery()
	require.NotEmpty(t, app.lottery.countErr)
	require.Nil(t, fake.created)

	app.lottery.count = 2
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

	want := fmt.Sprintf(tr("Seed (at least %d characters)"), lottery.MinSeedLength)
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
	theme := auroraTheme(false)
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

// TestHighContrastThemeStrengthensContrast pins the variant the window uses
// when the OS asks for increased contrast: the same dark identity, with the
// hairlines and secondary text pushed away from the surfaces.
func TestHighContrastThemeStrengthensContrast(t *testing.T) {
	normal := auroraTheme(false)
	high := auroraTheme(true)

	require.True(t, high.Dark, "high contrast stays within the dark design")
	for name, pair := range map[string][2]ui.Color{
		"text":   {normal.Text, high.Text},
		"muted":  {normal.TextMuted, high.TextMuted},
		"border": {normal.Border, high.Border},
		"focus":  {normal.Focus, high.Focus},
	} {
		require.Greaterf(t,
			contrastRatio(pair[1], high.Background),
			contrastRatio(pair[0], normal.Background),
			"%s must have more contrast in the high-contrast theme", name)
	}
}

// contrastRatio is the WCAG contrast ratio between two opaque colours.
func contrastRatio(a, b ui.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

// TestEveryPageRendersItsSectionsAsCards proves the layout did not drop a
// section while the pages were being restructured: each page's own headings
// must still be on screen, in a window tall enough to hold them.
func TestEveryPageRendersItsSectionsAsCards(t *testing.T) {
	want := map[page][]string{
		pageDashboard:  {"Dashboard", "Total Lotteries", "Ledger Integrity"},
		pageLottery:    {"Create Draw", "Verify a Draw", "History"},
		pageVoting:     {"Register Voter", "Register Candidate", "Create Session"},
		pageToken:      {"Create Token", "Inspect", "Mint", "Transfer", "Approve", "Burn", "Allowance", "History"},
		pageOracle:     {"Add Source", "Data Sources"},
		pageNFT:        {"Mint NFT", "Inspect", "Transfer", "Burn", "History"},
		pageBlockchain: {"Ledger"},
		pageSettings:   {"Language", "Session keys", "Data", "Appearance", "About Aurora"},
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
		if p == pageVoting {
			// Setup is the default stage; Ballot and Results are one tab away.
			require.NoError(t, tester.Click("Ballot"))
			tester.Frame()
			require.True(t, tester.HasText("Cast Vote"), "the Ballot stage must show Cast Vote")
			require.NoError(t, tester.Click("Results"))
			tester.Frame()
			require.True(t, tester.HasText("Results"), "the Results stage must show the tally card")
			app.voting.tab = 0
		}
		if p == pageOracle {
			require.NoError(t, tester.Click("Health"))
			tester.Frame()
			require.True(t, tester.HasText("Feed Health"), "the Health stage must show Feed Health")
			require.NoError(t, tester.Click("Readings"))
			tester.Frame()
			require.True(t, tester.HasText("Query Data"), "the Readings stage must show Query Data")
			require.True(t, tester.HasText("Latest Data"))
			app.oracle.tab = 0
		}
	}
}

// TestDrawFormOpensReady pins the winner-count default. newLotteryState set
// it, but nothing ever called it, so the field opened empty and the first
// Create click failed to parse a value the operator never saw.
func TestDrawFormOpensReady(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	require.Equal(t, float64(1), app.lottery.count)

	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	require.Empty(t, app.lottery.countErr, "the default count must be usable as-is")
	require.NotNil(t, fake.created, "the form must submit with only participants and seed typed")
}

// TestStatusLinesStateTheirOutcomeInWords pins that a status line is not
// colour-only. Both outcomes must say "Done" or "Failed", because a reader who
// cannot separate the two hues would otherwise have no signal at all.
func TestStatusLinesStateTheirOutcomeInWords(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Lottery")
	tester.Frame()

	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	tester.Frame()
	require.True(t, tester.HasText("Done"), "a successful action must say so")
	require.Contains(t, joinedTexts(tester), "recorded at block")

	// A rejected action takes the other word, not just the other colour. A
	// field-level rejection is drawn under its input; a service failure is the
	// status line's job.
	app.lottery.seed = "ab"
	app.createLottery()
	tester.Frame()
	require.True(t, tester.HasText("at least 3 characters"),
		"a rejected seed must be explained under the seed field")

	fake.createErr = errors.New("storage is unavailable")
	// A successful create clears the form, so put the roster back before
	// asking the service to fail.
	app.lottery.participants = "Alice,Bob"
	app.lottery.seed = "seed-abc"
	app.createLottery()
	tester.Frame()
	require.True(t, tester.HasText("Failed"), "a failed action must say so")
	require.Contains(t, joinedTexts(tester), "storage is unavailable")
}

// joinedTexts is every label the last frame drew, for substring checks.
func joinedTexts(tester *ui.Tester) string {
	out := ""
	for _, s := range tester.Texts() {
		out += s + "\n"
	}
	return out
}

// TestHistoryTableShowsEveryStoredDraw covers the table a draw's history
// renders: both rows must appear, with their winners, so a reader can pick an
// id to verify.
func TestHistoryTableShowsEveryStoredDraw(t *testing.T) {
	fake := &fakeServices{draws: []DrawRow{
		{ID: "aaaa1111", BlockHeight: 1, Participants: []string{"Alice", "Bob"}, Winners: []string{"Alice"}},
		{ID: "bbbb2222", BlockHeight: 2, Participants: []string{"Carol"}, Winners: []string{"Carol"}},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Lottery")
	tester.Frame()

	require.True(t, tester.HasText("aaaa1111"))
	require.True(t, tester.HasText("bbbb2222"))
	require.True(t, tester.HasText("Carol"), "a winner must be shown in its row")
	require.Equal(t, lotteryHistoryLimit+1, fake.historyN,
		"one lookahead row distinguishes a full page from the final page")
	require.Zero(t, fake.historyOffset)
}

// TestRefreshShortcutReloadsThePage pins the keyboard path to a refresh. The
// window's data is a read of the database, so the platform's reload shortcut
// must work without hunting for the button.
func TestRefreshShortcutReloadsThePage(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	tester.Frame()

	app.ShowPage("Lottery")
	tester.Frame()
	fake.historyN = 0

	tester.Key(platformRefreshModifier(), ui.KeyR)
	tester.Frame()
	require.Equal(t, lotteryHistoryLimit+1, fake.historyN,
		"the reload shortcut must ask the page for its data again")
}
