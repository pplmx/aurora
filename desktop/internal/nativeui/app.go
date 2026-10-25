package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// App is the window's whole state. MyGo rebuilds a view from this state after
// every input, so it holds everything a frame reads: which page is showing and
// each page's form values. No view keeps its own copy.
//
// Services are the application layer the REST gateway also drives, so both
// surfaces write the same database.
type App struct {
	services Services
	page     page

	lottery    lotteryState
	voting     votingState
	token      tokenState
	oracle     oracleState
	nft        nftState
	blockchain blockchainState
	dashboard  dashboardState
}

// Services is the subset of the API server's wiring a view needs. The concrete
// implementation lives in internal/api (Server.Services) so the desktop module
// does not import handler internals.
type Services interface {
	// LotteryHistory returns the stored draws, newest first.
	LotteryHistory(limit int) ([]DrawRow, error)
	// CreateDraw runs a VRF draw and stores it.
	CreateDraw(participants []string, seed string, winners int) (DrawRow, error)
	// VerifyDraw re-checks a stored draw against its VRF proof.
	VerifyDraw(id string) (VerifyResult, error)

	// Dashboard returns the overview counters.
	Dashboard() (DashboardStats, error)
	// LedgerStatus verifies the audit chain.
	LedgerStatus() (LedgerStatus, error)

	// Voting surface.
	RegisterVoter(name string) (VoterRow, error)
	RegisterCandidate(name, party, program string) (CandidateRow, error)
	CreateSession(title, description string, candidateIDs []string, start, end int64) (SessionRow, error)
	CastVote(sessionID, candidateID, voterPub, voterPriv string) error
	SessionResults(sessionID string) ([]ResultRow, error)
	Candidates() ([]CandidateRow, error)
	Sessions() ([]SessionRow, error)
	StartSession(id string) error
	EndSession(id string) error

	// Token surface.
	CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error)
	TokenInfo(id string) (TokenRow, error)
	TokenBalance(tokenID, owner string) (string, error)
	MintToken(tokenID, to, amount, priv string) error
	TransferToken(tokenID, from, to, amount, priv string) error
	ApproveToken(tokenID, owner, spender, amount, priv string) error
	TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error
	BurnToken(tokenID, from, amount, priv string) error
	TokenAllowance(tokenID, owner, spender string) (string, error)
	TokenHistory(tokenID, owner string, limit, offset int) ([]TransferRow, error)

	// Oracle surface.
	OracleSources() ([]SourceRow, error)
	OracleTemplates() ([]TemplateRow, error)
	AddOracleSource(in SourceInput) (SourceRow, error)
	DeleteOracleSource(id string) error
	SetOracleSourceEnabled(id string, enabled bool) error
	FetchOracleSource(id string) error
	OracleData(sourceID string, limit int) ([]DataRow, error)
	OracleLatest(sourceID string) (DataRow, error)
	OracleHealth() ([]HealthRow, error)

	// NFT surface.
	MintNFT(name, description, imageURL, tokenURI, creator string) (NFTRow, error)
	GetNFT(id string) (NFTRow, error)
	ListNFTs(owner string) ([]NFTRow, error)
	TransferNFT(id, from, to, priv string) error
	BurnNFT(id, owner, priv string) error
	NFTHistory(id string) ([]NFTOpRow, error)
}

// DrawRow is one draw as a view shows it.
type DrawRow struct {
	ID           string
	BlockHeight  int64
	Seed         string
	Participants []string
	Winners      []string
	Timestamp    int64
}

// VerifyResult is the outcome of re-checking a draw.
type VerifyResult struct {
	ID      string
	Valid   bool
	Summary string
	Checks  []string
}

// New builds the window's state and loads the first page. A nil Services is a
// programming error, not a user-visible condition: the shell only constructs
// an App after it has wired the database.
//
// The first page is loaded here rather than on the first render because a view
// only draws what the state already holds: a window that opened on the
// dashboard and never switched pages showed "Loading…" forever, since nothing
// else asked the services for the counters.
func New(services Services) *App {
	a := &App{services: services, page: pageDashboard}
	a.refresh()
	return a
}

// View is the frame builder handed to mygo.WindowOptions.Content.
func (a *App) View(c *ui.Context) {
	ui.Row(c).Fill().Children(func() {
		a.sidebar(c)
		ui.Column(c).Fill().Grow(1).Padding(20).Gap(16).Children(func() {
			a.header(c)
			ui.Divider(c)
			a.body(c)
		})
	})
}

// sidebar draws the module list. It replaces the Web UI's horizontal nav: a
// desktop window has vertical room, and seven entries fit without scrolling.
func (a *App) sidebar(c *ui.Context) {
	ui.Column(c).Width(180).FillHeight().Padding(12).Gap(4).Children(func() {
		ui.Text(c, "Aurora").FontSize(18).Bold()
		ui.Text(c, "Ledger suite").FontSize(11).TextColor(ui.Hex("#6b7c7f"))
		ui.Divider(c)
		for _, p := range pageOrder {
			p := p
			label := pageTitles[p]
			if a.page == p {
				label = "▸ " + label
			}
			if ui.Button(c, label).Clicked() {
				a.page = p
				a.refresh()
			}
		}
	})
}

// header names the page and gives the one action every page shares.
func (a *App) header(c *ui.Context) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, pageTitles[a.page]).FontSize(20).Bold()
		})
		if ui.Button(c, "Refresh").Clicked() {
			a.refresh()
		}
	})
}

// body dispatches to the current page.
func (a *App) body(c *ui.Context) {
	switch a.page {
	case pageDashboard:
		a.dashboardView(c)
	case pageLottery:
		a.lotteryView(c)
	case pageVoting:
		a.votingView(c)
	case pageToken:
		a.tokenView(c)
	case pageOracle:
		a.oracleView(c)
	case pageNFT:
		a.nftView(c)
	case pageBlockchain:
		a.blockchainView(c)
	}
}

// refresh reloads the current page's data. Every page implements it so the
// header button and a page switch behave the same.
func (a *App) refresh() {
	switch a.page {
	case pageDashboard:
		a.loadDashboard()
	case pageLottery:
		a.loadLottery()
	case pageVoting:
		a.loadVoting()
	case pageToken:
		a.loadToken()
	case pageOracle:
		a.loadOracle()
	case pageNFT:
		a.loadNFT()
	case pageBlockchain:
		a.loadBlockchain()
	}
}

// ShowPage switches the window to the page a deep link names. An unknown or
// empty name leaves the current page alone: a malformed link must not blank
// the window.
func (a *App) ShowPage(name string) {
	for p, title := range pageTitles {
		if strings.EqualFold(title, name) {
			a.page = p
			a.refresh()
			return
		}
	}
}

// ShowIntent applies a deep link: it selects the page the link names and, when
// the link also names a draw, verifies that draw. An empty page leaves the
// current page alone, so a malformed link cannot blank the window.
//
// Both the operating system's requests and the launch arguments go through
// this one method, so a link behaves the same whether it started the process
// or reached a running one.
func (a *App) ShowIntent(page, lotteryID string) {
	if page != "" {
		a.ShowPage(page)
	}
	if lotteryID != "" {
		a.ShowLotteryDraw(lotteryID)
	}
}

// ShowLotteryDraw opens the Lottery page focused on one draw and verifies it
// immediately, which is what `aurora://lottery?id=<draw>` promises.
func (a *App) ShowLotteryDraw(id string) {
	if id == "" {
		return
	}
	a.page = pageLottery
	a.lottery.verifyID = id
	a.refresh()
	a.verifyLottery()
}
