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
			ui.Text(c, tr("Loading counters…")).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
		})
		return
	}

	a.scrollPage(c, func() {
		// A 3x2 grid of cards: six counters read as one block. The ledger verdict
		// is the one value that can be bad news, so it takes the colour that says
		// which it is — and its note still spells the outcome out in words.
		integrityTone := auroraGreen
		if a.dashboard.Integrity != "verified" {
			integrityTone = auroraRed
		}
		metrics := []struct {
			label   string
			value   string
			note    string
			tone    string
			numeric bool
			target  page
		}{
			{"Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded", ink, true, pageLottery},
			{"Total Votes", strconv.Itoa(a.dashboard.Votes), "Signed ballots cast", ink, true, pageVoting},
			{"Candidates", strconv.Itoa(a.dashboard.Candidates), "Registered contenders", ink, true, pageVoting},
			{"Voting Sessions", strconv.Itoa(a.dashboard.Sessions), "Governance windows", ink, true, pageVoting},
			{"Ledger Integrity", a.dashboard.Integrity, "Hash-chain verification", integrityTone, false, pageBlockchain},
			{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds", ink, true, pageOracle},
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
// when the changed module (token, NFT) has no dashboard card.
func (a *App) dashboardIsEmpty() bool {
	d := a.dashboard
	return d.Lotteries == 0 &&
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
		a.dashboard.recentTable.Selected = &a.dashboard.recentChoice
		table := ui.Table(c, &a.dashboard.recentTable, []ui.TableColumn{
			{Title: tr("Block"), Width: 80, Align: ui.End},
			{Title: tr("Module"), Width: 120},
			{Title: tr("Summary"), MinWidth: 200},
		}, len(rows), func(row, col int) {
			entry := rows[row]
			switch col {
			case 0:
				ui.Text(c, strconv.FormatInt(entry.Height, 10))
			case 1:
				ui.Text(c, entry.Module).SingleLine()
			case 2:
				ui.Text(c, entry.Summary).FontSize(textSize(c, 12)).SingleLine()
			}
		})
		if table.Changed() && a.dashboard.recentChoice >= 0 && a.dashboard.recentChoice < len(rows) {
			if target, ok := pageForBlockModule(rows[a.dashboard.recentChoice].Module); ok {
				// The row names where the record came from; taking the reader
				// there is more useful than leaving a highlight on the dashboard.
				a.dashboard.recentChoice = -1
				a.setPage(target)
				return
			}
			a.dashboard.recentChoice = -1
		}
	})
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
			badge(c, "Hash chain verified", auroraGreen, night)
		} else {
			badge(c, "Hash chain is broken", auroraRed, night)
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
	if query != a.blockchain.queriedSearch || limit != a.blockchain.queriedLimit {
		// A changed search or page size starts at the newest rows. The helper
		// records the attempt even on failure, so a database error cannot turn
		// every repaint into another read.
		a.loadBlockchainAt(query, a.blockchain.blockLimit, 0)
	}
	if a.blockchain.blocksErr != nil &&
		!loadWarning(c, "Couldn't read chain blocks", a.blockchain.blocksErr, len(a.blockchain.rows) > 0) {
		return
	}
	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
		ui.Row(c).Grow(1).Children(func() {
			ui.SearchField(c, &a.blockchain.search).Label(tr("Search blocks"))
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
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		if secondaryButton(c, "Previous").Disabled(a.blockchain.blockOffset == 0).Clicked() {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				max(0, a.blockchain.blockOffset-limit))
			return
		}
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf(tr("Entries %d–%d of %d"), a.blockchain.blockOffset+1, a.blockchain.blockOffset+len(a.blockchain.rows), total)).
			FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted))
		ui.Spacer(c)
		if secondaryButton(c, "Next").Disabled(a.blockchain.blockOffset+len(a.blockchain.rows) >= total).Clicked() {
			a.loadBlockchainAt(query, a.blockchain.blockLimit,
				a.blockchain.blockOffset+limit)
		}
	})
	ui.Table(c, &a.blockchain.table, []ui.TableColumn{
		{Title: tr("Height"), Width: 90, Align: ui.End},
		{Title: tr("Module"), Width: 130},
		{Title: tr("Payload"), MinWidth: 200},
	}, len(a.blockchain.rows), func(row, col int) {
		b := a.blockchain.rows[row]
		switch col {
		case 0:
			ui.Text(c, strconv.FormatInt(b.Height, 10))
		case 1:
			ui.Text(c, b.Module).SingleLine()
		case 2:
			ui.Text(c, b.Summary).FontSize(textSize(c, 12)).SingleLine()
		}
	})
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
