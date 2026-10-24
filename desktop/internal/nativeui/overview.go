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
}

// blockchainState holds the ledger verification result.
type blockchainState struct {
	valid   bool
	checked bool
	height  int
	blocks  int
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
}

func (a *App) dashboardView(c *ui.Context) {
	sectionTitle(c, "At a glance", "Live counters from every Aurora domain.")
	if a.dashboard.loadErr != nil {
		ui.Text(c, "Couldn't load stats: "+a.dashboard.loadErr.Error()).FontSize(12).TextColor(ui.Hex("#c62828"))
	}
	if !a.dashboard.loaded {
		ui.Text(c, "Loading…").FontSize(12).TextColor(ui.Hex("#6b7c7f"))
		return
	}

	// A 3x2 grid, the same composition the Web UI dashboard used: six metrics
	// read as one block instead of a wall of cards.
	metrics := []struct {
		label string
		value string
		note  string
	}{
		{"Total Lotteries", strconv.Itoa(a.dashboard.Lotteries), "VRF draws recorded"},
		{"Total Votes", strconv.Itoa(a.dashboard.Votes), "Signed ballots cast"},
		{"Candidates", strconv.Itoa(a.dashboard.Candidates), "Registered contenders"},
		{"Voting Sessions", strconv.Itoa(a.dashboard.Sessions), "Governance windows"},
		{"Ledger Integrity", a.dashboard.Integrity, "Hash-chain verification"},
		{"Oracle Feeds", strconv.Itoa(a.dashboard.Oracle), "Current source health"},
	}
	ui.Grid(c).Columns(3).Gap(12).Children(func() {
		for _, m := range metrics {
			m := m
			ui.Column(c).Padding(14).Gap(4).Children(func() {
				ui.Text(c, m.label).FontSize(11).TextColor(ui.Hex("#6b7c7f"))
				ui.Text(c, m.value).FontSize(24).Bold()
				ui.Text(c, m.note).FontSize(11).TextColor(ui.Hex("#6b7c7f"))
			})
		}
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
}

func (a *App) blockchainView(c *ui.Context) {
	sectionTitle(c, "Ledger", "The audit chain every module writes to.")
	if a.blockchain.loadErr != nil {
		ui.Text(c, "Couldn't read the chain: "+a.blockchain.loadErr.Error()).FontSize(12).TextColor(ui.Hex("#c62828"))
		return
	}
	if !a.blockchain.checked {
		emptyNote(c, "Press Refresh to verify the chain.")
		return
	}
	ui.Text(c, "Blocks: "+strconv.Itoa(a.blockchain.blocks)).FontSize(13)
	ui.Text(c, "Height: "+strconv.Itoa(a.blockchain.height)).FontSize(13)
	if a.blockchain.valid {
		ui.Text(c, "✅ Hash chain verified").TextColor(ui.Hex("#2e7d32"))
	} else {
		ui.Text(c, "❌ Hash chain is broken").TextColor(ui.Hex("#c62828"))
	}
}
