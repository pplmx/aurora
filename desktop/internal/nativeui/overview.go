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
	recent       []BlockRow
	recentErr    error
	recentTable  ui.ListState
	recentChoice int
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
	table   ui.ListState
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
		// The verdict leads the page: it is the one fact that can be bad news,
		// and it decides whether anything else on the dashboard is trustworthy.
		a.dashboardHero(c)
		// The counters below answer "how much"; the activity card answers
		// "what changed"; the shortcuts come last for a store that already has
		// content. A fresh store reverses the priority: its first move is the
		// only useful thing on the page.
		textTone := c.Theme().Text
		metrics := []struct {
			label   string
			value   string
			note    string
			tone    ui.Color
			numeric bool
			target  page
		}{
			{"Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded", textTone, true, pageLottery},
			{"Total Votes", strconv.Itoa(a.dashboard.Votes), "Signed ballots cast", textTone, true, pageVoting},
			{"Candidates", strconv.Itoa(a.dashboard.Candidates), "Registered contenders", textTone, true, pageVoting},
			{"Voting Sessions", strconv.Itoa(a.dashboard.Sessions), "Governance windows", textTone, true, pageVoting},
			{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds", textTone, true, pageOracle},
			{"Ledger Blocks", strconv.Itoa(a.dashboard.Blocks), "Entries on the audit chain", textTone, true, pageBlockchain},
		}
		showMetrics := func() {
			// Three counters per row on a wide window; two at the minimum
			// width so a label and its note keep their line instead of
			// wrapping inside a 200px card.
			ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Center).Children(func() {
				eyebrow(c, "At a glance")
				ui.Row(c).Height(1).Grow(1).Background(c.Theme().Border)
			})
			width, _ := c.Size()
			// The breakpoints scale with the text preference: at 2x a card
			// that held one label and a note at 1x needs the width of two.
			scale := textScale(c)
			columns := 3
			if width < 1100*scale {
				columns = 2
			}
			if width < 760*scale {
				columns = 1
			}
			for start := 0; start < len(metrics); start += columns {
				start := start
				row := metrics[start:min(start+columns, len(metrics))]
				metricRow(c, len(row), func(i int) {
					m := row[i]
					if metricCard(c, m.label, m.value, m.note, m.tone, m.numeric, m.target) {
						a.setPage(m.target)
					}
				})
			}
		}
		if a.dashboardIsEmpty() {
			// A fresh install has nothing to summarize yet. Start with the
			// first move, then show the (zero) counters and the empty chain.
			a.dashboardQuickStart(c)
			showMetrics()
			a.dashboardRecent(c)
			return
		}
		// Returning users come here for the state, then for what changed;
		// the action card is the last thing, not the first.
		showMetrics()
		a.dashboardRecent(c)
		a.dashboardQuickStart(c)
	})
	a.liveRefresh(c, a.loadDashboard)
}

// dashboardHero is the page's verdict panel: the chain's integrity, spelled
// out in a word, with the explanation that makes the word mean something and
// the one route to the full audit page. It replaces a sixth counter card: the
// one value an operator must not miss deserves more than a third of a row.
func (a *App) dashboardHero(c *ui.Context) {
	verified := a.dashboard.Integrity == "verified"
	tone := ui.Hex(auroraGreen)
	description := "Every module appends its records to one hash chain; each block commits to the previous hash."
	if !verified {
		tone = ui.Hex(auroraRed)
		description = "The stored chain did not verify. Inspect the ledger before trusting the counters below."
	}
	panel := ui.Column(c).FillWidth().Padding(space6, space5).Gap(space4).
		Radius(radiusLarge).LinearGradient(panelGradient(c, tone, 0.1)).
		BorderWidth(1).BorderColor(tone.Alpha(0.28)).
		Shadow(0, 28, 60, -28, ui.RGBA(0, 0, 0, 0.6)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.05))
	accentHairline(c, panel, tone)
	panel.Children(func() {
		ui.Row(c).FillWidth().Gap(space4).AlignItems(ui.Center).Children(func() {
			iconTile(c, shieldCheckIcon, tone, iconTileLarge)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(space1).Children(func() {
				eyebrow(c, "Ledger Integrity")
				ui.Row(c).Gap(space3).AlignItems(ui.Center).Children(func() {
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
			ui.Column(c).AlignItems(ui.End).Children(func() {
				inspect := secondaryButton(c, "Inspect ledger")
				if inspect.Clicked() {
					a.setPage(pageBlockchain)
				}
			})
		})
	})
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
		ui.Row(c).Wrap().Gap(space2).Children(func() {
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
	tone := pageAccent(target)
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
	pressLift(c, b, b.Hovered())
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

// dashboardRecent draws the newest chain entries. The counters say how much
// the store holds; this card says what happened, which is what an operator
// opening the window usually wants to know first.
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
				if quietButton(c, "View all").Clicked() {
					a.setPage(pageBlockchain)
				}
			})
		})
		if a.dashboard.recentErr != nil &&
			!loadWarning(c, "Couldn't read recent activity", a.dashboard.recentErr, len(a.dashboard.recent) > 0) {
			return
		}
		if len(a.dashboard.recent) == 0 {
			emptyNote(c, "Nothing on the chain yet.")
			return
		}
		rows := a.dashboard.recent
		if stackedRecords(c) {
			a.dashboardRecentRecords(c, rows)
		} else {
			a.dashboard.recentTable.Selected = &a.dashboard.recentChoice
			table := namedTable(c, "Recent activity", &a.dashboard.recentTable, []ui.TableColumn{
				{Title: tr("Block"), MinWidth: tableColumnWidth(c, 140), Align: ui.End},
				{Title: tr("Module"), Width: 120},
				{Title: tr("Summary"), MinWidth: 200},
			}, len(rows), func(row, col int) {
				entry := rows[row]
				switch col {
				case 0:
					tableText(c, strconv.FormatInt(entry.Height, 10))
				case 1:
					tableText(c, entry.Module).TextColor(blockModuleTone(c, entry.Module))
				case 2:
					tableText(c, entry.Summary).FontSize(textSize(c, 12))
				}
			})
			if table.Changed() {
				a.chooseRecentBlock(rows, a.dashboard.recentChoice)
			}
		}
	})
	panel.Label(title)
}

// dashboardRecentRecords keeps the full audit payload visible and makes the
// whole event the navigation target when enlarged text no longer fits the
// three-column summary table.
func (a *App) dashboardRecentRecords(c *ui.Context, rows []BlockRow) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for i, entry := range rows {
			i, entry := i, entry
			displayName := fmt.Sprintf("%s #%d", valueOr(entry.Module), entry.Height)
			selected := i == a.dashboard.recentChoice
			record := selectableRecordPanel(c, selected,
				recordChoiceLabel(displayName, selected), func() {
					blockRecordBody(c, entry)
				})
			if record.Clicked() {
				a.chooseRecentBlock(rows, i)
			}
		}
	})
}

// chooseRecentBlock is shared by the table and large-text records. A known
// module opens its workbench; an unknown one still consumes the selection
// instead of leaving a stale highlight behind.
func (a *App) chooseRecentBlock(rows []BlockRow, index int) {
	if index < 0 || index >= len(rows) {
		return
	}
	a.dashboard.recentChoice = index
	if target, ok := pageForBlockModule(rows[index].Module); ok {
		// The row names where the record came from; taking the reader there is
		// more useful than leaving a highlight on the dashboard.
		a.dashboard.recentChoice = -1
		a.setPage(target)
		return
	}
	a.dashboard.recentChoice = -1
}

// blockRecordBody is the shared ledger-event record used by both the
// dashboard's selectable activity list and the Blockchain audit page.
func blockRecordBody(c *ui.Context, entry BlockRow) {
	ui.Row(c).FillWidth().Wrap().Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Text(c, valueOr(entry.Module)).FontSize(textSize(c, 13)).Bold().
			TextColor(c.Theme().Text)
		recordMetric(c, "Block", strconv.FormatInt(entry.Height, 10), c.Theme().Text)
	})
	recordField(c, "Summary", valueOr(entry.Summary))
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
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
		ui.Row(c).Grow(1).Children(func() {
			ui.SearchField(c, &a.blockchain.search).Label(tr("Search blocks")).
				Description(tr("Search matches module, payload or height across the whole chain."))
		})
		ui.Field(c, tr("Show"), func() {
			ui.NumberInput(c, &a.blockchain.blockLimit, 10, 10000, 10)
		})
	})
	if a.blockchain.blockTotal == 0 {
		if query != "" {
			emptyNote(c, fmt.Sprintf(tr("No blocks match “%s”."), query))
		} else {
			emptyNote(c, "Nothing recorded on the chain yet.")
		}
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
	if stackedRecords(c) {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			for _, entry := range a.blockchain.rows {
				entry := entry
				recordPanel(c, func() {
					blockRecordBody(c, entry)
				})
			}
		})
	} else {
		namedTable(c, "Blockchain ledger", &a.blockchain.table, []ui.TableColumn{
			{Title: tr("Height"), MinWidth: tableColumnWidth(c, 140), Align: ui.End},
			{Title: tr("Module"), Width: 130},
			{Title: tr("Payload"), MinWidth: 200},
		}, len(a.blockchain.rows), func(row, col int) {
			b := a.blockchain.rows[row]
			switch col {
			case 0:
				tableText(c, strconv.FormatInt(b.Height, 10))
			case 1:
				tableText(c, b.Module).TextColor(blockModuleTone(c, b.Module))
			case 2:
				tableText(c, b.Summary).FontSize(textSize(c, 12))
			}
		})
	}
}

// blockModuleTone gives each module its identity colour in a ledger table, so
// a reader scanning a height-ordered list can tell the writers apart before
// reading the column. An unknown module keeps the muted text colour; colour is
// never the only signal because the module name stays beside it.
func blockModuleTone(c *ui.Context, module string) ui.Color {
	if target, ok := pageForBlockModule(module); ok {
		return pageAccent(target)
	}
	return c.Theme().TextMuted
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
