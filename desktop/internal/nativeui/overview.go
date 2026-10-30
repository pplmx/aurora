package nativeui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// DashboardStats is the overview counter set the dashboard renders.
type DashboardStats struct {
	Lotteries  int
	Votes      int
	Candidates int
	Sessions   int
	Integrity  string
	Oracle     int
	// Blocks is the number of entries on the audit chain, including the
	// genesis block. It duplicates no module counter: token and NFT writes are
	// visible only here and on the Blockchain page.
	Blocks int
}

// dashboardState holds the overview counters plus the load outcome.
type dashboardState struct {
	DashboardStats
	loadErr error
	loaded  bool

	// recent is the newest handful of chain entries, shown under the counters
	// so the landing page answers "what happened lately" as well as "how much
	// is there". recentErr keeps the card's own failure separate from the
	// counters: a chain read failing must not blank the dashboard.
	recent    []BlockRow
	recentErr error
}

// dashboardRecentLimit is how many chain entries the dashboard shows. The
// Blockchain page has the full list; the overview only needs the latest few.
const dashboardRecentLimit = 5

// blockchainState holds the ledger verification result.
type blockchainState struct {
	valid   bool
	checked bool
	height  int
	blocks  int
	rows    []BlockRow
	loadErr error
	// blocksErr is the last failure to read the chain entries; the rows from
	// the previous successful read stay with a stale note.
	blocksErr error
	// search narrows the chain by module, payload or height, and blockLimit
	// bounds how many matching rows the table builds; the counters above it
	// still describe the whole chain.
	search      string
	blockLimit  float64
	blockOffset int
	blockTotal  int
	// queriedSearch/queriedLimit identify the block query the last attempt
	// used. They change before the read so a failed automatic query is not
	// retried on every repaint; blockOffset changes only after a successful
	// read, so a failed pager move leaves the label on the rows it shows.
	queriedSearch string
	queriedLimit  int
	// expanded is the index of the record whose exact payload is open, or -1
	// when every record shows its decoded sentence only.
	expanded int
}

func (a *App) loadDashboard() {
	if a.services == nil {
		return
	}
	stats, err := a.services.Dashboard()
	a.dashboard.loadErr = err
	if err != nil {
		return
	}
	a.dashboard.DashboardStats = stats
	a.dashboard.loaded = true

	// The activity card is secondary: its failure is reported in the card and
	// leaves the counters alone.
	blocks, blocksErr := a.services.LedgerBlocks(dashboardRecentLimit)
	a.dashboard.recentErr = blocksErr
	if blocksErr == nil {
		if len(blocks) > dashboardRecentLimit {
			blocks = blocks[:dashboardRecentLimit]
		}
		a.dashboard.recent = blocks
	}
}

func (a *App) dashboardView(c *ui.Context) {
	// A failed first load used to leave the spinner running forever, because
	// nothing else was ever going to ask for the counters.
	if a.dashboard.loadErr != nil && !loadWarning(c, "Couldn't load stats", a.dashboard.loadErr, a.dashboard.loaded) {
		return
	}
	if !a.dashboard.loaded {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, tr("Loading counters…")).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
		})
		return
	}

	a.scrollPage(c, func() {
		// The page has three reading levels: integrity is the verdict, the
		// module rails show the shape of the ledger, and the timeline is what
		// changed. A fresh store still starts with the first move, because an
		// empty composition has nothing to explain until the operator creates
		// its first record.
		if a.dashboardIsEmpty() {
			a.dashboardHero(c)
			a.dashboardQuickStart(c)
			a.dashboardComposition(c)
			a.dashboardRecent(c)
			return
		}
		a.dashboardHero(c)
		a.dashboardComposition(c)
		a.dashboardRecent(c)
		a.dashboardQuickStart(c)
	})
	a.liveRefresh(c, a.loadDashboard)
}

// dashboardHero is the page's verdict panel. It answers only the question that
// can be bad news: whether the stored chain verifies. The composition below
// owns every total, so the verdict does not repeat a block count or compete
// with the module rails for attention.
func (a *App) dashboardHero(c *ui.Context) {
	verified := a.dashboard.Integrity == "verified"
	tone := c.Theme().Success
	description := "Every module appends its records to one hash chain; each block commits to the previous hash."
	switch {
	case !verified:
		tone = c.Theme().Danger
		description = "The stored chain did not verify. Inspect the ledger before trusting the counters below."
	case a.dashboardIsEmpty():
		description = "The ledger is ready. Start with a quick action below; the first record opens block 1."
	}
	panel := ui.Column(c).FillWidth().Padding(space6, space5).Gap(space4).
		Radius(radiusLarge).LinearGradient(panelGradient(c, tone, 0.1)).
		BorderWidth(1).BorderColor(tone.Alpha(0.28)).
		Shadow(0, 28, 60, -28, ui.RGBA(0, 0, 0, 0.6)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.05))
	accentHairline(c, panel, tone)
	panel.Children(func() {
		verdict := func() {
			iconTile(c, shieldCheckIcon, tone, iconTileLarge)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(space1).Children(func() {
				eyebrow(c, "Ledger Integrity")
				ui.Row(c).Wrap().Gap(space3).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr(a.dashboard.Integrity)).FontSize(textSize(c, 26)).FontWeight(700).
						TextColor(tone).SingleLine()
					ui.Row(c).Padding(2, space2).Radius(999).Background(tone.Alpha(0.12)).
						BorderWidth(1).BorderColor(tone.Alpha(0.3)).Children(func() {
						ui.Text(c, tr("Hash chain")).FontSize(textSize(c, 10)).FontWeight(600).
							TextColor(tone).SingleLine()
					})
				})
				ui.Text(c, tr(description)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			})
		}
		openLedger := func() {
			if quietButton(c, "Open ledger").Clicked() {
				a.setPage(pageBlockchain)
			}
		}
		width, _ := c.Size()
		if width < 860*textScale(c) {
			ui.Column(c).FillWidth().Gap(space3).Children(func() {
				ui.Row(c).FillWidth().Gap(space4).AlignItems(ui.Center).Children(verdict)
				ui.Row(c).FillWidth().Justify(ui.End).Children(openLedger)
			})
			return
		}
		ui.Row(c).FillWidth().Gap(space4).AlignItems(ui.Center).Children(func() {
			verdict()
			ui.Spacer(c)
			openLedger()
		})
	})
}

// dashboardComposition is the only place on the page that reports totals. A
// ledger is not a list of unrelated counters: voting has two scales (people
// and sessions), the audit chain contains every module, and the
// lottery/oracle counts are smaller independent streams. The rail makes that
// shape visible before a single number is read.
func (a *App) dashboardComposition(c *ui.Context) {
	sectionRule(c, "Ledger composition")
	card(c, func() {
		sectionHeading(c, "What the ledger is made of",
			"Module records first; the audit chain carries all of them.")
		width, _ := c.Size()
		if width < twoColumnMinWidth*textScale(c) {
			ui.Column(c).FillWidth().Gap(space2).Children(func() {
				a.dashboardModuleRail(c, pageLottery, "Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded")
				a.dashboardModuleRail(c, pageVoting, "Voting", strconv.Itoa(a.dashboard.Votes+a.dashboard.Candidates+a.dashboard.Sessions), "Votes, candidates and sessions")
				a.dashboardModuleRail(c, pageOracle, "Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds")
				a.dashboardModuleRail(c, pageBlockchain, "Ledger Blocks", strconv.Itoa(a.dashboard.Blocks), "Entries on the audit chain")
			})
		} else {
			ui.Row(c).FillWidth().Gap(space2).Children(func() {
				ui.Column(c).Grow(1).Shrink(1).Gap(space2).Children(func() {
					a.dashboardModuleRail(c, pageLottery, "Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded")
					a.dashboardModuleRail(c, pageOracle, "Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds")
				})
				ui.Column(c).Grow(1).Shrink(1).Gap(space2).Children(func() {
					a.dashboardModuleRail(c, pageVoting, "Voting", strconv.Itoa(a.dashboard.Votes+a.dashboard.Candidates+a.dashboard.Sessions), "Votes, candidates and sessions")
					a.dashboardModuleRail(c, pageBlockchain, "Ledger Blocks", strconv.Itoa(a.dashboard.Blocks), "Entries on the audit chain")
				})
			})
		}
		ui.Column(c).FillWidth().Gap(space1).Children(func() {
			eyebrow(c, "Voting detail")
			ui.Row(c).FillWidth().Wrap().Gap(space4).Children(func() {
				a.dashboardModuleMetric(c, "Total Votes", strconv.Itoa(a.dashboard.Votes), "Signed ballots cast", pageVoting)
				a.dashboardModuleMetric(c, "Candidates", strconv.Itoa(a.dashboard.Candidates), "Registered contenders", pageVoting)
				a.dashboardModuleMetric(c, "Voting Sessions", strconv.Itoa(a.dashboard.Sessions), "Governance windows", pageVoting)
			})
		})
	})
}

// dashboardModuleRail is one module's share of the composition. The rail
// length is intentionally proportional to the module's visible count, with a
// floor so a zero still reads as a registered module rather than an error.
func (a *App) dashboardModuleRail(c *ui.Context, target page, title, value, note string) {
	tone := pageAccent(c, target)
	share := dashboardShare(target, a.dashboard.DashboardStats)
	b := ui.ButtonBase(c).FillWidth().Padding(space3).Gap(space2).
		Radius(radiusSmall + 2).AlignItems(ui.Start).Label(tr(title)).
		Description(tr(note)).Tooltip(tr(note))
	fill, outline := c.Theme().Background.Mix(c.Theme().Surface, 0.2), c.Theme().Border
	switch {
	case b.Pressed():
		fill = c.Theme().SurfacePressed
		outline = tone.Alpha(0.6)
	case b.Hovered():
		fill = c.Theme().SurfaceHover
		outline = tone.Alpha(0.6)
	}
	b.Background(b.AnimateColor("module-rail", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.BorderWidth(1).BorderColor(outline)
	b.Children(func() {
		ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Start).Children(func() {
			iconTile(c, pageIcons[target], tone, iconTileSmall)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(space1).Children(func() {
				ui.Text(c, tr(title)).FontSize(textSize(c, 10.5)).FontWeight(600).
					TextColor(c.Theme().TextMuted).SingleLine()
				ui.Text(c, value).FontSize(textSize(c, 22)).FontWeight(700).
					TextColor(tone).Font("monospace").FontFeatures("tnum").SingleLine()
				ui.Text(c, tr(note)).FontSize(textSize(c, 10)).TextColor(c.Theme().TextMuted)
			})
		})
		ui.Row(c).FillWidth().Height(3).Radius(1.5).Background(c.Theme().Border).Children(func() {
			ui.Row(c).WidthPercent(share).Height(3).Radius(1.5).Background(tone)
		})
	})
	if b.Clicked() {
		a.setPage(target)
	}
}

// dashboardModuleMetric is the second level of the composition: it gives the
// voting module's three stored scales their own labels without returning to a
// grid of equal cards.
func (a *App) dashboardModuleMetric(c *ui.Context, label, value, note string, target page) {
	b := ui.ButtonBase(c).Grow(1).Shrink(1).Padding(space2, space3).Radius(radiusSmall).
		AlignItems(ui.Start).Label(tr(label)).Description(tr(note))
	fill := c.Theme().Background.Mix(c.Theme().Surface, 0.18)
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(fill).BorderWidth(1).BorderColor(c.Theme().Border)
	b.Children(func() {
		ui.Column(c).Gap(1).Children(func() {
			eyebrow(c, label)
			ui.Text(c, value).FontSize(textSize(c, 18)).FontWeight(700).
				TextColor(pageAccent(c, target)).Font("monospace").FontFeatures("tnum").SingleLine()
			ui.Text(c, tr(note)).FontSize(textSize(c, 10)).TextColor(c.Theme().TextMuted).SingleLine()
		})
	})
	if b.Clicked() {
		a.setPage(target)
	}
}

// dashboardShare gives each module rail a visible proportion. The audit chain
// always has a floor because it contains the module records themselves; the
// other modules share the remaining space by their own counts. Voting's rail
// represents all three stored scales, so its sum participates in the
// denominator just like the other visible modules.
func dashboardShare(target page, d DashboardStats) float32 {
	const floor = 10
	largest := max(1, d.Lotteries, d.Votes+d.Candidates+d.Sessions, d.Oracle, d.Blocks)
	var value int
	switch target {
	case pageLottery:
		value = d.Lotteries
	case pageVoting:
		value = d.Votes + d.Candidates + d.Sessions
	case pageOracle:
		value = d.Oracle
	case pageBlockchain:
		value = d.Blocks
	}
	if value <= 0 {
		return floor
	}
	return min(100, float32(floor)+float32(value)/float32(largest)*90)
}

// dashboardIsEmpty distinguishes a fresh store from a partial one. Every
// visible counter participates, and a non-empty audit chain also counts even
// when the changed module (token, NFT) has no dashboard card. If that secondary
// read failed, emptiness is unknown rather than proven.
func (a *App) dashboardIsEmpty() bool {
	d := a.dashboard
	return d.recentErr == nil &&
		d.Lotteries == 0 &&
		d.Votes == 0 &&
		d.Candidates == 0 &&
		d.Sessions == 0 &&
		d.Oracle == 0 &&
		len(d.recent) == 0
}

// dashboardQuickStart names the first move in each module. The sidebar lists
// modules, not tasks; an operator opening a fresh window otherwise has to
// guess which page starts a draw, a token or a feed.
func (a *App) dashboardQuickStart(c *ui.Context) {
	formCard(c, "Quick start", "One first move per module; use the sidebar for everything else.", func() {
		if a.dashboardIsEmpty() {
			emptyNote(c, "Nothing recorded yet — run a draw to create the first ledger entry.")
		}
		ui.Row(c).FillWidth().Wrap().Gap(space2).Children(func() {
			a.quickAction(c, "Run a draw", pageLottery)
			a.quickAction(c, "Create a token", pageToken)
			a.quickAction(c, "Register a voter", pageVoting)
			a.quickAction(c, "Add a data source", pageOracle)
			a.quickAction(c, "Mint an NFT", pageNFT)
		})
	})
}

// quickAction is one first move drawn as a small tile: the module's glyph,
// the task, and the module it opens. Five buttons of equal weight read as a
// toolbar; five tiles with their module marks read as destinations.
func (a *App) quickAction(c *ui.Context, label string, target page) {
	tone := pageAccent(c, target)
	b := ui.ButtonBase(c).Width(textSize(c, 184)).Padding(space3).Gap(space2).
		Radius(radiusSmall + 2).AlignItems(ui.Start).Label(tr(label))
	fill, outline := c.Theme().Background.Mix(c.Theme().Surface, 0.35), c.Theme().Border
	switch {
	case b.Pressed():
		fill = c.Theme().SurfacePressed
		outline = tone.Alpha(0.45)
	case b.Hovered():
		fill = c.Theme().SurfaceHover
		outline = tone.Alpha(0.45)
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.BorderWidth(1).BorderColor(outline)
	hoverRaise(c, b, b.Hovered())
	b.Children(func() {
		ui.Column(c).FillWidth().Gap(space2).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				iconTile(c, pageIcons[target], tone, iconTileSmall)
				ui.Spacer(c)
				ui.Icon(c, arrowRightIcon).Size(13, 13).TextColor(c.Theme().TextMuted)
			})
			ui.Text(c, tr(label)).FontSize(textSize(c, 12.5)).FontWeight(600).
				TextColor(c.Theme().Text).SingleLine()
			ui.Text(c, tr(pageTitles[target])).FontSize(textSize(c, 10.5)).
				TextColor(c.Theme().TextMuted).SingleLine()
		})
	})
	if b.Clicked() {
		a.setPage(target)
	}
}

// dashboardRecent draws the newest chain entries. The composition says how
// much the store holds; this card says what happened, and every record opens
// the module that wrote it.
func (a *App) dashboardRecent(c *ui.Context) {
	title := tr("Recent activity")
	panel := card(c, func() {
		ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
				sectionHeading(c, "Recent activity", "The newest entries written to the audit chain.")
			})
			ui.Row(c).Gap(space2).AlignItems(ui.Center).Children(func() {
				if n := len(a.dashboard.recent); n > 0 {
					badge(c, fmt.Sprintf("%d", n), c.Theme().Selection, c.Theme().Text)
				}
			})
		})
		if a.dashboard.recentErr != nil &&
			!loadWarning(c, "Couldn't read recent activity", a.dashboard.recentErr, len(a.dashboard.recent) > 0) {
			return
		}
		if len(a.dashboard.recent) == 0 {
			// A fresh chain is a real state, not a table that failed to load:
			// the mark and the sentence say so in the same language as the
			// list's empty states instead of a lone grey line.
			ui.Column(c).FillWidth().Padding(space6, space5).Gap(space2).
				AlignItems(ui.Center).Children(func() {
				iconTile(c, pageIcons[pageBlockchain], pageAccent(c, pageBlockchain), iconTileMedium)
				ui.Text(c, tr("Nothing on the chain yet.")).FontSize(textSize(c, 12)).
					TextColor(c.Theme().TextMuted).TextAlign(ui.Center)
			})
			return
		}
		// The activity card draws the same records as the Blockchain page, in
		// the same language and with the same module hues. Two renderings of
		// the audit chain would be two different accounts of what happened.
		// The whole record is the navigation target: it names where the entry
		// came from, and taking the reader there is more useful than leaving a
		// highlight behind.
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			for i, entry := range a.dashboard.recent {
				entry := entry
				last := i == len(a.dashboard.recent)-1
				if clicked := dashboardTimelineEntry(c, entry, last); clicked {
					if target, ok := pageForBlockModule(entry.Module); ok {
						a.setPage(target)
					}
				}
			}
		})
	})
	panel.Label(title)
}

// dashboardTimelineEntry draws one event as a point on a vertical ledger
// spine. The spine is painted from the row's laid-out height rather than a
// fixed segment, so a wrapped record cannot leave a gap between events. The
// last record ends at its dot instead of drawing a line into empty space.
func dashboardTimelineEntry(c *ui.Context, entry BlockRow, last bool) bool {
	tone := recordModuleTone(c, entry.Module)
	var clicked bool
	row := ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Start)
	row.Draw(func(p *ui.Painter, r ui.Rect) {
		if last {
			return
		}
		p.Fill(ui.Rect{X: r.X + 8.5, Y: r.Y, W: 1, H: r.H}, c.Theme().Border, 0)
	})
	row.Children(func() {
		ui.Column(c).Width(18).AlignItems(ui.Center).Children(func() {
			statusDot(c, tone, 9)
		})
		b := ui.ButtonBase(c).FillWidth().Padding(space3+2, space4).Gap(space2 + 2).
			Radius(radiusSmall + 2).AlignItems(ui.Start).Label(blockRecordLabel(entry))
		fill, outline := c.Theme().Background.Mix(c.Theme().Surface, 0.35), c.Theme().Border
		switch {
		case b.Pressed():
			fill = c.Theme().SurfacePressed
			outline = tone.Alpha(0.55)
		case b.Hovered():
			fill = c.Theme().SurfaceHover
			outline = tone.Alpha(0.55)
		}
		b.Background(b.AnimateColor("timeline-fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
		b.BorderWidth(1).BorderColor(outline)
		b.Children(func() {
			ui.Column(c).FillWidth().Gap(space2).Children(func() {
				recordSentence(c, entry, false, nil)
				recordFacts(c, entry)
			})
		})
		clicked = b.Clicked()
	})
	return clicked
}

func (a *App) loadBlockchain() {
	a.loadBlockchainAt(a.blockchain.search, a.blockchain.blockLimit, a.blockchain.blockOffset)
}

// loadBlockchainAt loads one block query and commits its pager state only
// after the rows succeed. The status read remains a separate step so a broken
// chain warning is never replaced by an empty table.
func (a *App) loadBlockchainAt(search string, rawLimit float64, offset int) bool {
	if a.services == nil {
		return false
	}
	search = strings.TrimSpace(search)
	limit := blockchainBlockLimit(rawLimit)
	if offset < 0 {
		offset = 0
	}
	a.blockchain.queriedSearch, a.blockchain.queriedLimit = search, limit
	res, err := a.services.LedgerStatus()
	a.blockchain.loadErr = err
	if err != nil {
		return false
	}
	a.blockchain.valid = res.Valid
	a.blockchain.checked = true
	a.blockchain.height = res.Height
	a.blockchain.blocks = res.Blocks
	// The blocks themselves are what an auditor reads; the counters alone say
	// how much chain there is, not what it recorded. A failure here is its own
	// warning: the counters and the verdict above it are still valid.
	page, blocksErr := a.services.LedgerBlocksPage(search, limit, offset)
	a.blockchain.blocksErr = blocksErr
	if blocksErr != nil {
		return false
	}
	a.blockchain.rows = page.Rows
	a.blockchain.blockTotal = page.Total
	a.blockchain.blockOffset = offset
	return true
}

func blockchainBlockLimit(raw float64) int {
	limit := int(raw)
	if limit < 10 {
		return 10
	}
	if limit > 10000 {
		return 10000
	}
	return limit
}

func (a *App) blockchainView(c *ui.Context) {
	a.scrollPage(c, func() {
		guide(c, &a.guides.blockchain,
			"Every module appends its records to one hash chain; each block commits to the previous hash.",
			"The badge verifies the chain. Search matches module, payload or height across the whole chain; Show sets the page size.",
		)
		// The page header already says what the ledger is, so the card does not
		// repeat it.
		formCard(c, "Ledger", "", func() {
			a.blockchainBody(c)
		})
	})
}

func (a *App) blockchainBody(c *ui.Context) {
	if a.blockchain.loadErr != nil && !loadWarning(c, "Couldn't read the chain", a.blockchain.loadErr, a.blockchain.checked) {
		return
	}
	if !a.blockchain.checked {
		emptyNote(c, "Press Refresh to verify the chain.")
		return
	}
	// The facts and the verdict share one line: a full-width readout for two
	// short numbers left a band of empty panel above the controls that use
	// them. The tiles stay beside the verdict so the health of the chain is
	// read in the same glance as its size.
	ui.Row(c).FillWidth().Wrap().Gap(space2).AlignItems(ui.Center).Children(func() {
		statChip(c, "Blocks", strconv.Itoa(a.blockchain.blocks))
		statChip(c, "Height", strconv.Itoa(a.blockchain.height))
		if a.blockchain.valid {
			verdictPill(c, true, "Hash chain verified")
		} else {
			verdictPill(c, false, "Hash chain is broken")
		}
	})
	a.blockchainBlocks(c)
}

// blockchainBlocks lists what the chain recorded, newest first. The adapter
// searches the whole chain but returns only one bounded page and its exact
// total, so Previous/Next work the same for filtered and unfiltered views.
func (a *App) blockchainBlocks(c *ui.Context) {
	limit := blockchainBlockLimit(a.blockchain.blockLimit)
	query := strings.TrimSpace(a.blockchain.search)
	queryChanged := query != a.blockchain.queriedSearch
	if queryChanged || limit != a.blockchain.queriedLimit {
		// A changed search or page size starts at the newest rows. The helper
		// records the attempt even on failure, so a database error cannot turn
		// every repaint into another read.
		a.loadBlockchainAt(query, a.blockchain.blockLimit, 0)
	}
	if a.blockchain.blocksErr != nil &&
		!loadWarning(c, "Couldn't read chain blocks", a.blockchain.blocksErr, len(a.blockchain.rows) > 0) {
		return
	}
	announceSearchResults(c, queryChanged, a.blockchain.blockTotal)
	if a.blockchain.blockTotal == 0 && query == "" {
		// Only the genesis block exists: there is nothing to search or page
		// through, and the counters above already report the block. A search
		// field over an empty list is a control that cannot work.
		emptyPanel(c, pageIcons[pageBlockchain], pageAccent(c, pageBlockchain),
			"No module records yet; the first write adds one.")
		return
	}
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
		ui.Row(c).Grow(1).Children(func() {
			searchField(c, &a.blockchain.search, "Search blocks").
				Description(tr("Search matches module, payload or height across the whole chain."))
		})
		ui.Field(c, tr("Show"), func() {
			ui.NumberInput(c, &a.blockchain.blockLimit, 10, 10000, 10)
		})
	})
	if a.blockchain.blockTotal == 0 {
		emptyNote(c, fmt.Sprintf(tr("No blocks match “%s”."), query))
		return
	}
	if len(a.blockchain.rows) == 0 {
		// The selected page no longer exists, for example after the chain was
		// reset elsewhere. Return to the first page that does.
		a.blockchain.blockOffset = 0
		a.loadBlockchainAt(query, a.blockchain.blockLimit, 0)
		return
	}
	total := a.blockchain.blockTotal
	rangeLabel := fmt.Sprintf(tr("Entries %d–%d of %d"),
		a.blockchain.blockOffset+1, a.blockchain.blockOffset+len(a.blockchain.rows), total)
	if previous, next := pagerControls(c, "Blockchain ledger", a.blockchain.blockOffset, rangeLabel,
		a.blockchain.blockOffset == 0, a.blockchain.blockOffset+len(a.blockchain.rows) >= total); previous || next {
		if previous {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				max(0, a.blockchain.blockOffset-limit))
		} else {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				a.blockchain.blockOffset+limit)
		}
	}
	// The ledger is a stream of records, not a grid of payload cells: each
	// entry reads as "module · action · object" with its facts and its exact
	// payload one click away. This replaced a three-column table whose
	// payload cell was the raw JSON or pipe-delimited key bytes, which made
	// the page an unreadable dump of the very thing it was meant to explain.
	//
	// A record already wraps its facts and its payload, so the same layout
	// serves the large-text preference instead of switching to a second
	// rendering of the same data.
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for i, entry := range a.blockchain.rows {
			i, entry := i, entry
			blockRecord(c, entry, a.blockchain.expanded == i, func() {
				a.toggleBlockPayload(i)
			})
		}
	})
}

// toggleBlockPayload opens or closes one record's exact payload. Only one is
// open at a time: the page is a scroll of sentences, and a reader comparing
// two payloads can toggle between them faster than by scrolling past both.
func (a *App) toggleBlockPayload(index int) {
	if a.blockchain.expanded == index {
		a.blockchain.expanded = -1
		return
	}
	a.blockchain.expanded = index
}

// pageForBlockModule maps an audit payload's module label to the page that
// owns it. Unknown or synthetic blocks stay on the dashboard.
func pageForBlockModule(module string) (page, bool) {
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "lottery":
		return pageLottery, true
	case "voting":
		return pageVoting, true
	case "token":
		return pageToken, true
	case "nft":
		return pageNFT, true
	case "oracle":
		return pageOracle, true
	default:
		return 0, false
	}
}
