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
	// liveReadAt is when the last live tick actually completed a read. The
	// window refreshes on a timer, so without it a reader cannot tell a current
	// page from one that stopped updating: a stalled drainer, a busy write and a
	// healthy quiet system all look identical.
	liveReadAt time.Time
	// windowTitle is the last title pushed to the native window, so the frame can
	// skip a redundant platform call on every repaint.
	windowTitle string
	// badgeCount is the count last pushed to the application icon, and badgePushed
	// records that it has been pushed at all — zero is a real value to send, so
	// the zero value of the field cannot stand in for "not yet".
	badgeCount  int
	badgePushed bool
	// trouble tracks how long a page has been unreadable, so a notice is raised
	// for an outage that lasted rather than for a read that failed once.
	trouble troubleNotice
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
	// outboxState/outboxRead cache the audit backlog the sidebar footer draws.
	// The footer renders every frame, so the value is re-read on the live-refresh
	// tick instead of on each paint.
	outboxState OutboxState
	outboxRead  time.Time
	// paletteSearch caches the ledger search behind the palette. The panel
	// rebuilds every frame, so the query runs once per edit rather than once per
	// frame.
	paletteSearch paletteRecordSearch
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
	// busyLabel is the running write's label without translation. The busy bar
	// translates it per frame, but the completion notification is raised after
	// the write settles, when a.busy is already cleared.
	busyLabel string
	// origin is where the running background write began, so a result that
	// arrives after the reader moved can still be reported where they are.
	origin     writeOrigin
	invalidate func()
	// themeMode is the saved appearance choice. systemDark is refreshed by
	// the shell when the desktop changes while Follow system is selected.
	themeMode  ThemeMode
	systemDark bool
	// contentZoom is the window content's zoom, 1 for none. It is MyGo's own
	// content zoom, which lays the view out in a window 1/z the size and draws
	// it at the screen's resolution times z, so glyphs stay sharp at any ratio.
	// It is separate from the operating system's text size: that setting is
	// about text, this one is about how much of the page fits, and a reader
	// should not have to change a system preference to make a window bigger.
	// zoomNotice asks for the ratio to be said once, because a zoom applied to a
	// fixed-size window changes how much of the page fits, which otherwise
	// looks like content vanishing.
	contentZoom float32
	zoomNotice  bool
	// displayTextScale is the operating system's text size as the last frame
	// read it. The copied diagnostics names it, and diagnosticsText has no frame
	// context of its own: the view records the value each frame so the copy can
	// quote what the reader is actually looking at.
	displayTextScale float32

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
		themeMode: ThemeModeDark, systemDark: true, contentZoom: 1,
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
	// The native title carries the failure state as well as the page, so a reader
	// who has switched to another window sees that something needs them. Pushed
	// from here rather than from each of the seven places that record a load
	// error: a comparison against the last pushed value covers all of them,
	// including the ones a future page adds.
	a.syncWindowTitle()
	// The badge is its own signal: a second failing page leaves the title
	// unchanged (it is already prefixed), so riding on the title's change check
	// would freeze the count at one.
	a.syncBadgeCount()
	a.maybeNotifyTrouble(a.clockNow())
	paletteWasOpen := a.paletteOpen
	if a.pageAnnouncement != "" {
		c.Announce(a.pageAnnouncement)
		a.pageAnnouncement = ""
	}
	c.SetTheme(auroraThemeForState(c.Preferences(), a.themeMode, a.systemDark, a.page))
	// The content zoom is MyGo's own, not a transform: it lays the view out in a
	// window 1/z the size and draws it at the screen's resolution times z, so
	// every glyph stays sharp at the larger size. It is set before the tree is
	// built because a frame that changes it rebuilds at the new size.
	c.SetZoom(clampZoom(a.contentZoom))
	a.displayTextScale = c.Preferences().TextScale
	a.showZoomNotice(c)
	// The footer draws the audit backlog, and its facts are storage reads. They
	// are refreshed on the same slow tick the live pages use, and never while a
	// write owns the database.
	now := a.clockNow()
	if a.busy == "" {
		a.refreshOutbox(now)
	}
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
func (a *App) pageStatus() statusLine { return a.statusOf(a.page) }

// statusOf returns a page's outcome line, whether or not it is showing. A
// background write needs this for the page it started on: the result is written
// into that page's line even after the reader moved elsewhere, and the line is
// what tells the window whether there is an outcome to report at all.
func (a *App) statusOf(p page) statusLine {
	switch p {
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
	// A background write owns the database. The Refresh button, its shortcut and
	// the palette entry each gate on that, but navigation does not — a reader may
	// switch pages during a write, and doing so reads the new page. A read that
	// lands on a locked database is recorded as a page that could not be read,
	// which then shows in the sidebar, the window title and the icon: a transient
	// lock would be reported as a broken page. Guarding the one place every read
	// starts covers all of them, including a future caller that forgets.
	if a.readBlocked() {
		return
	}
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
