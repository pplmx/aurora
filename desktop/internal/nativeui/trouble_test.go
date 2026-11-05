package nativeui

import (
	"errors"
	"testing"
	"time"

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

// TestTheSidebarDrawsItselfForEachSignal covers the guard as one predicate. The
// block carries four unrelated signals now — the File menu, the version badge,
// the freshness stamp and the read-failure line — and each new one used to have
// to remember to widen the condition. Forgetting is silent: the feature simply
// never appeared in a window built without the shell, which is the window a view
// test builds.
func TestTheSidebarDrawsItselfForEachSignal(t *testing.T) {
	fresh := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)

	t.Run("nothing to say", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		assert.False(t, app.sidebarHasContent(),
			"an empty window with no stamp and no failure draws nothing")
	})

	t.Run("version badge", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.SetActions(Actions{Version: "0.0.1"})
		assert.True(t, app.sidebarHasContent(),
			"the version line is content even when the File menu is not wired")
	})

	t.Run("file menu", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.SetActions(Actions{ShowAbout: func() {}})
		assert.True(t, app.sidebarHasContent())
	})

	t.Run("freshness stamp", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.liveUpdates = true
		app.liveReadAt = fresh.Add(-3 * time.Second)
		assert.True(t, app.sidebarHasContent(),
			"a stamp with no File menu action is still content")
	})

	t.Run("read failure", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.token.loadErr = errors.New("database is locked")
		assert.True(t, app.sidebarHasContent(),
			"a failing page must be able to draw the line that reports it")
	})
}

// TestEachSignalIsActuallyDrawn proves the predicate is not merely consulted: the
// block is on screen for a window with no actions at all, which is the case the
// separate conditions each used to drop.
func TestEachSignalIsActuallyDrawn(t *testing.T) {
	fresh := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)

	t.Run("freshness stamp", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.SetActions(Actions{Version: "0.0.1"})
		tester := ui.NewTester(app.View, 1280, 900)
		app.liveUpdates = true
		app.liveReadAt = fresh.Add(-3 * time.Second)
		app.ShowPage("Dashboard")
		tester.Frame()
		assert.True(t, tester.HasText("Updated 3s ago"))
	})

	t.Run("read failure", func(t *testing.T) {
		app := New(workbenchFakes())
		app.liveNow = func() time.Time { return fresh }
		app.SetActions(Actions{Version: "0.0.1"})
		tester := ui.NewTester(app.View, 1280, 900)
		app.token.loadErr = errors.New("database is locked")
		app.ShowPage("Dashboard")
		tester.Frame()
		assert.True(t, tester.HasText("Couldn't read Token"))
	})
}

// TestTheTroubleLineFitsANarrowWindow covers the sidebar's smallest real width.
// The line is the one signal that has to be readable at a glance, and the
// counting choice exists because a list of names in this width would wrap.
func TestTheTroubleLineFitsANarrowWindow(t *testing.T) {
	app := New(workbenchFakes())
	app.liveNow = func() time.Time { return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC) }
	app.SetActions(Actions{Version: "0.0.1"})
	tester := ui.NewTester(app.View, 900, 600)
	app.token.loadErr = errors.New("database is locked")
	app.nft.loadErr = errors.New("database is locked")
	app.oracle.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	tester.Frame()

	require.Len(t, app.loadFailures(), 3)
	require.True(t, tester.HasText("3 pages couldn't be read"),
		"three failures are counted, and the count survives the narrow sidebar")
	require.False(t, tester.HasText("Couldn't read Token"),
		"naming one would hide the other two")
}

// TestThePaletteOffersTheFailingPage completes the discovery story. The sidebar
// line reports a failure, but the palette is where a reader goes when they cannot
// find something — and "the page is failing" answers a surprising number of those
// searches. The entry opens the page, so it reads as a destination.
func TestThePaletteOffersTheFailingPage(t *testing.T) {
	app := New(workbenchFakes())
	app.SetActions(Actions{Version: "0.0.1"})
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, tester.HasText("Open Token"),
		"the failing page is offered by name")
	require.True(t, tester.HasText("which could not be read"),
		"and the detail says why it is there")

	require.NoError(t, tester.Click("Open Token"))
	tester.Frame()
	require.Equal(t, pageToken, app.page, "the entry navigates to the page")
}

// TestThePaletteSaysNothingWhenEveryPageLoads keeps the entry out of a healthy
// window: an "open nothing" action would be noise in a list of commands.
func TestThePaletteSaysNothingWhenEveryPageLoads(t *testing.T) {
	app := New(workbenchFakes())
	app.SetActions(Actions{Version: "0.0.1"})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Dashboard")
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.False(t, tester.HasText("could not be read"),
		"a healthy window offers no failing page")
}
