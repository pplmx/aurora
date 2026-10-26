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
	count        float64

	verifyID     string
	verifyResult *VerifyResult

	rows    []DrawRow
	loadErr error
	// search narrows the history by id, seed, participant or winner;
	// appliedSearch is the query the current highlight belongs to.
	search        string
	appliedSearch string

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

	// selected is the history row the operator chose. The id column shows a
	// shortened id (the full one does not fit beside the roster), so choosing
	// a row is how the full id reaches the verifier.
	selected int
}

// lotteryHistoryLimit is how many draws the history table asks for. It is one
// page of the store, newest first, which is what an operator scrolling a list
// reads; the CLI's own history command pages the same way.
const lotteryHistoryLimit = 100

// loadLottery fetches the stored draws.
func (a *App) loadLottery() {
	if a.services == nil {
		return
	}
	rows, err := a.services.LotteryHistory(lotteryHistoryLimit)
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
		// The two forms sit side by side; the history table keeps the full
		// width, because a half-width table wrapped participant lists onto a
		// second line that the fixed row height then clipped.
		twoColumns(c, func() {
			a.lotteryCreate(c)
		}, func() {
			a.lotteryVerify(c)
		})
		a.lotteryHistory(c)
	})
}

func (a *App) lotteryCreate(c *ui.Context) {
	formCard(c, "Create Draw", "Participants and seed determine the draw; the count is how many winners to pick.", func() {
		field(c, "Participants (comma-separated)", &a.lottery.participants)
		fieldError(c, a.lottery.participantsErr)
		field(c, fmt.Sprintf("Seed (at least %d characters)", lottery.MinSeedLength), &a.lottery.seed)
		fieldError(c, a.lottery.seedErr)
		ui.Field(c, "Count", func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.NumberInput(c, &a.lottery.count, 1, float64(lottery.MaxWinners), 1)
				hint(c, "How many winners to draw; must not exceed the participant count.")
			})
		})
		fieldError(c, a.lottery.countErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Create Draw").Clicked() {
				a.createLottery()
			}
		})
	})
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
	count := int(a.lottery.count)

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
	if float64(count) != a.lottery.count {
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
	a.lottery.count = 1
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
	formCard(c, "History", "Every recorded draw, newest first. Choose a row to load its id into Verify.", func() {
		a.lotteryHistoryBody(c)
	})
}

func (a *App) lotteryHistoryBody(c *ui.Context) {
	if a.lottery.loadErr != nil && !loadWarning(c, "Failed to load history", a.lottery.loadErr, len(a.lottery.rows) > 0) {
		return
	}
	if len(a.lottery.rows) == 0 {
		emptyNote(c, "No draws recorded yet. Create one above.")
		return
	}
	rows := make([]DrawRow, 0, len(a.lottery.rows))
	ui.SearchField(c, &a.lottery.search).Label("Search draws")
	if a.lottery.search != a.lottery.appliedSearch {
		a.lottery.appliedSearch = a.lottery.search
		a.lottery.selected = -1
	}
	for _, draw := range a.lottery.rows {
		if containsFold(a.lottery.search, draw.ID, draw.Seed,
			strings.Join(draw.Participants, " "), strings.Join(draw.Winners, " ")) {
			rows = append(rows, draw)
		}
	}
	if len(rows) == 0 {
		emptyNote(c, "No draws match “"+strings.TrimSpace(a.lottery.search)+"”.")
		return
	}
	// Choosing a row fills the verifier with that draw's full id: the table
	// shows a shortened id, so this is how a reader gets the real one.
	a.lottery.table.Selected = &a.lottery.selected
	table := ui.Table(c, &a.lottery.table, []ui.TableColumn{
		{Title: "ID", Width: 150},
		{Title: "Block", Width: 80, Align: ui.End},
		{Title: "Participants", MinWidth: 140},
		{Title: "Winners", MinWidth: 120},
	}, len(rows), func(row, col int) {
		r := rows[row]
		switch col {
		case 0:
			ui.Text(c, shortID(r.ID)).SingleLine()
		case 1:
			ui.Text(c, strconv.FormatInt(r.BlockHeight, 10))
		case 2:
			ui.Text(c, joinList(r.Participants)).SingleLine()
		case 3:
			ui.Text(c, joinList(r.Winners)).SingleLine()
		}
	})
	if table.Changed() && a.lottery.selected >= 0 && a.lottery.selected < len(rows) {
		a.lottery.verifyID = rows[a.lottery.selected].ID
	}
}

func (a *App) lotteryVerify(c *ui.Context) {
	formCard(c, "Verify a Draw", "Recompute the winners from the stored VRF proof for any draw.", func() {
		field(c, "Draw ID or block height", &a.lottery.verifyID)
		fieldError(c, a.lottery.verifyErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Verify").Clicked() {
				a.verifyLottery()
			}
		})
		if r := a.lottery.verifyResult; r != nil {
			// The verdict is a word first and a colour second: the tick or
			// cross is decoration for a reader who can see it.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if r.Valid {
					badge(c, "Verified", auroraGreen, night)
				} else {
					badge(c, "Failed", auroraRed, night)
				}
				ui.Text(c, r.ID).FontSize(12).TextColor(ui.Hex(inkMuted)).SingleLine()
			})
			for _, line := range r.Checks {
				ui.Text(c, "• "+line).FontSize(12).TextColor(ui.Hex(inkMuted))
			}
		}
	})
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
