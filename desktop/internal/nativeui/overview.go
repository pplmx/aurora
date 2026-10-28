package nativeui

import (
	"fmt"
	"strconv"
	"strings"

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
		// A 3x2 grid of cards: six counters read as one block. The ledger verdict
		// is the one value that can be bad news, so it takes the colour that says
		// which it is — and its note still spells the outcome out in words.
		integrityTone := ui.Hex(auroraGreen)
		if a.dashboard.Integrity != "verified" {
			integrityTone = ui.Hex(auroraRed)
		}
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
			{"Ledger Integrity", a.dashboard.Integrity, "Hash-chain verification", integrityTone, false, pageBlockchain},
			{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds", textTone, true, pageOracle},
		}
		showMetrics := func() {
			// Six cards read as two rows of three on a wide window. At the
			// minimum width they become two columns, so a label and its note
			// keep their line instead of wrapping inside a 200px card.
			width, _ := c.Size()
			columns := 3
			if width < 1100 {
				columns = 2
			}
			if width < 760 {
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
		ui.Row(c).Wrap().Gap(8).Children(func() {
			a.quickStartButton(c, "Run a draw", pageLottery)
			a.quickStartButton(c, "Create a token", pageToken)
			a.quickStartButton(c, "Register a voter", pageVoting)
			a.quickStartButton(c, "Add a data source", pageOracle)
			a.quickStartButton(c, "Mint an NFT", pageNFT)
		})
	})
}

func (a *App) quickStartButton(c *ui.Context, label string, target page) {
	if secondaryButton(c, label).Clicked() {
		a.setPage(target)
	}
}

// dashboardRecent draws the newest chain entries. The counters say how much
// the store holds; this card says what happened, which is what an operator
// opening the window usually wants to know first.
func (a *App) dashboardRecent(c *ui.Context) {
	formCard(c, "Recent activity", "The newest entries written to the audit chain.", func() {
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
					tableText(c, entry.Module)
				case 2:
					tableText(c, entry.Summary).FontSize(textSize(c, 12))
				}
			})
			if table.Changed() {
				a.chooseRecentBlock(rows, a.dashboard.recentChoice)
			}
		}
	})
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
	readout(c, []readoutRow{
		{"Blocks", strconv.Itoa(a.blockchain.blocks)},
		{"Height", strconv.Itoa(a.blockchain.height)},
	})
	// The badge is inline: drawn straight into the card it took the card's
	// whole width and read as a banner rather than a verdict.
	ui.Row(c).Children(func() {
		if a.blockchain.valid {
			badge(c, "Hash chain verified", ui.Hex(auroraGreen), c.Theme().Background)
		} else {
			badge(c, "Hash chain is broken", ui.Hex(auroraRed), c.Theme().Background)
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
				tableText(c, b.Module)
			case 2:
				tableText(c, b.Summary).FontSize(textSize(c, 12))
			}
		})
	}
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
