package nativeui

import (
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
	blocks, blocksErr := a.services.LedgerBlocks()
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
			ui.Text(c, "Loading counters…").FontSize(12).TextColor(ui.Hex(inkMuted))
		})
		return
	}

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
		{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Current source health", ink, true},
	}
	// A fresh install is all zeroes and a verified empty chain, which is
	// correct but says nothing about what to do next. One line names the
	// first move instead of leaving the operator to guess.
	if a.dashboard.Lotteries == 0 && a.dashboard.Votes == 0 && a.dashboard.Oracle == 0 {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Nothing recorded yet — start from Lottery to run a verifiable draw.").
				FontSize(12).TextColor(ui.Hex(inkMuted))
		})
	}
	for start := 0; start < len(metrics); start += 3 {
		start := start
		row := metrics[start:min(start+3, len(metrics))]
		metricRow(c, len(row), func(i int) {
			m := row[i]
			metricCard(c, m.label, m.value, m.note, m.tone, m.numeric)
		})
	}
	a.dashboardRecent(c)
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
			{Title: "Block", Width: 80, Align: ui.End},
			{Title: "Module", Width: 120},
			{Title: "Summary", MinWidth: 200},
		}, len(rows), func(row, col int) {
			entry := rows[row]
			switch col {
			case 0:
				ui.Text(c, strconv.FormatInt(entry.Height, 10))
			case 1:
				ui.Text(c, entry.Module).SingleLine()
			case 2:
				ui.Text(c, entry.Summary).FontSize(12).SingleLine()
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
	// how much chain there is, not what it recorded.
	if rows, err := a.services.LedgerBlocks(); err == nil {
		a.blockchain.rows = rows
	}
}

func (a *App) blockchainView(c *ui.Context) {
	// The page header already says what the ledger is, so the card does not
	// repeat it.
	formCard(c, "Ledger", "", func() {
		a.blockchainBody(c)
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

// blockchainBlocks lists what the chain recorded, newest first.
func (a *App) blockchainBlocks(c *ui.Context) {
	if len(a.blockchain.rows) == 0 {
		emptyNote(c, "Nothing recorded on the chain yet.")
		return
	}
	rows := a.blockchain.rows
	ui.Table(c, &a.blockchain.table, []ui.TableColumn{
		{Title: "Height", Width: 90, Align: ui.End},
		{Title: "Module", Width: 130},
		{Title: "Payload", MinWidth: 200},
	}, len(rows), func(row, col int) {
		b := rows[row]
		switch col {
		case 0:
			ui.Text(c, strconv.FormatInt(b.Height, 10))
		case 1:
			ui.Text(c, b.Module).SingleLine()
		case 2:
			ui.Text(c, b.Summary).FontSize(12).SingleLine()
		}
	})
}
