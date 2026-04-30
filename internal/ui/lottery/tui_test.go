package lottery

import (
	"testing"

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
