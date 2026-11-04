package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTabSwitchesKeepThePageState holds what a reader expects when they glance at
// another tab and come back: the page is where they left it. The tabs are a view
// of one page's state, not a reset, so the search field and the highlight survive
// a round trip.
func TestTabSwitchesKeepThePageState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		page   page
		tabs   func(*App) *int
		search func(*App) *string
	}{
		{"Oracle", pageOracle, func(a *App) *int { return &a.oracle.tab }, func(a *App) *string { return &a.oracle.search }},
		{"Voting", pageVoting, func(a *App) *int { return &a.voting.tab }, func(a *App) *string { return &a.voting.voterSearch }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, 1280, 2400)
			app.ShowPage(pageTitles[tc.page])
			tester.Frame()

			*tc.search(app) = "needle"
			tester.Frame()

			for tab := range 3 {
				*tc.tabs(app) = tab
				tester.Frame()
			}
			*tc.tabs(app) = 0
			tester.Frame()

			require.Equal(t, "needle", *tc.search(app),
				"%s must keep what the reader typed while they looked at another tab", tc.name)
		})
	}
}

// TestSwitchingTabsDoesNotReadStorage: a tab is a different view of data the page
// already loaded, so flipping between them must not turn into a read per switch.
// The exceptions are reads a page makes once when it opens.
func TestSwitchingTabsDoesNotReadStorage(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Oracle")
	tester.Frame()

	before := storageReadCounters(fake)
	for _, tab := range []int{1, 2, 0, 2, 1, 0} {
		app.oracle.tab = tab
		tester.Frame()
	}
	after := storageReadCounters(fake)

	for name, count := range after {
		require.Equalf(t, before[name], count,
			"switching tabs read %s %d times; a tab shows data the page already loaded",
			name, count-before[name])
	}
}
