package nativeui

import (
	"fmt"
	"strconv"

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
	recent      []BlockRow
	recentErr   error
	recentTable ui.ListState
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
	search     string
	blockLimit float64
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
		}{
			{"Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded", ink, true},
			{"Total Votes", strconv.Itoa(a.dashboard.Votes), "Signed ballots cast", ink, true},
			{"Candidates", strconv.Itoa(a.dashboard.Candidates), "Registered contenders", ink, true},
			{"Voting Sessions", strconv.Itoa(a.dashboard.Sessions), "Governance windows", ink, true},
			{"Ledger Integrity", a.dashboard.Integrity, "Hash-chain verification", integrityTone, false},
			{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Registered data feeds", ink, true},
		}
		// A fresh install is all zeroes and a verified empty chain, which is
		// correct but says nothing about what to do next. The quick-start card
		// names a first move for each module; the same buttons stay useful once
		// the ledger has data, so the page does not only guide a first launch.
		a.dashboardQuickStart(c)
		for start := 0; start < len(metrics); start += 3 {
			start := start
			row := metrics[start:min(start+3, len(metrics))]
			metricRow(c, len(row), func(i int) {
				m := row[i]
				metricCard(c, m.label, m.value, m.note, m.tone, m.numeric)
			})
		}
		a.dashboardRecent(c)
	})
	a.liveRefresh(c, a.loadDashboard)
}

// dashboardQuickStart names the first move in each module. The sidebar lists
// modules, not tasks; an operator opening a fresh window otherwise has to
// guess which page starts a draw, a token or a feed.
func (a *App) dashboardQuickStart(c *ui.Context) {
	formCard(c, "Quick start", "One first move per module; use the sidebar for everything else.", func() {
		if a.dashboard.Lotteries == 0 && a.dashboard.Votes == 0 && a.dashboard.Oracle == 0 {
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
		ui.Table(c, &a.dashboard.recentTable, []ui.TableColumn{
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
	})
}

func (a *App) loadBlockchain() {
	if a.services == nil {
		return
	}
	res, err := a.services.LedgerStatus()
	a.blockchain.loadErr = err
	if err != nil {
		return
	}
	a.blockchain.valid = res.Valid
	a.blockchain.checked = true
	a.blockchain.height = res.Height
	a.blockchain.blocks = res.Blocks
	// The blocks themselves are what an auditor reads; the counters alone say
	// how much chain there is, not what it recorded. A failure here is its own
	// warning: the counters and the verdict above it are still valid.
	rows, blocksErr := a.services.LedgerBlocks(0)
	a.blockchain.blocksErr = blocksErr
	if blocksErr == nil {
		a.blockchain.rows = rows
	}
}

func (a *App) blockchainView(c *ui.Context) {
	a.scrollPage(c, func() {
		guide(c, &a.guides.blockchain,
			"Every module appends its records to one hash chain; each block commits to the previous hash.",
			"The badge verifies the chain. Search narrows by module, payload or height, and Show bounds how many rows are built.",
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

// blockchainBlocks lists what the chain recorded, newest first, narrowed by
// the search field and bounded by the row limit: a long-lived ledger can hold
// tens of thousands of entries, and building every row each frame is work the
// reader did not ask for.
func (a *App) blockchainBlocks(c *ui.Context) {
	if a.blockchain.blocksErr != nil &&
		!loadWarning(c, "Couldn't read chain blocks", a.blockchain.blocksErr, len(a.blockchain.rows) > 0) {
		return
	}
	if len(a.blockchain.rows) == 0 {
		emptyNote(c, "Nothing recorded on the chain yet.")
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
	matching := make([]BlockRow, 0, len(a.blockchain.rows))
	for _, block := range a.blockchain.rows {
		if containsFold(a.blockchain.search, block.Module, block.Summary, strconv.FormatInt(block.Height, 10)) {
			matching = append(matching, block)
		}
	}
	if len(matching) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No blocks match “%s”."), a.blockchain.search))
		return
	}
	limit := int(a.blockchain.blockLimit)
	if limit < 10 {
		limit = 10
	}
	rows := matching
	if len(rows) > limit {
		rows = rows[:limit]
	}
	ui.Table(c, &a.blockchain.table, []ui.TableColumn{
		{Title: tr("Height"), Width: 90, Align: ui.End},
		{Title: tr("Module"), Width: 130},
		{Title: tr("Payload"), MinWidth: 200},
	}, len(rows), func(row, col int) {
		b := rows[row]
		switch col {
		case 0:
			ui.Text(c, strconv.FormatInt(b.Height, 10))
		case 1:
			ui.Text(c, b.Module).SingleLine()
		case 2:
			ui.Text(c, b.Summary).FontSize(textSize(c, 12)).SingleLine()
		}
	})
	if len(rows) < len(matching) {
		hint(c, fmt.Sprintf(tr("Showing the newest %d of %d matching entries; raise Show or search to narrow."), len(rows), len(matching)))
	}
}
