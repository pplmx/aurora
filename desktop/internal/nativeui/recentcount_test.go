package nativeui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// recentFixture builds a chain longer than the dashboard shows.
func recentFixture(chainLength int) *fakeServices {
	rows := make([]BlockRow, chainLength)
	for i := range rows {
		rows[i] = BlockRow{
			Height: int64(chainLength - i), Module: "token", Action: "Mint",
			Summary: fmt.Sprintf("mint-%02d", i),
		}
	}
	return &fakeServices{blocks: rows}
}

// TestTheRecentCardSaysHowMuchItIsNotShowing covers what a bare badge left
// unsaid. The card draws the newest few entries and labelled them with that
// count, so a five-entry preview of a thirty-entry chain looked like a five-entry
// chain — and nothing pointed at the rest.
func TestTheRecentCardSaysHowMuchItIsNotShowing(t *testing.T) {
	fake := recentFixture(30)
	app := New(fake)
	app.dashboard.Blocks = 30
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Dashboard")
	tester.Frame()

	require.Len(t, app.dashboard.recent, dashboardRecentLimit)
	require.Truef(t, tester.HasText(fmt.Sprintf("%d of 30", dashboardRecentLimit)),
		"the badge must say how many of the chain's entries are on screen")
}

// TestTheRecentCardDoesNotOverclaimOnAShortChain: when the card shows everything
// there is, the bare count is the whole truth and "5 of 5" would only add noise.
func TestTheRecentCardDoesNotOverclaimOnAShortChain(t *testing.T) {
	fake := recentFixture(dashboardRecentLimit)
	app := New(fake)
	app.dashboard.Blocks = dashboardRecentLimit
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Dashboard")
	tester.Frame()

	require.True(t, tester.HasText(fmt.Sprintf("%d", dashboardRecentLimit)))
	require.False(t, tester.HasText(fmt.Sprintf("%d of %d", dashboardRecentLimit, dashboardRecentLimit)),
		"a card showing the whole chain needs no ratio")
}
