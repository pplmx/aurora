package nativeui

import (
	"fmt"
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
	// paletteOpen/paletteQuery drive the Ctrl/Cmd+K page switcher. It is a
	// second, keyboard-first entry into the same setPage path as the sidebar.
	paletteOpen    bool
	paletteQuery   string
	paletteApplied string
	paletteChoice  int
	paletteScroll  ui.ScrollState
	paletteReveal  bool
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

	lottery    lotteryState
	voting     votingState
	token      tokenState
	oracle     oracleState
	nft        nftState
	blockchain blockchainState
	dashboard  dashboardState
}

func New(services Services) *App {
	a := &App{services: services, page: pageDashboard, liveNow: time.Now}
	// The draw form opens ready to submit: one winner is the domain's own
	// default, and an empty count field rejected the first Create click with a
	// parse error for a value the operator never had a chance to see.
	a.lottery.count = 1
	a.token.decimals = 8
	a.token.historyLimit = 50
	a.nft.limit = nftPageSize
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
	a.voting.voterChoice = -1
	a.dashboard.recentChoice = -1
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
// set here rather than left to the system appearance: Aurora is dark by
// design, and a light system theme used to paint white controls onto the dark
// window.
func (a *App) View(c *ui.Context) {
	a.invalidate = c.Invalidate
	a.pollAsync(c)
	if a.pageAnnouncement != "" {
		c.Announce(a.pageAnnouncement)
		a.pageAnnouncement = ""
	}
	c.SetTheme(auroraThemeFor(c.Preferences()))
	ui.Row(c).Fill().Children(func() {
		a.sidebarColumn(c)
		ui.Column(c).Fill().Grow(1).Padding(24).Gap(18).Children(func() {
			a.header(c)
			// The outcome of the last action lives in the fixed part of the
			// window, not at the top of the scrolling body: an action at the
			// bottom of a form could report a failure above the viewport, and
			// the operator would see the form unchanged and assume it worked.
			renderStatus(c, a.page, a.pageStatus(), a.clearPageStatus)
			a.busyBar(c)
			// A write owns the page body, not the navigation: the sidebar stays
			// usable, while form edits and a second submit cannot race the
			// result. The busy bar and page header stay live.
			ui.Column(c).Fill().Grow(1).Disabled(a.busy != "").Children(func() {
				a.body(c)
			})
		})
	})
	if a.paletteOpen {
		dialog := ui.Modal(c, &a.paletteOpen, func() { a.commandPalette(c) })
		dialog.Label(tr("Jump to"))
	}
}

// paletteEntry is one row in the Ctrl/Cmd+K switcher: either a page or a
// command. The action is nil for pages; navigation and side effects share the
// same selection model so Enter never means two different things.
type paletteEntry struct {
	title  string
	detail string
	icon   *ui.SVG
	page   page
	action func()
}

// commandPalette lets a keyboard-first operator jump to a page or run a file
// action without leaving the current form. Dialogs keep their own confirmation
// model; the palette only invokes the same callbacks as the File menu.
func (a *App) commandPalette(c *ui.Context) {
	width, height := c.Size()
	panelWidth := min(520*textScale(c), width-100)
	panel := ui.Column(c).Width(panelWidth).MaxHeight(height-40).Padding(18).Gap(10).Radius(14).
		Background(c.Theme().Surface).BorderWidth(1).BorderColor(c.Theme().Border).
		Shadow(0, 18, 48, -8, ui.RGBA(0, 0, 0, 0.5)).Children(func() {
		heading(c, tr("Jump to"), 17, 2)
		submitted := ui.SearchField(c, &a.paletteQuery).Label(tr("Search pages")).AutoFocus().Submitted()
		queryChanged := a.paletteQuery != a.paletteApplied
		if queryChanged {
			a.paletteApplied = a.paletteQuery
			a.paletteChoice = 0
			a.paletteScroll.Y = 0
		}
		matches := a.paletteMatches()
		announceSearchResults(c, queryChanged, len(matches))
		if len(matches) == 0 {
			emptyNote(c, fmt.Sprintf(tr("No pages match “%s”."), strings.TrimSpace(a.paletteQuery)))
			return
		}
		if a.paletteChoice >= len(matches) {
			a.paletteChoice = len(matches) - 1
		}
		if a.paletteChoice < 0 {
			a.paletteChoice = 0
		}
		reveal := a.paletteReveal
		a.paletteReveal = false
		list := ui.Scroll(c).FillWidth().Gap(10).
			MaxHeight(paletteListHeight(c, len(matches))).
			TrackScroll(&a.paletteScroll)
		list.Children(func() {
			for i, entry := range matches {
				row, clicked := paletteRow(c, entry, i == a.paletteChoice)
				if i == a.paletteChoice && reveal {
					row.ScrollIntoView()
				}
				if clicked || (submitted && i == a.paletteChoice) {
					a.paletteOpen = false
					if entry.action != nil {
						entry.action()
					} else {
						a.setPage(entry.page)
					}
					return
				}
			}
		})
	})
	// Arrow keys move the highlight; Enter is handled by SearchField above.
	if panel.OverlayShortcut(0, ui.KeyDown) {
		a.paletteChoice++
		a.paletteReveal = true
		c.Invalidate()
	}
	if panel.OverlayShortcut(0, ui.KeyUp) {
		a.paletteChoice--
		a.paletteReveal = true
		c.Invalidate()
	}
}

func (a *App) paletteMatches() []paletteEntry {
	entries := make([]paletteEntry, 0, len(pageOrder)+4)
	for _, p := range pageOrder {
		entries = append(entries, paletteEntry{
			title: pageTitles[p], detail: pagePurpose[p], icon: pageIcons[p], page: p,
		})
	}
	if a.page != pageSettings {
		entries = append(entries, paletteEntry{
			title: "Refresh current page", detail: "Reload the data shown on this page",
			icon: refreshIcon, action: a.refresh,
		})
	}
	if a.actions.BackUpDatabase != nil {
		entries = append(entries, paletteEntry{
			title: "Back up database…", detail: "Create a WAL-complete snapshot",
			icon: backupIcon, action: a.actions.BackUpDatabase,
		})
	}
	if a.actions.ExportLotteries != nil {
		entries = append(entries, paletteEntry{
			title: "Export lotteries…", detail: "Write the portable lottery document",
			icon: exportIcon, action: a.actions.ExportLotteries,
		})
	}
	if a.actions.ImportLotteries != nil {
		entries = append(entries, paletteEntry{
			title: "Import lotteries…", detail: "Load a portable lottery document",
			icon: importIcon, action: a.actions.ImportLotteries,
		})
	}
	matches := entries[:0]
	for _, entry := range entries {
		// Search what the palette shows, not only the English source strings.
		// A Chinese window renders translated titles and purposes, so a query
		// copied from the screen must match the same text.
		if containsFold(a.paletteQuery,
			entry.title, tr(entry.title), entry.detail, tr(entry.detail)) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func paletteRow(c *ui.Context, entry paletteEntry, selected bool) (ui.Element, bool) {
	title := tr(entry.title)
	b := ui.ButtonBase(c).FillWidth().Height(paletteRowHeight(c)).PaddingX(10 * textScale(c)).Radius(8).
		AlignItems(ui.Center).Label(title).TextColor(c.Theme().Text).Selected(selected)
	fill := ui.Transparent
	if selected {
		fill = c.Theme().Selection
	}
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(fill)
	b.Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			size := 18 * textScale(c)
			ui.Icon(c, entry.icon).Size(size, size).TextColor(ui.Hex(aurora))
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, title).FontSize(textSize(c, 13)).Bold().TextColor(c.Theme().Text).SingleLine()
				ui.Text(c, tr(entry.detail)).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).SingleLine()
			})
		})
	})
	return b, b.Clicked()
}

func paletteRowHeight(c *ui.Context) float32 {
	return 48 * textScale(c)
}

func paletteListHeight(c *ui.Context, rows int) float32 {
	if rows <= 0 {
		return 0
	}
	_, height := c.Size()
	rowHeight := paletteRowHeight(c)
	contentHeight := float32(rows)*rowHeight + float32(rows-1)*10
	availableHeight := max(rowHeight, height-190)
	return min(contentHeight, availableHeight)
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

// sidebarColumn is the chrome around the module list: the app's name, then the
// list itself on its own tone, separated from the content by a hairline. The
// reference apps (GoRex, the NetEase client) frame navigation this way; a bare
// list on the window background read as another column of content.
func (a *App) sidebarColumn(c *ui.Context) {
	width := windowWidth(c)
	ui.Column(c).Width(sidebarWidth(c, width)).FillHeight().Background(ui.Hex(sidebarSurface)).
		BorderWidth(0, 1, 0, 0).BorderColor(c.Theme().Border).Children(func() {
		a.brand(c)
		a.sidebar(c)
		a.sidebarActions(c)
	})
}

// brand is the window's identity at the top of the sidebar.
func (a *App) brand(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(18, 18, 10, 8).Gap(2).Children(func() {
		ui.Text(c, tr("Aurora")).FontSize(textSize(c, 19)).Bold().TextColor(c.Theme().Text).SingleLine()
		subtitle := tr("Verifiable ledger suite")
		ui.Text(c, subtitle).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).
			SingleLine().Tooltip(subtitle)
	})
}

// sidebar draws the module list as a real sidebar: a chosen item that the
// toolkit tracks, arrows and first-letter keys to move between modules, and
// one focus stop for the whole list. Every item carries its page glyph, so the
// column is scannable without reading all seven labels.
func (a *App) sidebar(c *ui.Context) {
	selected := pageIDs[a.page]
	reveal := a.sidebarReveal
	ui.Sidebar(c, &selected, func() {
		ui.SidebarSection(c, tr("Modules"), nil, func() {
			for _, p := range moduleOrder {
				// The chosen item is already tinted by the toolkit; a marker
				// beside it would say the same thing twice.
				label := tr(pageTitles[p])
				item := ui.SidebarItem(c, pageIDs[p], pageIcons[p], label).Tooltip(label)
				if reveal && p == a.page {
					item.ScrollIntoView()
				}
			}
		})
		ui.SidebarSection(c, tr("Application"), nil, func() {
			label := tr(pageTitles[pageSettings])
			item := ui.SidebarItem(c, pageIDs[pageSettings], pageIcons[pageSettings], label).Tooltip(label)
			if reveal && a.page == pageSettings {
				item.ScrollIntoView()
			}
		})
	}).Label(tr("Navigation")).FillWidth().Grow(1).Shrink(1).TrackScroll(&a.sidebarScroll)
	a.sidebarReveal = false
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
// liveRefreshInterval is deliberately slow: the dashboard and Oracle health
// are snapshots, and a background scheduler does not need a busy screen. A
// live choice is opt-in on Settings, so no page starts a timer on its own.
const liveRefreshInterval = 10 * time.Second

// liveRefresh schedules the next frame while live updates are on and runs the
// page's reload when it is due. It is called at the end of a page's view, so a
// reload invalidates the frame and the next one shows the new data. Forms keep
// their text; callers choose reloads that only read lists and counters.
func (a *App) liveRefresh(c *ui.Context, reload func()) {
	if !a.liveUpdates || reload == nil {
		return
	}
	now := time.Now()
	if a.liveNow != nil {
		now = a.liveNow()
	}
	if a.liveNext.IsZero() {
		a.liveNext = now.Add(liveRefreshInterval)
	}
	if !now.Before(a.liveNext) {
		a.liveNext = now.Add(liveRefreshInterval)
		reload()
		c.Invalidate()
	}
	c.After(a.liveNext.Sub(now))
}

func (a *App) setLiveUpdates(on bool) {
	a.liveUpdates = on
	a.liveNext = time.Time{}
}

func (a *App) header(c *ui.Context) {
	a.actionShortcuts(c)
	if !a.paletteOpen && c.Shortcut(platformRefreshModifier(), ui.KeyK) {
		a.paletteQuery = ""
		a.paletteApplied = ""
		a.paletteChoice = 0
		a.paletteScroll.Y = 0
		a.paletteReveal = false
		a.paletteOpen = true
	}
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
				heading(c, tr(pageTitles[a.page]), 26, 1)
				ui.Text(c, tr(pagePurpose[a.page])).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			})
		})
		// Ctrl on Linux and Windows, Cmd on macOS: the platform's own
		// modifier for a reload. The shortcut is the window's, not the
		// button's: an element's shortcut only fires while that element has
		// the keyboard focus, and a reload has to work from anywhere.
		// Settings reads no database, so a Refresh button there would be a
		// control with no effect; keep the action on data pages.
		if a.liveUpdates && (a.page == pageDashboard || a.page == pageOracle) {
			badge(c, "Live · 10s", ui.Hex(successTint), ui.Hex(auroraGreen))
		}
		if a.page != pageSettings {
			refresh := secondaryButton(c, "Refresh").Disabled(a.busy != "")
			if a.busy == "" &&
				(c.Shortcut(platformRefreshModifier(), ui.KeyR) || refresh.Clicked()) {
				a.refresh()
			}
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
	// Each page has its own content tree and scroll state. Give the scroll
	// widget a page identity so MyGo does not reconcile one page's controls
	// into the next page's tree when the two have different shapes. Without
	// this, Oracle -> NFT reused an incompatible layout and clipped the Mint
	// button at the bottom of its card.
	c.Key(a.page)
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
	a.sidebarReveal = true
	a.pageAnnouncement = a.PageTitle()
	if a.actions.SetPageTitle != nil {
		a.actions.SetPageTitle(a.PageTitle())
	}
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

// shortcutChord renders a platform shortcut for the Settings reference. The
// keys are the ones header/actionShortcuts register.
func shortcutChord(key string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

// windowWidth returns the current window width, or zero before the first
// layout (where the sidebar uses its comfortable width).
func windowWidth(c *ui.Context) float32 {
	w, _ := c.Size()
	return w
}

// sidebarWidthFor keeps the navigation readable without taking a quarter of a
// narrow window. The labels need about 160px including padding; 184px is the
// compact step, 216px the regular one.
func sidebarWidthFor(width float32) float32 {
	if width > 0 && width < 1000 {
		return sidebarCompactWidth
	}
	return sidebarRegularWidth
}

const (
	sidebarCompactWidth  float32 = 184
	sidebarRegularWidth  float32 = 216
	sidebarMaxFraction   float32 = 0.38
	sidebarMeasureSlack  float32 = 2
	sidebarItemChrome    float32 = 60
	sidebarSectionChrome float32 = 36
	sidebarBrandChrome   float32 = 36
)

// sidebarWidth keeps the fixed navigation width at normal text size, then
// makes room for the actual translated labels as the system text setting
// grows. Doubling the type inside an unscaled 184px column cut every English
// module name to a prefix. The window fraction caps the growth so body content
// still has useful space in a narrow window; row tooltips remain the fallback
// for a translation longer than that cap.
func sidebarWidth(c *ui.Context, window float32) float32 {
	base := sidebarWidthFor(window)
	if textScale(c) <= 1 {
		return base
	}
	required := max(base, sidebarContentWidth(c))
	if window > 0 {
		limit := window * sidebarMaxFraction
		if limit < base {
			limit = base
		}
		required = min(required, limit)
	}
	return required
}

// sidebarContentWidth measures the widest text the fixed chrome must show:
// a module row, a section heading or either brand line. The constants are the
// horizontal padding, icon and gaps MyGo's Sidebar and this brand block add
// around the text.
func sidebarContentWidth(c *ui.Context) float32 {
	measure := func(text string, size float32, weight int) float32 {
		width, _ := c.MeasureText(0, ui.Span{Text: text, Size: size, Weight: weight})
		return width
	}
	font := c.Theme().FontSize
	item := float32(0)
	for _, p := range pageOrder {
		item = max(item, measure(tr(pageTitles[p]), font, 400))
	}
	section := max(
		measure(tr("Modules"), font-1, 600),
		measure(tr("Application"), font-1, 600),
	)
	brand := max(
		measure(tr("Aurora"), textSize(c, 19), 700),
		measure(tr("Verifiable ledger suite"), textSize(c, 11), 400),
	)
	return max(
		item+sidebarItemChrome,
		section+sidebarSectionChrome,
		brand+sidebarBrandChrome,
	) + sidebarMeasureSlack
}

// SetBackgroundWrites enables the asynchronous write path used by the real
// desktop. It is opt-in so view tests keep the synchronous contract.
func (a *App) SetBackgroundWrites(on bool) { a.backgroundWrites = on }

// runWrite runs a service write. In the desktop it leaves the frame thread
// first, so the busy banner is painted and the window stays responsive; the
// apply closure always runs on the frame thread. Tests keep the synchronous
// path and observe the same state immediately.
func (a *App) runWrite(label string, work func() func()) {
	if !a.backgroundWrites {
		work()()
		return
	}
	if a.asyncCh != nil {
		return // one write at a time; the banner names the running one
	}
	ch := make(chan func(), 1)
	a.asyncCh = ch
	a.busy = label
	a.busySince = time.Now()
	a.busyAnnouncement = tr(label)
	a.busySlowAnnounced = false
	if a.invalidate != nil {
		a.invalidate()
	}
	go func() {
		ch <- work()
	}()
}

// pollAsync applies a completed write and schedules the next poll while one
// runs. It is called at the top of View, on the frame thread.
func (a *App) pollAsync(c *ui.Context) {
	if a.asyncCh == nil {
		return
	}
	select {
	case apply := <-a.asyncCh:
		a.asyncCh = nil
		a.busy = ""
		a.busySince = time.Time{}
		apply()
	default:
		c.After(40 * time.Millisecond)
	}
}

// busyBar is the fixed feedback for a write that has left the frame thread.
// It sits beside the page status so it is visible from any scroll position.
func (a *App) busyBar(c *ui.Context) {
	if a.busy == "" {
		return
	}
	text := tr(a.busy)
	if !a.busySince.IsZero() && time.Since(a.busySince) > 20*time.Second {
		text = tr("Still working…") + " " + tr("This can take a while; the operation will finish on its own.")
		if !a.busySlowAnnounced {
			a.busyAnnouncement, a.busySlowAnnounced = text, true
		}
	}
	bar := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Padding(10).Radius(8).
		Background(c.Theme().SurfaceHover).BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		ui.Spinner(c)
		ui.Text(c, text).FontSize(textSize(c, 12)).TextColor(c.Theme().Text)
	})
	bar.Role(ui.RoleStatus).Label(text)
	if a.busyAnnouncement != "" {
		c.Announce(a.busyAnnouncement)
		a.busyAnnouncement = ""
	}
}
