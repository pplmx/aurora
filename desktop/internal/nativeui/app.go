package nativeui

import (
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
	// liveUpdates is an explicit choice to refresh the dashboard and Oracle
	// health while they are open. liveNext is when the next refresh is due;
	// liveNow is injectable so a view test can move the clock.
	liveUpdates bool
	liveNext    time.Time
	liveNow     func() time.Time
	// guides holds each page's "How this works" disclosure. They start closed
	// so the page stays task-first, but the explanation is one click away.
	guides guideState
	// scroll is the page body's position. Every page uses the same container,
	// and setPage resets it so a new page starts at the top instead of
	// inheriting the previous page's offset.
	scroll ui.ScrollState
	// sidebarScroll keeps the module list reachable when enlarged text makes it
	// taller than the window. sidebarReveal brings a newly chosen row into view
	// once, without preventing the operator from scrolling it away afterwards.
	sidebarScroll ui.ScrollState
	sidebarReveal bool
	// pageAnnouncement is the title waiting for the next frame to tell a
	// screen reader. setPage runs from pointer and keyboard handlers, where the
	// new page is not built yet; announcing it in View keeps navigation and
	// content in the same frame.
	pageAnnouncement string

	// settingsStatus is the result of a Settings action that can fail (saving
	// a preference); other pages' status lines stay with their forms.
	settingsStatus statusLine
	// confirmForgetKeys gates the one irreversible Settings action. When it
	// came from turning the policy off, confirmation must disable the policy as
	// well as clear the remembered keys; the explicit Forget button only clears
	// what is already held.
	confirmForgetKeys        bool
	disableKeyMemoryOnForget bool
	rememberKeysControl      ui.Handle
	forgetKeysButton         ui.Handle
	// paletteOpen/paletteQuery drive the Ctrl/Cmd+K page switcher. It is a
	// second, keyboard-first entry into the same setPage path as the sidebar.
	paletteOpen    bool
	paletteQuery   string
	paletteApplied string
	paletteChoice  int
	paletteScroll  ui.ScrollState
	paletteReveal  bool
	// navigationControl is the sidebar's one keyboard stop. Closing the
	// command palette moves focus here when its search field disappears, so a
	// keyboard user is never left at the window root.
	navigationControl ui.Handle
	// backgroundWrites is enabled by the real desktop. Tests keep the default
	// synchronous path so a handler's returned state is deterministic; the app
	// uses it to show a busy banner while a service call runs off the frame
	// thread.
	backgroundWrites  bool
	asyncCh           chan func()
	busy              string
	busySince         time.Time
	busyAnnouncement  string
	busySlowAnnounced bool
	invalidate        func()
	// themeMode is the saved appearance choice. systemDark is refreshed by
	// the shell when the desktop changes while Follow system is selected.
	themeMode  ThemeMode
	systemDark bool

	lottery    lotteryState
	voting     votingState
	token      tokenState
	oracle     oracleState
	nft        nftState
	blockchain blockchainState
	dashboard  dashboardState
}

func New(services Services) *App {
	a := &App{
		services: services, page: pageDashboard, liveNow: time.Now,
		themeMode: ThemeModeDark, systemDark: true,
	}
	// The draw form opens ready to submit: one winner is the domain's own
	// default, and an empty count field rejected the first Create click with a
	// parse error for a value the operator never had a chance to see.
	a.lottery.count = 1
	a.token.decimals = 8
	a.token.historyLimit = 50
	a.nft.limit = nftPageSize
	a.oracle.queryLim = 10
	a.blockchain.blockLimit = 200
	// -1 is "no row chosen": 0 is a real row, so it cannot double as the
	// unset value or the first click on row 0 would look like no change.
	a.lottery.selected = -1
	a.token.chosen = -1
	a.nft.selected = -1
	a.oracle.selected = -1
	a.blockchain.expanded = -1
	a.voting.sessionChoice = -1
	a.voting.candChoice = -1
	a.voting.voterChoice = -1
	// The session form opens with a concrete window (now to a day later):
	// empty date and time controls are a rejected first attempt the operator
	// would have to fix before doing anything.
	a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
	a.refresh()
	return a
}

// PageTitle is the visible page name in the current language, for the native
// window title and a language-change refresh.
func (a *App) PageTitle() string { return tr(pageTitles[a.page]) }

// Services returns the application layer the window drives. It is how a test
// (and any other in-process surface) reaches the same calls a view makes,
// without a window or a click.
func (a *App) Services() Services { return a.services }

// View is the frame builder handed to mygo.WindowOptions.Content.
//
// The window is a sidebar of modules beside one scrolling page. The theme is
// rebuilt for the saved appearance and the open module. Every module owns
// one hue, so section marks, tabs and the primary action all belong to the
// page instead of a global accent layer.
func (a *App) View(c *ui.Context) {
	a.invalidate = c.Invalidate
	a.pollAsync(c)
	paletteWasOpen := a.paletteOpen
	if a.pageAnnouncement != "" {
		c.Announce(a.pageAnnouncement)
		a.pageAnnouncement = ""
	}
	c.SetTheme(auroraThemeForState(c.Preferences(), a.themeMode, a.systemDark, a.page))
	ui.Row(c).Fill().Children(func() {
		a.sidebarColumn(c)
		workspace := ui.Column(c).Fill().Grow(1).LinearGradient(workspaceFill(c))
		workspace.Children(func() {
			// The header is a band, not a line of text: it carries the module's
			// own light and closes with a hairline, so the fixed chrome and the
			// scrolling body read as two surfaces instead of one long column.
			a.headerBand(c, func() {
				a.header(c)
			})
			ui.Column(c).Fill().Grow(1).Padding(space6).Gap(pageGap).Children(func() {
				// The outcome of the last action lives in the fixed part of the
				// window, not at the top of the scrolling body: an action at
				// the bottom of a form could report a failure above the
				// viewport, and the operator would see the form unchanged and
				// assume it worked.
				if renderStatus(c, a.page, a.pageStatus(), a.clearPageStatus) {
					a.navigationControl.Focus(c)
				}
				a.busyBar(c)
				// A write owns the page body, not the navigation: the sidebar
				// stays usable, while form edits and a second submit cannot
				// race the result. The busy bar and page header stay live.
				// The status and busy strips are conditional siblings above
				// this host. Without a stable identity, inserting one shifts
				// the body in the sibling list and MyGo drops focus from the
				// control whose action produced the message.
				ui.Column(c).Key("page-body").Fill().Grow(1).Disabled(a.busy != "").Children(func() {
					a.body(c)
				})
			})
		})
	})
	if a.paletteOpen {
		dialog := ui.Modal(c, &a.paletteOpen, func() { a.commandPalette(c) })
		dialog.Label(tr("Jump to"))
	}
	if paletteWasOpen && !a.paletteOpen {
		a.navigationControl.Focus(c)
	}
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

// clearPageStatus dismisses the current page's outcome line. The line is
// useful after a write, but a stale "Done" left on screen while the operator
// prepares the next action reads as a claim about the new state.
func (a *App) clearPageStatus() {
	switch a.page {
	case pageLottery:
		a.lottery.status = statusLine{}
	case pageVoting:
		a.voting.status = statusLine{}
	case pageToken:
		a.token.status = statusLine{}
	case pageOracle:
		a.oracle.status = statusLine{}
	case pageNFT:
		a.nft.status = statusLine{}
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
		if len(a.voting.results) > 0 || a.voting.resultsLoadedID != "" || a.voting.resultsErr != nil {
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
