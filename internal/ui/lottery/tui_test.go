package lottery

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/domain/lottery"
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

func TestUpdate_QReturnsToMenu(t *testing.T) {
	app := NewLotteryApp()
	app.view = "create"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
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
