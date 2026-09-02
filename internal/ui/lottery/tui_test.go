package lottery

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestNewLotteryApp(t *testing.T) {
	app := NewLotteryApp()
	assert.NotNil(t, app)
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, "3", app.count)
	assert.Equal(t, 0, app.menuIndex)
	assert.False(t, app.showHelp)
}

func TestModelInit(t *testing.T) {
	app := NewLotteryApp()
	cmd := app.Init()
	assert.Nil(t, cmd)
}

func TestViewMenuState(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewCreateState(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewHistoryState(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewResultStateNoResult(t *testing.T) {
	app := NewLotteryApp()
	app.view = "result"
	app.result = nil
	view := app.View()
	assert.NotNil(t, view)
}

func TestViewHelpState(t *testing.T) {
	app := NewLotteryApp()
	app.showHelp = true
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestMenuViewRenders(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	view := app.menuView()
	assert.NotEmpty(t, view)
}

func TestCreateViewRenders(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	view := app.createView()
	assert.NotEmpty(t, view)
}

func TestCreateViewWithError(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.err = "test error"
	view := app.createView()
	assert.Contains(t, view, "test error")
}

func TestCreateViewWithSuccess(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.successMsg = "test success"
	view := app.createView()
	assert.Contains(t, view, "test success")
}

func TestHistoryViewRenders(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	view := app.historyView()
	assert.NotEmpty(t, view)
}

func TestResultViewRendersWithNil(t *testing.T) {
	app := NewLotteryApp()
	app.result = nil
	view := app.resultView()
	assert.NotEmpty(t, view)
}

func TestResultViewRendersWithResult(t *testing.T) {
	app := NewLotteryApp()
	app.result = &lottery.LotteryRecord{
		ID:              "test-id",
		BlockHeight:     1,
		Winners:         []string{"winner1", "winner2"},
		WinnerAddresses: []string{"addr1", "addr2"},
		VRFOutput:       "vrf-output",
		VRFProof:        "vrf-proof",
	}
	view := app.resultView()
	assert.NotEmpty(t, view)
}

func TestRunLottery(t *testing.T) {
	app := NewLotteryApp()
	result := app.runLottery([]string{"p1", "p2", "p3", "p4", "p5"}, "seed", 2)
	assert.NotNil(t, result)
	assert.Len(t, result.Winners, 2)
}

// TestRunLotteryPersistsToRecorder pins the TASK-203 dual-write contract: a
// draw created in the TUI must land in the same lottery_records store the CLI
// history/stats/export/verify read, and the TUI history must render it.
func TestRunLotteryPersistsToRecorder(t *testing.T) {
	repo, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "lottery.db"))
	assert.NoError(t, err)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()

	app := NewLotteryApp()
	app.repo = repo

	result := app.runLottery([]string{"p1", "p2", "p3", "p4", "p5"}, "seed", 2)
	assert.NotNil(t, result)

	records, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, records, 1, "TUI draw must be persisted to lottery_records for the CLI surfaces")
	assert.Equal(t, result.ID, records[0].ID)

	// The TUI history view reads the recorder (not raw chain blocks), so the
	// freshly drawn record renders there.
	app.loadHistory()
	assert.Contains(t, app.viewport.View(), result.ID)
}

func TestHelpViewRenders(t *testing.T) {
	app := NewLotteryApp()
	view := app.helpView()
	assert.NotEmpty(t, view)
}

func TestParseTextArea(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single line",
			input:    "participant1",
			expected: []string{"participant1"},
		},
		{
			name:     "multiple lines",
			input:    "p1\np2\np3",
			expected: []string{"p1", "p2", "p3"},
		},
		{
			name:     "with empty lines",
			input:    "p1\n\np2\n",
			expected: []string{"p1", "p2"},
		},
		{
			name:     "with whitespace",
			input:    "  p1  \n  p2  ",
			expected: []string{"p1", "p2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTextArea(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s})
}

func TestUpdate_QuitFromMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("q"))
	assert.NotNil(t, cmd)
}

func TestUpdate_CtrlCFromMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("ctrl+c"))
	assert.NotNil(t, cmd)
}

func TestUpdate_ToggleHelp(t *testing.T) {
	app := NewLotteryApp()
	app.Update(keyPress("?"))
	assert.True(t, app.showHelp)
	app.Update(keyPress("?"))
	assert.False(t, app.showHelp)
	app.Update(keyPress("?"))
	assert.True(t, app.showHelp)
	app.Update(keyPress("esc"))
	assert.False(t, app.showHelp)
}

func TestUpdate_HelpModeExitsOnEsc(t *testing.T) {
	app := NewLotteryApp()
	app.showHelp = true
	_, cmd := app.Update(keyPress("esc"))
	assert.Nil(t, cmd)
	assert.False(t, app.showHelp)
}

func TestUpdate_HelpModeIgnoresOtherKeys(t *testing.T) {
	app := NewLotteryApp()
	app.showHelp = true
	_, cmd := app.Update(keyPress("q"))
	assert.Nil(t, cmd)
	assert.True(t, app.showHelp)
}

func TestUpdate_UpNavigation(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("up"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.menuIndex)
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_DownNavigation(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("j"))
	assert.Equal(t, 2, app.menuIndex)
	app.Update(keyPress("j"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_EnterCreatesView(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.menuIndex = 0
	app.participantsInput.SetValue("A\nB\nC\nD")
	app.seedInput.SetValue("test-seed")
	app.Update(keyPress("enter"))
	assert.Equal(t, "create", app.view)
}

func TestHandleCreateSuccessfully(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A B C D")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("1")
	msg := app.handleCreate()
	assert.Nil(t, msg)
	assert.Equal(t, "result", app.view)
	assert.NotNil(t, app.result)
	assert.NotEmpty(t, app.successMsg)
}

func TestUpdate_EnterOpensHistory(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "history", app.view)
}

func TestUpdate_EnterExitsFromMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.menuIndex = 2
	_, cmd := app.Update(keyPress("enter"))
	assert.NotNil(t, cmd)
}

func TestUpdate_NumericShortcuts(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.Update(keyPress("1"))
	assert.Equal(t, 0, app.menuIndex)
	app.Update(keyPress("2"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("3"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_EscReturnsToMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.err = "some error"
	app.successMsg = "some success"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, "", app.err)
	assert.Equal(t, "", app.successMsg)
}

func TestUpdate_EscDoesNothingFromMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "menu"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
}

// q must be typable inside the create form (the letter in "quack" etc), not
// a back-to-menu key there — the help screen scopes q to the menu (TASK-161,
// ISS-154). The focused participant input receives the "q" character.
func TestUpdate_QIsTypableInCreateForm(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.Update(keyPress("q"))
	assert.Equal(t, "create", app.view)
	assert.Equal(t, "q", app.participantsInput.Value())
}

// q still returns to the menu from read-only views (result shows a freshly
// drawn lottery; q is not a typable character there).
func TestUpdate_QReturnsToMenuFromResult(t *testing.T) {
	app := NewLotteryApp()
	app.view = "result"
	app.err = "x"
	app.successMsg = "y"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
	assert.Empty(t, app.err)
	assert.Empty(t, app.successMsg)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	app := NewLotteryApp()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	assert.Equal(t, 76, app.viewport.Width())
	assert.Equal(t, 12, app.viewport.Height())
}

func TestUpdate_EnterInCreateViewReturnsCmd(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A\nB\nC")
	app.seedInput.SetValue("seed")
	_, cmd := app.Update(keyPress("enter"))
	assert.NotNil(t, cmd)
}

func TestUpdate_HandleCreateTooFewParticipants(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("3")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestUpdate_HandleCreateEmptySeed(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A\nB\nC")
	app.seedInput.SetValue("")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestUpdate_EnterInHistoryReturnsToMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, "", app.successMsg)
}

func TestUpdate_EnterInResultReturnsToMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "result"
	app.result = &lottery.LotteryRecord{ID: "test"}
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

// Round-97 (TASK-123): the create form's text inputs never received
// keystrokes — typing did nothing and Enter always errored. These tests pin
// the fix: keypresses must reach the focused input, Tab/up/down must cycle
// focus, and the winner-count must be validated (no negative-count panic).

func TestUpdate_CreateFormReceivesKeystrokes(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 0
	app.updateInputFocus()

	app.Update(keyPress("A"))
	app.Update(keyPress("l"))
	app.Update(keyPress("i"))
	assert.Equal(t, "Ali", app.participantsInput.Value())
}

// TestUpdate_CreateFormJKTypable pins the TASK-161-class follow-up (ISS-164):
// the letters j and k are ordinary characters in the create form and must be
// typed (participants like "Jack"/"Jill", seeds with 'k'), NOT swallowed as
// up/down navigation. Arrow keys and Tab remain the form-navigation keys.
func TestUpdate_CreateFormJKTypable(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 0
	app.updateInputFocus()

	app.Update(keyPress("j"))
	app.Update(keyPress("a"))
	app.Update(keyPress("k"))
	assert.Equal(t, "jak", app.participantsInput.Value())
	assert.Equal(t, 0, app.inputFocus, "letter keys must not move form focus")
}

// TestUpdate_CreateFormQuestionMarkTypable scopes help to non-form views:
// "?" must be a typable character in the create form (seeds like "a?b"),
// not an unconditional help toggle.
func TestUpdate_CreateFormQuestionMarkTypable(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 1
	app.updateInputFocus()
	app.seedInput.SetValue("")

	app.Update(keyPress("?"))
	assert.Equal(t, "?", app.seedInput.Value())
	assert.False(t, app.showHelp, "? in a form must not open help (ISS-164)")
}

func TestUpdate_CreateFormSeedingAndCount(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 1
	app.updateInputFocus()
	app.Update(keyPress("s"))
	app.Update(keyPress("e"))
	assert.Equal(t, "se", app.seedInput.Value())

	app.inputFocus = 2
	app.updateInputFocus()
	app.countInput.SetValue("")
	app.Update(keyPress("2"))
	assert.Equal(t, "2", app.countInput.Value())
}

func TestUpdate_CreateFormTabCyclesFocus(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 0
	app.updateInputFocus()

	app.Update(keyPress("tab"))
	assert.Equal(t, 1, app.inputFocus)
	assert.False(t, app.participantsInput.Focused())
	assert.True(t, app.seedInput.Focused())

	app.Update(keyPress("tab"))
	assert.Equal(t, 2, app.inputFocus)
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_CreateFormUpDownCyclesFocus(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.inputFocus = 0
	app.updateInputFocus()

	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.inputFocus)
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestHandleCreate_NegativeCountRejected(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A,B,C")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("-1")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.NotEqual(t, "result", app.view)
}

func TestHandleCreate_ZeroCountRejected(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A,B,C")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("0")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.NotEqual(t, "result", app.view)
}

// A cleared/garbage winners field must be a visible error, not a silent
// 3-winner draw (TASK-165, ISS-158).
func TestHandleCreate_InvalidCountRejected(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A,B,C")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("")
	app.handleCreate()
	assert.Equal(t, i18n.GetText("lottery.tui.winners_invalid"), app.err)
	assert.NotEqual(t, "result", app.view)
}

func TestHandleCreate_EmptyParticipantsRejected(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("")
	app.seedInput.SetValue("seed")
	app.countInput.SetValue("1")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.NotEqual(t, "result", app.view)
}

func TestParseTextArea_CommaSeparated(t *testing.T) {
	got := parseTextArea("A,B,C")
	assert.Equal(t, []string{"A", "B", "C"}, got)
}

func TestParseTextArea_NewlineAndWhitespace(t *testing.T) {
	got := parseTextArea("A\nB\n  C  ")
	assert.Equal(t, []string{"A", "B", "C"}, got)
}

// Round-98 (TASK-127): the history view is a viewport with no scroll
// binding, so a long history was clipped at the 15-row view and
// unreachable. These tests pin that scrolling keys reach the viewport.

func TestUpdate_HistoryScrolling(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("1\n2\n3\n4\n5\n6\n7\n8")

	// Initially at the top; pressing "down"/"j" scrolls.
	y0 := app.viewport.YOffset()
	app.Update(keyPress("down"))
	assert.Greater(t, app.viewport.YOffset(), y0)
	app.Update(keyPress("j"))
	assert.Greater(t, app.viewport.YOffset(), y0)

	// Scrolling up returns toward the top.
	app.Update(keyPress("up"))
	assert.LessOrEqual(t, app.viewport.YOffset(), app.viewport.YOffset())
	app.Update(keyPress("k"))
	app.Update(keyPress("k"))
	assert.LessOrEqual(t, app.viewport.YOffset(), 2)
}

func TestUpdate_HistoryPgDnScrolls(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("1\n2\n3\n4\n5\n6\n7\n8")

	y0 := app.viewport.YOffset()
	app.Update(keyPress("pgdown"))
	assert.Greater(t, app.viewport.YOffset(), y0)
}

func TestUpdate_HistoryEnterStillReturnsToMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "history"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

// TASK-246 (ISS-248): the TUI create form previously enforced only loose
// ad-hoc checks (non-empty list, count>=1, count<=len, seed!=""), so draws the
// CLI/API reject — duplicate participant names, a seed shorter than
// MinSeedLength, a winner count above MaxWinners — were silently created here
// and then failed record.Validate() on re-import. These tests pin that the TUI
// now runs the same shared domain validators as the CLI/API.
func TestUpdate_HandleCreateRejectsDuplicateParticipants(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("Winner,Winner,Loser")
	app.seedInput.SetValue("valid-seed")
	app.countInput.SetValue("1")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.Equal(t, i18n.GetText("lottery.tui.duplicate_participant"), app.err)
	// A rejected draw must not advance to the result view.
	assert.Equal(t, "create", app.view)
}

func TestUpdate_HandleCreateRejectsShortSeed(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A,B,C")
	app.seedInput.SetValue("x") // < MinSeedLength=3
	app.countInput.SetValue("1")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.Equal(t, i18n.GetText("lottery.tui.seed_too_short"), app.err)
	assert.Equal(t, "create", app.view)
}

func TestUpdate_HandleCreateRejectsTooManyWinners(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue("A,B,C")
	app.seedInput.SetValue("valid-seed")
	app.countInput.SetValue("150") // > MaxWinners=100, but <= len? no — 150 > 3
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.Equal(t, "create", app.view)
}

func TestUpdate_HandleCreateRejectsOverCapWinners(t *testing.T) {
	// 1-char seed is below MinSeedLength but the domain MaxWinners=100 cap must
	// also be exercised with a valid seed: 101 winners on 101 participants.
	var parts []string
	for i := 0; i < 101; i++ {
		parts = append(parts, "P"+strconv.Itoa(i))
	}
	app := NewLotteryApp()
	app.view = "create"
	app.participantsInput.SetValue(strings.Join(parts, ","))
	app.seedInput.SetValue("valid-seed")
	app.countInput.SetValue("101")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
	assert.Equal(t, i18n.GetText("lottery.tui.winners_too_many"), app.err)
	assert.Equal(t, "create", app.view)
}
