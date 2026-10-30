package nativeui

import (
	"runtime"
	"time"

	"github.com/egoist/mygo/ui"
)

// sidebarColumn is the chrome around the module list: the app's name, then the
// list itself on its own tone, separated from the content by a hairline. The
// reference apps (GoRex, the NetEase client) frame navigation this way; a bare
// list on the window background read as another column of content.
func (a *App) sidebarColumn(c *ui.Context) {
	width := windowWidth(c)
	ui.Column(c).Width(sidebarWidth(c, width)).FillHeight().LinearGradient(sidebarFill(c)).
		BorderWidth(0, 1, 0, 0).BorderColor(c.Theme().Border).Children(func() {
		a.brand(c)
		a.paletteButton(c)
		a.sidebar(c)
		a.sidebarActions(c)
	})
}

// brand is the window's identity at the top of the sidebar: a lit mark, the
// wordmark, and the one line that says what the product is. The mark is drawn
// from the two module hues rather than an image so it stays crisp at every
// scale and follows the window's own palette.
func (a *App) brand(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(space5, space5, space3, space1).Gap(space1).Children(func() {
		ui.Row(c).Gap(space3).AlignItems(ui.Center).Children(func() {
			mark := ui.Row(c).Size(iconTileSmall+4, iconTileSmall+4).Center().Radius(radiusSmall+2).
				Gradient(brandAccent(c), brandWarm(c), 135).
				Shadow(0, 10, 24, -8, brandAccent(c).Alpha(0.5)).
				InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.28))
			mark.Children(func() {
				ui.Icon(c, sparkIcon).Size(18, 18).TextColor(themeAccentText(c.Theme().Dark))
			})
			ui.Text(c, tr("Aurora")).FontSize(textSize(c, 20)).FontWeight(700).
				LetterSpacing(-0.2).TextColor(c.Theme().Text).SingleLine()
		})
		subtitle := tr("Verifiable ledger suite")
		ui.Text(c, subtitle).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).
			SingleLine().Tooltip(subtitle)
	})
}

// paletteButton makes the keyboard palette visible to a pointer user. The
// shortcut is printed on the control so Ctrl/Cmd+K is learned where the action
// lives, instead of only in the Settings reference.
func (a *App) paletteButton(c *ui.Context) {
	label := tr("Jump to")
	ui.Column(c).FillWidth().Padding(space3, space5, space1, 0).Children(func() {
		b := ui.ButtonBase(c).FillWidth().Height(max(34, textSize(c, 34))).
			PaddingX(space3).Radius(radiusSmall + 2).AlignItems(ui.Center).Label(label)
		fill, outline := c.Theme().Background, c.Theme().Border
		switch {
		case b.Pressed():
			fill = c.Theme().SurfacePressed
		case b.Hovered():
			fill = c.Theme().Surface
			outline = c.Theme().Accent.Alpha(0.48)
		}
		b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
		b.BorderWidth(1).BorderColor(outline)
		b.Children(func() {
			ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, searchIcon).Size(14, 14).TextColor(c.Theme().TextMuted)
				ui.Text(c, label).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
					Grow(1).Shrink(1).MinWidth(0).SingleLine()
				shortcutCap(c, shortcutChord("K"))
			})
		})
		if b.Clicked() {
			a.openPalette()
		}
	})
}

// sidebar draws the module list as a real sidebar: a chosen item that the
// toolkit tracks, arrows and first-letter keys to move between modules, and
// one focus stop for the whole list. Every item carries its page glyph, so the
// column is scannable without reading all seven labels.
func (a *App) sidebar(c *ui.Context) {
	selected := pageIDs[a.page]
	reveal := a.sidebarReveal
	var navigation ui.Element
	navigation = ui.Sidebar(c, &selected, func() {
		ui.SidebarSection(c, tr("Modules"), nil, func() {
			for _, p := range moduleOrder {
				// The chosen item is already tinted by the toolkit; a marker
				// beside it would say the same thing twice.
				label := tr(pageTitles[p])
				item := ui.SidebarItem(c, pageIDs[p], pageIcons[p], label).Tooltip(label)
				if p == a.page {
					markSidebarChoice(c, item, &navigation)
				}
				if reveal && p == a.page {
					item.ScrollIntoView()
				}
			}
		})
		ui.SidebarSection(c, tr("Application"), nil, func() {
			label := tr(pageTitles[pageSettings])
			item := ui.SidebarItem(c, pageIDs[pageSettings], pageIcons[pageSettings], label).Tooltip(label)
			if a.page == pageSettings {
				markSidebarChoice(c, item, &navigation)
			}
			if reveal && a.page == pageSettings {
				item.ScrollIntoView()
			}
		})
	}).Label(tr("Navigation")).FillWidth().Grow(1).Shrink(1).TrackScroll(&a.sidebarScroll)
	navigation.Bind(&a.navigationControl)
	a.sidebarReveal = false
	if p, ok := pageByID(selected); ok {
		a.setPage(p)
	}
}

// markSidebarChoice draws the chosen item's left accent. MyGo's sidebar fills
// the row with the accent when the list has keyboard focus and with a raised
// surface when it does not; the bar gives both states the same "this is the
// current page" mark and keeps the row recognisable when the fill is subtle.
func markSidebarChoice(c *ui.Context, item ui.Element, navigation *ui.Element) {
	item.DrawOver(func(p *ui.Painter, r ui.Rect) {
		tone := c.Theme().Accent
		if navigation.Focused() {
			// The focused row is filled with the accent, so the bar uses the
			// dark ink that reads on top of it.
			tone = c.Theme().AccentText
		}
		p.Fill(ui.Rect{X: r.X + 1, Y: r.Y + r.H*0.27, W: 3, H: r.H * 0.46}, tone, 1.5)
	})
}

// headerBand is the fixed chrome above the scrolling body: the page's name,
// its purpose and the one shared action, on a strip tinted with the module's
// own hue and closed by a hairline.
//
// The band is what makes the module's identity reach the page at all. Before
// it, the header was the same line of text on every page, with only the small
// tile to say which module was open; the tint is a few percent of the hue, so
// it changes the temperature of the strip without touching the contrast
// ratios text and borders depend on.
func (a *App) headerBand(c *ui.Context, fn func()) {
	tone := pageAccent(c, a.page)
	band := ui.Column(c).FillWidth().Padding(space5, space6, space5, space6)
	band.Draw(func(p *ui.Painter, r ui.Rect) {
		clear := tone.Alpha(0)
		p.FillGradient(r, ui.LinearGradient{
			Angle: 115,
			Stops: []ui.GradientStop{
				{Color: tone.Alpha(0.13), At: 0},
				{Color: tone.Alpha(0.045), At: 0.45},
				{Color: clear, At: 1},
			},
		}, 0)
	})
	band.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, c.Theme().Border, 0)
	})
	band.Children(fn)
}

// header names the page and gives the one action every page shares.
//
// Refresh carries the shortcut its platform uses for "reload": a page whose
// data is a read of the database is refreshed the way any other window is,
// without hunting for a button. Ctrl/Cmd+1..8 selects a page in sidebar order,
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
		a.openPalette()
	}
	if c.Shortcut(platformRefreshModifier(), ui.KeyComma) {
		a.navigateByKeyboard(c, pageSettings)
	}
	for i, p := range pageOrder {
		if c.Shortcut(platformRefreshModifier(), ui.Key(int(ui.Key1)+i)) {
			a.navigateByKeyboard(c, p)
		}
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(space4).Children(func() {
		pageGlyph(c, a.page)
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			heading(c, tr(pageTitles[a.page]), pageTitleSize, 1).LetterSpacing(-0.3)
			ui.Text(c, tr(pagePurpose[a.page])).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
		})
		// Ctrl on Linux and Windows, Cmd on macOS: the platform's own
		// modifier for a reload. The shortcut is the window's, not the
		// button's: an element's shortcut only fires while that element has
		// the keyboard focus, and a reload has to work from anywhere.
		// Settings reads no database, so a Refresh button there would be a
		// control with no effect; keep the action on data pages.
		if a.liveUpdates && (a.page == pageDashboard || a.page == pageOracle) {
			liveBadge(c)
		}
		if a.page != pageSettings {
			refresh := refreshButton(c).Disabled(a.busy != "")
			if a.busy == "" &&
				(c.Shortcut(platformRefreshModifier(), ui.KeyR) || refresh.Clicked()) {
				a.refresh()
			}
		}
	})
}

// liveBadge marks a page whose timer refreshes it in the background. The dot
// and the word carry the same state; the dot alone would be colour-only.
func liveBadge(c *ui.Context) {
	ui.Row(c).Gap(space2).AlignItems(ui.Center).Padding(space1, space2+2).Radius(999).
		Background(successTint(c)).BorderWidth(1).BorderColor(c.Theme().Success.Alpha(0.35)).
		Children(func() {
			statusDot(c, c.Theme().Success, 6)
			ui.Text(c, tr("Live · 10s")).FontSize(textSize(c, 11)).FontWeight(500).
				TextColor(c.Theme().Success).SingleLine()
		})
}

// refreshButton is the header's one shared action, drawn as a quiet glass
// control with its chord: the label says what it does, the chord says how to
// do it without the pointer.
func refreshButton(c *ui.Context) ui.Element {
	label := tr("Refresh")
	b := ui.ButtonBase(c).Height(max(buttonHeight, textSize(c, buttonHeight))).
		PaddingX(max(space3, textSize(c, 12))).Radius(radiusSmall + 2).
		AlignItems(ui.Center).Label(label)
	from, to := c.Theme().Surface, c.Theme().Surface.Mix(c.Theme().Background, 0.2)
	outline := c.Theme().Border
	switch {
	case b.Pressed():
		from, to = c.Theme().SurfacePressed, c.Theme().SurfaceHover
	case b.Hovered():
		from, to = c.Theme().SurfaceHover, c.Theme().Surface
		outline = c.Theme().Accent.Alpha(0.62)
	}
	b.Gradient(from, to, 180)
	b.BorderWidth(1).BorderColor(outline)
	b.Children(func() {
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, refreshIcon).Size(14, 14).TextColor(c.Theme().TextMuted)
			ui.Text(c, label).FontSize(textSize(c, 12)).FontWeight(600).SingleLine().TextColor(c.Theme().Text)
			shortcutCap(c, shortcutChord("R"))
		})
	})
	return b
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
	//
	// The gap is part of the page's rhythm: a card's rounded corner needs the
	// background between it and its neighbour, or two cards read as one slab
	// with a seam.
	c.Key(a.page)
	ui.Scroll(c).Fill().Gap(pageGap).TrackScroll(&a.scroll).Children(fn)
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
