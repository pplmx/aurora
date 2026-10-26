package nativeui

import (
	"runtime"
	"strings"
	"time"

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
	// actions are the application-level callbacks the File menu and Settings
	// page run: file operations, language and build facts. They are optional
	// so a view test can render without a shell.
	actions Actions
	// rememberKeys is an explicit, session-only choice. savedKeys maps a
	// base64 public key to the matching private key; it is never written to
	// disk and is cleared when the process ends or the user forgets it.
	rememberKeys bool
	savedKeys    map[string]string
	// guides holds each page's "How this works" disclosure. They start closed
	// so the page stays task-first, but the explanation is one click away.
	guides guideState
	// scroll is the page body's position. Every page uses the same container,
	// and setPage resets it so a new page starts at the top instead of
	// inheriting the previous page's offset.
	scroll ui.ScrollState

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
	// LedgerBlocks returns the payload of every recorded block, newest
	// first, so the page can show what the chain holds rather than only how
	// much of it there is.
	LedgerBlocks(limit int) ([]BlockRow, error)

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

	// GenerateKeyPair makes an Ed25519 key pair (base64 public and private) for a
	// form that needs an owner or creator key; the CLI generates one when the
	// user omits it.
	GenerateKeyPair() (publicKey, privateKey string, err error)

	// Token surface.
	CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error)
	TokenInfo(id string) (TokenRow, error)
	// ListTokens returns every stored token, newest first.
	ListTokens() ([]TokenRow, error)
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
	// ListAllNFTs returns every NFT, newest first. Without it the page could
	// only reach an asset whose owner key the operator already had.
	ListAllNFTs() ([]NFTRow, error)
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

// BlockRow is one audit block as the ledger page shows it.
type BlockRow struct {
	// Height is the block's position in the chain.
	Height int64
	// Module names the subsystem that recorded it, taken from the payload.
	Module string
	// Summary is the payload, shortened for a table cell.
	Summary string
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
	// The draw form opens ready to submit: one winner is the domain's own
	// default, and an empty count field rejected the first Create click with a
	// parse error for a value the operator never had a chance to see.
	a.lottery.count = 1
	a.token.decimals = 8
	a.token.historyLimit = 50
	a.oracle.queryLim = 20
	a.blockchain.blockLimit = 200
	// -1 is "no row chosen": 0 is a real row, so it cannot double as the
	// unset value or the first click on row 0 would look like no change.
	a.lottery.selected = -1
	a.token.chosen = -1
	a.nft.selected = -1
	a.oracle.selected = -1
	a.voting.sessionChoice = -1
	a.voting.candChoice = -1
	// The session form opens with a concrete window (now to a day later):
	// empty date and time controls are a rejected first attempt the operator
	// would have to fix before doing anything.
	a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
	a.refresh()
	return a
}

// Services returns the application layer the window drives. It is how a test
// (and any other in-process surface) reaches the same calls a view makes,
// without a window or a click.
func (a *App) Services() Services { return a.services }

// View is the frame builder handed to mygo.WindowOptions.Content.
//
// The window is a sidebar of modules beside one scrolling page. The theme is
// set here rather than left to the system appearance: Aurora is dark by
// design, and a light system theme used to paint white controls onto the dark
// window.
func (a *App) View(c *ui.Context) {
	c.SetTheme(auroraTheme(c.Preferences().HighContrast))
	ui.Row(c).Fill().Children(func() {
		a.sidebarColumn(c)
		ui.Column(c).Fill().Grow(1).Padding(24).Gap(18).Children(func() {
			a.header(c)
			// The outcome of the last action lives in the fixed part of the
			// window, not at the top of the scrolling body: an action at the
			// bottom of a form could report a failure above the viewport, and
			// the operator would see the form unchanged and assume it worked.
			renderStatus(c, a.pageStatus())
			a.body(c)
		})
	})
}

// pageStatus is the transient outcome line of the current page.
func (a *App) pageStatus() statusLine {
	switch a.page {
	case pageLottery:
		return a.lottery.status
	case pageVoting:
		return a.voting.status
	case pageToken:
		return a.token.status
	case pageOracle:
		return a.oracle.status
	case pageNFT:
		return a.nft.status
	default:
		return statusLine{}
	}
}

// sidebarColumn is the chrome around the module list: the app's name, then the
// list itself on its own tone, separated from the content by a hairline. The
// reference apps (GoRex, the NetEase client) frame navigation this way; a bare
// list on the window background read as another column of content.
func (a *App) sidebarColumn(c *ui.Context) {
	ui.Column(c).Width(216).FillHeight().Background(ui.Hex(sidebarSurface)).
		BorderWidth(0, 1, 0, 0).BorderColor(ui.Hex(border)).Children(func() {
		a.brand(c)
		a.sidebar(c)
		ui.Spacer(c)
		a.sidebarActions(c)
	})
}

// brand is the window's identity at the top of the sidebar.
func (a *App) brand(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(18, 18, 10, 8).Gap(2).Children(func() {
		ui.Text(c, tr("Aurora")).FontSize(textSize(c, 19)).Bold().TextColor(ui.Hex(ink)).SingleLine()
		ui.Text(c, tr("Verifiable ledger suite")).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted)).SingleLine()
	})
}

// sidebar draws the module list as a real sidebar: a chosen item that the
// toolkit tracks, arrows and first-letter keys to move between modules, and
// one focus stop for the whole list. Every item carries its page glyph, so the
// column is scannable without reading all seven labels.
func (a *App) sidebar(c *ui.Context) {
	selected := pageIDs[a.page]
	ui.Sidebar(c, &selected, func() {
		ui.SidebarSection(c, tr("Modules"), nil, func() {
			for _, p := range moduleOrder {
				// The chosen item is already tinted by the toolkit; a marker
				// beside it would say the same thing twice.
				ui.SidebarItem(c, pageIDs[p], pageIcons[p], tr(pageTitles[p]))
			}
		})
		ui.SidebarSection(c, tr("Application"), nil, func() {
			ui.SidebarItem(c, pageIDs[pageSettings], pageIcons[pageSettings], tr(pageTitles[pageSettings]))
		})
	}).FillWidth()
	if p, ok := pageByID(selected); ok {
		a.setPage(p)
	}
}

// header names the page and gives the one action every page shares.
//
// Refresh carries the shortcut its platform uses for "reload": a page whose
// data is a read of the database is refreshed the way any other window is,
// without hunting for a button. Ctrl/Cmd+1..7 selects a page in sidebar order,
// the desktop convention the reference apps follow.
func (a *App) header(c *ui.Context) {
	a.actionShortcuts(c)
	if c.Shortcut(platformRefreshModifier(), ui.KeyComma) {
		a.setPage(pageSettings)
	}
	for i, p := range pageOrder {
		if c.Shortcut(platformRefreshModifier(), ui.Key(int(ui.Key1)+i)) {
			a.setPage(p)
		}
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Gap(12).AlignItems(ui.Center).Children(func() {
			pageGlyph(c, a.page)
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, tr(pageTitles[a.page])).FontSize(textSize(c, 26)).Bold().TextColor(ui.Hex(ink))
				ui.Text(c, tr(pagePurpose[a.page])).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
			})
		})
		// Ctrl on Linux and Windows, Cmd on macOS: the platform's own
		// modifier for a reload. The shortcut is the window's, not the
		// button's: an element's shortcut only fires while that element has
		// the keyboard focus, and a reload has to work from anywhere.
		if c.Shortcut(platformRefreshModifier(), ui.KeyR) || secondaryButton(c, "Refresh").Clicked() {
			a.refresh()
		}
	})
	ui.Divider(c)
}

// platformRefreshModifier is the modifier a desktop uses for "reload".
func platformRefreshModifier() ui.Modifiers {
	if runtime.GOOS == "darwin" {
		return ui.Super
	}
	return ui.Ctrl
}

// pagePurpose is the one line under a page title: what the module is for,
// stated once in the chrome instead of repeated by every section.
var pagePurpose = map[page]string{
	pageDashboard:  "What the ledger holds, and what happened recently",
	pageLottery:    "Run a draw, then re-check its proof",
	pageVoting:     "Register voters, open a session, cast signed ballots",
	pageToken:      "Create a token, then mint, send, approve or burn it",
	pageOracle:     "Connect an HTTP feed and read what it returns",
	pageNFT:        "Mint an asset, then transfer or burn it",
	pageBlockchain: "Every record the modules wrote, newest first",
	pageSettings:   "Language, data location, appearance and build information",
}

// scrollPage wraps a page body in the one tracked scroll container. Tracking
// it lets setPage reset the offset explicitly instead of relying on the old
// content happening to clamp it.
func (a *App) scrollPage(c *ui.Context, fn func()) {
	ui.Scroll(c).Fill().TrackScroll(&a.scroll).Children(fn)
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
	case pageSettings:
		a.settingsView(c)
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
		if len(a.voting.results) > 0 || a.voting.resultsErr != nil {
			id := strings.TrimSpace(a.voting.resultsSession)
			a.clearTallyForAnotherSession(id)
			_ = a.loadSessionResults(id)
		}
	case pageToken:
		a.refreshToken()
	case pageOracle:
		a.loadOracle()
		a.refreshOracleReads()
	case pageNFT:
		a.refreshNFTPage()
	case pageBlockchain:
		a.loadBlockchain()
	}
}

// Reload re-reads the current page, for a change made outside the view (an
// imported export, a forwarded launch). A caller that owns a window must also
// invalidate it: changing this state does not schedule a frame on its own.
func (a *App) Reload() { a.refresh() }

// ShowPage switches the window to the page a deep link names. An unknown or
// empty name leaves the current page alone: a malformed link must not blank
// the window.
func (a *App) ShowPage(name string) {
	// "index" is the old dashboard document; a link or a caller may still
	// spell the destination that way.
	if strings.EqualFold(name, "index") {
		a.setPage(pageDashboard)
		return
	}
	for p, id := range pageIDs {
		// The ids are the stable contract (sidebar choices, deep links); the
		// titles are shown text and may be translated.
		if strings.EqualFold(id, name) {
			a.setPage(p)
			return
		}
	}
	for p, title := range pageTitles {
		if strings.EqualFold(title, name) {
			a.setPage(p)
			return
		}
	}
}

// setPage switches the window to p and loads its data, which is what every
// entry point (a sidebar click, a deep link) has to do: a page that only drew
// what was already loaded showed an empty list until the operator pressed
// Refresh.
func (a *App) setPage(p page) {
	if a.page == p {
		return
	}
	a.page = p
	a.scroll.X, a.scroll.Y = 0, 0
	a.refresh()
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
	a.setPage(pageLottery)
	a.lottery.verifyID = id
	a.verifyLottery()
}
