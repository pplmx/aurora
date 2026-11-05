package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingPage drives one page into a read failure and then moves away, which is
// the state the sidebar line exists for: the page's own warning is right where
// the reader is looking, and the reader is somewhere else.
func failingPage(t *testing.T, target string) (*App, *ui.Tester) {
	t.Helper()
	fake := workbenchFakes()
	fake.loadErrs.tokens = errors.New("database is locked")
	fake.loadErrs.nfts = errors.New("database is locked")
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 900)
	app.SetActions(Actions{Version: "0.0.1"})
	app.ShowPage(target)
	tester.Frame()
	app.ShowPage("Dashboard")
	tester.Frame()
	return app, tester
}

// TestAFailingPageIsReportedFromEverywhere covers the gap the sidebar line fills.
// A page warns inside its own cards, which is right where that reader is looking
// — but a reader on Dashboard cannot see that the token page failed to load, and
// a page failing quietly looks exactly like a page with nothing to report.
func TestAFailingPageIsReportedFromEverywhere(t *testing.T) {
	app, tester := failingPage(t, "Token")

	failures := app.loadFailures()
	require.Len(t, failures, 1, "one page failed")
	assert.Equal(t, pageToken, failures[0].Page)
	assert.Equal(t, "Couldn't read Token", troubleLine(failures),
		"one page is named, not counted")
	require.True(t, tester.HasText("Couldn't read Token"),
		"and the line is visible from a different page")
}

// TestOneFailureIsNamedAndSeveralAreCounted keeps the line scannable. A list of
// names in a status line wraps and stops being readable at a glance, which is the
// one thing this line has to be.
func TestOneFailureIsNamedAndSeveralAreCounted(t *testing.T) {
	app, tester := failingPage(t, "Token")
	app.ShowPage("NFT")
	tester.Frame()

	failures := app.loadFailures()
	require.Len(t, failures, 2)
	assert.Equal(t, "2 pages couldn't be read", troubleLine(failures))
	require.True(t, tester.HasText("2 pages couldn't be read"))
}

// TestTheTroubleLineOpensTheFirstFailingPage makes the line actionable. A status
// line that only reports leaves the reader to find the page, which is the problem
// it was added to fix.
func TestTheTroubleLineOpensTheFirstFailingPage(t *testing.T) {
	app, tester := failingPage(t, "Token")
	require.Equal(t, pageDashboard, app.page)

	// The order is sidebar order, so the same state always leads to the same
	// page rather than to whichever field was checked first.
	app.ShowPage("Lottery")
	tester.Frame()
	app.openFirstFailure()
	assert.Equal(t, pageToken, app.page)
}

// TestAHealthyWindowHasNoTroubleLine keeps the line silent when nothing is wrong:
// a permanent "0 errors" is noise that trains a reader to ignore the position.
func TestAHealthyWindowHasNoTroubleLine(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 900)
	app.SetActions(Actions{Version: "0.0.1"})
	app.ShowPage("Token")
	tester.Frame()
	app.ShowPage("Dashboard")
	tester.Frame()

	assert.Empty(t, app.loadFailures())
	assert.Equal(t, "", troubleLine(nil))
	assert.False(t, tester.HasText("Couldn't read"))
}
