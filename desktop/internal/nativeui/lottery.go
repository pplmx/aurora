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

	// selected is the history record the operator chose. Choosing a record is
	// how the full id reaches the verifier without retyping it.
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
		validatedField(c, "Participants (comma-separated)", &a.lottery.participants, &a.lottery.participantsErr)
		validatedField(c, fmt.Sprintf(tr("Seed (at least %d characters)"), lottery.MinSeedLength), &a.lottery.seed, &a.lottery.seedErr)
		var countInput ui.Element
		count := ui.Field(c, tr("Count"), func() {
			ui.Column(c).Gap(2).Children(func() {
				countInput = ui.NumberInput(c, &a.lottery.count, 1, float64(lottery.MaxWinners), 1).
					Description(tr("How many winners to draw; must not exceed the participant count."))
				hint(c, "How many winners to draw; must not exceed the participant count.")
			})
		})
		if countInput.Changed() {
			a.lottery.countErr = ""
		}
		attachFieldError(c, count, "Count", a.lottery.countErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Create Draw", allSet(a.lottery.participants, a.lottery.seed)).Clicked() {
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
		emptyPanel(c, pageIcons[pageLottery], pageAccent(c, pageLottery),
			"No draws recorded yet. Create one above.")
		return
	}
	rows := make([]DrawRow, 0, len(a.lottery.rows))
	queryChanged := a.lottery.search != a.lottery.appliedSearch
	if queryChanged {
		a.lottery.appliedSearch = a.lottery.search
		a.lottery.selected = -1
	}
	previous, next := collectionToolbar(c, &a.lottery.search, "Search draws",
		"Search narrows the current page; use Previous/Next for older draws.",
		"Draw history", a.lottery.offset, len(a.lottery.rows), a.lottery.hasMore)
	if previous || next {
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
	for _, draw := range a.lottery.rows {
		if containsFold(a.lottery.search, draw.ID, draw.Seed,
			strings.Join(draw.Participants, " "), strings.Join(draw.Winners, " ")) {
			rows = append(rows, draw)
		}
	}
	announceSearchResults(c, queryChanged, len(rows))
	if len(rows) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No draws match “%s”."), strings.TrimSpace(a.lottery.search)))
		return
	}
	// Each draw is a record with its own Verify handoff: choosing the row loads
	// its full id into the verifier, and the button does the same without the
	// detour through the form.
	collectionList(c, "Draw history",
		collectionRecords(rows, func(draw DrawRow) collectionRecord {
			return lotteryHistoryRecord(draw, func() {
				a.lottery.verifyID = draw.ID
				a.lottery.selected = -1
				a.clearLotteryVerdict()
			})
		}),
		a.lottery.selected, func(i int) {
			a.lottery.selected = i
			a.lottery.verifyID = rows[i].ID
			// The previous verdict belongs to the previously chosen draw; keep
			// it from sitting beside a new id as if it described it.
			a.clearLotteryVerdict()
		})
}

// lotteryHistoryRecord projects a stored draw onto a record. The id is the
// title because it is what the verifier takes, and the winners are the fact
// an operator scans for.
func lotteryHistoryRecord(draw DrawRow, use func()) collectionRecord {
	return collectionRecord{
		title: draw.ID,
		facts: []collectionFact{
			{label: "Block", value: strconv.FormatInt(draw.BlockHeight, 10)},
			{label: "Winners", value: joinList(draw.Winners)},
		},
		fields: []collectionField{
			{label: "Participants", value: joinList(draw.Participants)},
		},
		action: func(c *ui.Context) {
			if secondaryButton(c, "Use this draw for Verify").Clicked() {
				use()
			}
		},
	}
}

func (a *App) lotteryVerify(c *ui.Context) {
	formCard(c, "Verify a Draw", "Recompute the winners from the stored VRF proof for any draw.", func() {
		current := strings.TrimSpace(a.lottery.verifyID)
		message := ""
		if a.lottery.verifyInput == current {
			message = a.lottery.verifyErr
		}
		fieldWithError(c, "Draw ID or block height", message, func() {
			ui.TextInput(c, &a.lottery.verifyID)
		})
		copyRow(c, "Draw ID", strings.TrimSpace(a.lottery.verifyID), nil)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if primaryButton(c, "Verify", allSet(a.lottery.verifyID)).Clicked() {
				a.verifyLottery()
			}
			// The newest recorded draw is one click away: the History table
			// already holds it, and asking an operator to copy its id to
			// re-check the draw they just ran was busywork.
			if len(a.lottery.rows) > 0 {
				if secondaryButton(c, "Use latest draw").Clicked() {
					a.lottery.verifyID = a.lottery.rows[0].ID
					a.verifyLottery()
				}
			}
		})
		if r := a.lottery.verifyResult; r != nil && a.lottery.verifyInput == current {
			// The verdict is a word first and a colour second: the tick or
			// cross is decoration for a reader who can see it.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if r.Valid {
					badge(c, "Verified", c.Theme().Success, c.Theme().Background)
				} else {
					badge(c, "Failed", c.Theme().Danger, c.Theme().Background)
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
