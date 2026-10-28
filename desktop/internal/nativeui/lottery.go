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
	// verifyInput is the exact verifier input behind verifyResult/verifyErr.
	// The field can be edited or repopulated from History, and a verdict or
	// lookup error from another id must not remain displayed under it.
	verifyInput string

	rows    []DrawRow
	loadErr error
	// offset/hasMore page the stored history. The loader asks for one row
	// beyond the page so an exact final page disables Next.
	offset  int
	hasMore bool
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

// lotteryHistoryLimit is the number of rows in one history page.
const lotteryHistoryLimit = 100

// loadLottery refreshes the current page.
func (a *App) loadLottery() {
	a.loadLotteryPage(a.lottery.offset)
}

// loadLotteryPage commits a page only after its read succeeds, so a failed
// refresh or pager move leaves the rows and offset describing the same data.
func (a *App) loadLotteryPage(offset int) bool {
	if a.services == nil {
		return false
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.services.LotteryHistory(lotteryHistoryLimit+1, offset)
	a.lottery.loadErr = err
	if err != nil {
		return false
	}
	if len(rows) == 0 && offset > 0 {
		return a.loadLotteryPage(0)
	}
	a.lottery.offset = offset
	a.lottery.hasMore = len(rows) > lotteryHistoryLimit
	if a.lottery.hasMore {
		rows = rows[:lotteryHistoryLimit]
	}
	a.lottery.rows = rows
	return true
}

// lotteryView renders the draw form, the history table and the verifier.
func (a *App) lotteryView(c *ui.Context) {
	a.scrollPage(c, func() {
		guide(c, &a.guides.lottery,
			"Enter comma-separated participants and a seed of at least the shown length; Count is how many winners to draw.",
			"Create the draw: the result and its VRF proof are recorded on the ledger.",
			"Choose the new row in History and press Verify to recompute the winners from the stored proof.",
		)
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
		validatedField(c, "Participants (comma-separated)", &a.lottery.participants, a.lottery.participantsErr)
		validatedField(c, fmt.Sprintf(tr("Seed (at least %d characters)"), lottery.MinSeedLength), &a.lottery.seed, a.lottery.seedErr)
		fieldWithError(c, "Count", a.lottery.countErr, func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.NumberInput(c, &a.lottery.count, 1, float64(lottery.MaxWinners), 1)
				hint(c, "How many winners to draw; must not exceed the participant count.")
			})
		})
		ui.Row(c).Children(func() {
			if primaryButton(c, "Create Draw").Disabled(!allSet(a.lottery.participants, a.lottery.seed)).Clicked() {
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

	a.runWrite("Running draw…", func() func() {
		row, err := a.services.CreateDraw(participants, seed, count)
		return func() {
			if err != nil {
				a.lottery.status = errStatus(err)
				return
			}
			a.lottery.status = okStatus("Draw %s recorded at block %d. Winners: %s", row.ID, row.BlockHeight, joinList(row.Winners))
			// Advance the verifier to the fresh draw so the operator can re-check
			// it without copying the id, matching the Web UI's behaviour.
			a.lottery.verifyID = row.ID
			a.clearLotteryVerdict()
			a.lottery.participants = ""
			a.lottery.seed = ""
			a.lottery.count = 1
			if a.loadLotteryPage(0) {
				a.selectLotteryByID(row.ID)
			}
		}
	})
}

// selectLotteryByID highlights the new draw in the history table and clears a
// search that would hide it, so a successful draw is visible in the list and
// not only in the status sentence.
func (a *App) selectLotteryByID(id string) {
	a.lottery.search, a.lottery.appliedSearch = "", ""
	for i, draw := range a.lottery.rows {
		if draw.ID == id {
			a.lottery.selected = i
			return
		}
	}
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
	ui.SearchField(c, &a.lottery.search).Label(tr("Search draws"))
	if a.lottery.search != a.lottery.appliedSearch {
		a.lottery.appliedSearch = a.lottery.search
		a.lottery.selected = -1
	}
	if previous, next := listPager(c, a.lottery.offset, len(a.lottery.rows), a.lottery.hasMore); previous || next {
		target := a.lottery.offset
		if previous {
			target = max(0, target-lotteryHistoryLimit)
		} else {
			target += lotteryHistoryLimit
		}
		if a.loadLotteryPage(target) {
			a.lottery.selected = -1
		}
		return
	}
	if a.lottery.search != "" {
		hint(c, "Search narrows the current page; use Previous/Next for older draws.")
	}
	for _, draw := range a.lottery.rows {
		if containsFold(a.lottery.search, draw.ID, draw.Seed,
			strings.Join(draw.Participants, " "), strings.Join(draw.Winners, " ")) {
			rows = append(rows, draw)
		}
	}
	if len(rows) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No draws match “%s”."), strings.TrimSpace(a.lottery.search)))
		return
	}
	if stackedRecords(c) {
		a.lotteryHistoryRecords(c, rows)
		return
	}
	// Choosing a row fills the verifier with that draw's full id: the table
	// shows a shortened id, so this is how a reader gets the real one.
	a.lottery.table.Selected = &a.lottery.selected
	table := namedTable(c, "Draw history", &a.lottery.table, []ui.TableColumn{
		{Title: tr("ID"), Width: 150},
		{Title: tr("Block"), MinWidth: tableColumnWidth(c, 140), Align: ui.End},
		{Title: tr("Participants"), MinWidth: 140},
		{Title: tr("Winners"), MinWidth: 120},
	}, len(rows), func(row, col int) {
		r := rows[row]
		switch col {
		case 0:
			tableTextFull(c, shortID(r.ID), r.ID)
		case 1:
			tableText(c, strconv.FormatInt(r.BlockHeight, 10))
		case 2:
			tableText(c, joinList(r.Participants))
		case 3:
			tableText(c, joinList(r.Winners))
		}
	})
	if table.Changed() && a.lottery.selected >= 0 && a.lottery.selected < len(rows) {
		a.lottery.verifyID = rows[a.lottery.selected].ID
		// The previous verdict belongs to the previously chosen draw; keep it
		// from sitting beside a new id as if it described it.
		a.clearLotteryVerdict()
	}
}

// lotteryHistoryRecords keeps every draw fact and the Verify handoff visible
// when large text makes the four-column history table wider than the window.
func (a *App) lotteryHistoryRecords(c *ui.Context, rows []DrawRow) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		for _, draw := range rows {
			draw := draw
			recordPanel(c, func() {
				ui.Text(c, draw.ID).FontSize(textSize(c, 13)).Bold().
					TextColor(ui.Hex(aurora))
				recordField(c, "Block", strconv.FormatInt(draw.BlockHeight, 10))
				recordField(c, "Participants", joinList(draw.Participants))
				recordField(c, "Winners", joinList(draw.Winners))
				ui.Row(c).Children(func() {
					if secondaryButton(c, "Use this draw for Verify").Clicked() {
						a.lottery.verifyID = draw.ID
						a.clearLotteryVerdict()
					}
				})
			})
		}
	})
}

func (a *App) lotteryVerify(c *ui.Context) {
	formCard(c, "Verify a Draw", "Recompute the winners from the stored VRF proof for any draw.", func() {
		current := strings.TrimSpace(a.lottery.verifyID)
		message := ""
		if a.lottery.verifyInput == current {
			message = a.lottery.verifyErr
		}
		validatedField(c, "Draw ID or block height", &a.lottery.verifyID, message)
		copyRow(c, "Draw ID", strings.TrimSpace(a.lottery.verifyID), nil)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Verify").Disabled(!allSet(a.lottery.verifyID)).Clicked() {
				a.verifyLottery()
			}
		})
		if r := a.lottery.verifyResult; r != nil && a.lottery.verifyInput == current {
			// The verdict is a word first and a colour second: the tick or
			// cross is decoration for a reader who can see it.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if r.Valid {
					badge(c, "Verified", ui.Hex(auroraGreen), c.Theme().Background)
				} else {
					badge(c, "Failed", ui.Hex(auroraRed), c.Theme().Background)
				}
				ui.Text(c, r.ID).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).SingleLine()
			})
			for _, line := range r.Checks {
				ui.Text(c, "• "+line).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			}
		}
	})
}

func (a *App) verifyLottery() {
	// A new attempt replaces the evidence: the previous verdict belongs to the
	// id that was in the field before, and a failed lookup must not leave it
	// looking current.
	a.clearLotteryVerdict()
	id := strings.TrimSpace(a.lottery.verifyID)
	a.lottery.verifyInput = id
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

// clearLotteryVerdict drops both halves of the last verification attempt. A
// new input or selected draw must not inherit another subject's verdict or
// error.
func (a *App) clearLotteryVerdict() {
	a.lottery.verifyErr = ""
	a.lottery.verifyResult = nil
	a.lottery.verifyInput = ""
}
