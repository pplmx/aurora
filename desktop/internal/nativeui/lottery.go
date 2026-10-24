package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

// lotteryState is the Lottery page's form and result state.
type lotteryState struct {
	participants string
	seed         string
	count        string

	verifyID     string
	verifyResult *VerifyResult

	rows    []DrawRow
	loadErr error

	// Field errors are per-input so the message appears under the control that
	// caused it, the same placement the Web UI's inline error nodes had.
	participantsErr string
	seedErr         string
	countErr        string
	verifyErr       string

	status statusLine

	// table holds the history table's scroll position and column layout, which
	// MyGo persists across frames by identity.
	table ui.ListState
}

func newLotteryState() lotteryState {
	return lotteryState{count: "1"}
}

// loadLottery fetches the stored draws.
func (a *App) loadLottery() {
	if a.services == nil {
		return
	}
	rows, err := a.services.LotteryHistory(100)
	a.lottery.loadErr = err
	if err == nil {
		a.lottery.rows = rows
	}
	// On failure the previous rows stay: a transient read error must not blank
	// a list the operator was reading.
}

// lotteryView renders the draw form, the history table and the verifier.
func (a *App) lotteryView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			a.lotteryCreate(c)
			a.lotteryHistory(c)
			a.lotteryVerify(c)
		})
	})
}

func (a *App) lotteryCreate(c *ui.Context) {
	sectionTitle(c, "Create Lottery", "Participants and seed determine the draw; the count is how many winners to pick.")
	field(c, "Participants (comma-separated)", &a.lottery.participants)
	if a.lottery.participantsErr != "" {
		ui.Text(c, a.lottery.participantsErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	field(c, fmt.Sprintf("Seed (at least %d characters)", lottery.MinSeedLength), &a.lottery.seed)
	if a.lottery.seedErr != "" {
		ui.Text(c, a.lottery.seedErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	fieldHint(c, "Count", "How many winners to draw; must not exceed the participant count.", &a.lottery.count)
	if a.lottery.countErr != "" {
		ui.Text(c, a.lottery.countErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	if ui.PrimaryButton(c, "Create Lottery").Clicked() {
		a.createLottery()
	}
	renderStatus(c, a.lottery.status)
}

// createLottery validates against the API contract before calling the service,
// so a rejection is explained next to the field instead of surfacing as an
// opaque error. The rules mirror the domain validators: at least one
// participant, a seed of three or more characters, a whole count in
// [1, participants].
func (a *App) createLottery() {
	a.lottery.participantsErr, a.lottery.seedErr, a.lottery.countErr = "", "", ""
	a.lottery.status = statusLine{}

	participants := splitParticipants(a.lottery.participants)
	seed := strings.TrimSpace(a.lottery.seed)
	count, countErr := strconv.Atoi(strings.TrimSpace(a.lottery.count))

	// The rules live in the domain validators, not here: a window that
	// hardcoded "at least 3 characters" would silently disagree with the CLI
	// and the API the day MinSeedLength changed.
	if err := lottery.ValidateParticipants(participants); err != nil {
		a.lottery.participantsErr = err.Error()
		return
	}
	if err := lottery.ValidateSeed(seed); err != nil {
		a.lottery.seedErr = err.Error()
		return
	}
	if countErr != nil {
		a.lottery.countErr = "Winner count must be a whole number."
		return
	}
	if err := lottery.ValidateWinnerCount(count, len(participants)); err != nil {
		a.lottery.countErr = err.Error()
		return
	}

	row, err := a.services.CreateDraw(participants, seed, count)
	if err != nil {
		a.lottery.status = errStatus(err)
		return
	}
	a.lottery.status = okStatus("Draw %s recorded at block %d. Winners: %s", row.ID, row.BlockHeight, joinList(row.Winners))
	// Advance the verifier to the fresh draw so the operator can re-check it
	// without copying the id, matching the Web UI's behaviour.
	a.lottery.verifyID = row.ID
	a.lottery.participants = ""
	a.lottery.seed = ""
	a.lottery.count = "1"
	a.loadLottery()
}

// splitParticipants mirrors the Web UI's split(',').map(trim).filter(Boolean).
func splitParticipants(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *App) lotteryHistory(c *ui.Context) {
	sectionTitle(c, "Lottery History", "Every recorded draw, newest first.")
	if a.lottery.loadErr != nil {
		ui.Text(c, "Failed to load history: "+a.lottery.loadErr.Error()).FontSize(12).TextColor(ui.Hex("#c62828"))
		return
	}
	if len(a.lottery.rows) == 0 {
		emptyNote(c, "No lotteries yet.")
		return
	}
	rows := a.lottery.rows
	ui.Table(c, &a.lottery.table, []ui.TableColumn{
		{Title: "ID", Width: 150},
		{Title: "Block", Width: 60, Align: ui.End},
		{Title: "Participants"},
		{Title: "Winners"},
	}, len(rows), func(row, col int) {
		r := rows[row]
		switch col {
		case 0:
			ui.Text(c, r.ID).SingleLine()
		case 1:
			ui.Text(c, strconv.FormatInt(r.BlockHeight, 10))
		case 2:
			ui.Text(c, joinList(r.Participants))
		case 3:
			ui.Text(c, joinList(r.Winners))
		}
	})
}

func (a *App) lotteryVerify(c *ui.Context) {
	sectionTitle(c, "Verify Draw", "Recompute the winners from the stored VRF proof for any draw.")
	field(c, "Lottery ID or block height", &a.lottery.verifyID)
	if a.lottery.verifyErr != "" {
		ui.Text(c, a.lottery.verifyErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	if ui.Button(c, "Verify Draw").Clicked() {
		a.verifyLottery()
	}
	if r := a.lottery.verifyResult; r != nil {
		if r.Valid {
			ui.Text(c, "✅ "+r.ID+" verified").TextColor(ui.Hex("#2e7d32"))
		} else {
			ui.Text(c, "❌ "+r.ID+" failed verification").TextColor(ui.Hex("#c62828"))
		}
		for _, line := range r.Checks {
			ui.Text(c, "• "+line).FontSize(12)
		}
	}
}

func (a *App) verifyLottery() {
	a.lottery.verifyErr = ""
	id := strings.TrimSpace(a.lottery.verifyID)
	if id == "" {
		a.lottery.verifyErr = "Enter a lottery ID or block height."
		return
	}
	res, err := a.services.VerifyDraw(id)
	if err != nil {
		a.lottery.verifyErr = err.Error()
		return
	}
	a.lottery.verifyResult = &res
}
