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

// TestTheWindowTitleSaysSomethingNeedsAttention carries the failure state out of
// the window. The sidebar line is only visible to a reader looking at the window;
// the native title is what a task switcher shows, so it is the one place a reader
// in another app can learn that something needs them.
func TestTheWindowTitleSaysSomethingNeedsAttention(t *testing.T) {
	var pushed []string
	app := New(workbenchFakes())
	app.SetActions(Actions{SetPageTitle: func(s string) { pushed = append(pushed, s) }})
	tester := ui.NewTester(app.View, 1280, 900)

	app.ShowPage("Token")
	tester.Frame()
	require.Equal(t, "Token", app.windowTitle, "a healthy window names the page alone")

	app.token.loadErr = errors.New("database is locked")
	tester.Frame()
	require.Equal(t, tr("Needs attention")+" — Token", app.windowTitle,
		"a failing page says so before the page name")

	app.token.loadErr = nil
	tester.Frame()
	require.Equal(t, "Token", app.windowTitle, "a recovered page drops the prefix")
}

// TestTheWindowTitleIsNotRepushedEveryFrame keeps a platform call off the repaint
// path. The frame runs for a hover or a caret blink as readily as for new data,
// and a task switcher does not need the same string set sixty times a second.
func TestTheWindowTitleIsNotRepushedEveryFrame(t *testing.T) {
	var pushes int
	app := New(workbenchFakes())
	app.SetActions(Actions{SetPageTitle: func(string) { pushes++ }})
	tester := ui.NewTester(app.View, 1280, 900)

	app.ShowPage("Token")
	tester.Frame()
	settled := pushes
	require.Positive(t, settled, "the first push happens")

	for i := 0; i < 5; i++ {
		tester.Frame()
	}
	require.Equal(t, settled, pushes,
		"an unchanged title is not pushed again, however many times the frame runs")

	app.token.loadErr = errors.New("database is locked")
	tester.Frame()
	require.Equal(t, settled+1, pushes, "a change is pushed exactly once")
}

// notifyRecorder captures the notices the view would raise.
type notifyRecorder struct {
	shown []NotifyOptions
}

func (r *notifyRecorder) actions(active bool) Actions {
	return Actions{
		Notify: func(o NotifyOptions) { r.shown = append(r.shown, o) },
		Active: func() bool { return active },
	}
}

// TestAFailureThatLastsRaisesOneNotice covers the case a notification is worth:
// without live updates a failed page does not retry itself, so it stays failed
// until the reader comes back — and the reader may not come back soon.
func TestAFailureThatLastsRaisesOneNotice(t *testing.T) {
	recorder := &notifyRecorder{}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app := New(workbenchFakes())
	app.liveNow = func() time.Time { return now }
	app.SetActions(recorder.actions(false))
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")

	tester.Frame()
	require.Empty(t, recorder.shown, "a failure that just appeared is not worth a notice")

	now = now.Add(troubleNoticeAfter - time.Second)
	tester.Frame()
	require.Empty(t, recorder.shown, "and it is still inside the threshold")

	now = now.Add(2 * time.Second)
	tester.Frame()
	require.Len(t, recorder.shown, 1, "a failure that outlasted the threshold is")
	require.Equal(t, "token", recorder.shown[0].Page,
		"the notice names the page so a click can open it")
	assert.Contains(t, recorder.shown[0].Body, "Token")

	// The outage continues; the notice must not.
	now = now.Add(time.Minute)
	tester.Frame()
	assert.Len(t, recorder.shown, 1, "a long outage notifies once, not every frame")
}

// TestAQuickFailureStaysQuiet guards the other half. A page that recovers on the
// next read — a locked database, a transient write — heals itself, and a notice
// for it is noise that trains the operator to ignore notices.
func TestAQuickFailureStaysQuiet(t *testing.T) {
	recorder := &notifyRecorder{}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app := New(workbenchFakes())
	app.liveNow = func() time.Time { return now }
	app.SetActions(recorder.actions(false))
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	tester.Frame()

	now = now.Add(troubleNoticeAfter + time.Second)
	app.token.loadErr = nil
	tester.Frame()
	assert.Empty(t, recorder.shown, "a failure that healed before the threshold says nothing")
}

// TestAFailureInFrontIsNotNotified keeps the notice from competing with the
// window it belongs to: the sidebar line and the page's own warning are already
// in front of the reader.
func TestAFailureInFrontIsNotNotified(t *testing.T) {
	recorder := &notifyRecorder{}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app := New(workbenchFakes())
	app.liveNow = func() time.Time { return now }
	app.SetActions(recorder.actions(true))
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	tester.Frame()

	now = now.Add(troubleNoticeAfter + time.Second)
	tester.Frame()
	assert.Empty(t, recorder.shown, "the front window already shows the failure")
}

// TestANoticeForAnotherPageRestartsTheClock keeps one outage from borrowing
// another's age: a second page failing is a new condition, and it has to outlast
// the threshold on its own.
func TestANoticeForAnotherPageRestartsTheClock(t *testing.T) {
	recorder := &notifyRecorder{}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	app := New(workbenchFakes())
	app.liveNow = func() time.Time { return now }
	app.SetActions(recorder.actions(false))
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	tester.Frame()

	now = now.Add(troubleNoticeAfter - time.Second)
	app.nft.loadErr = errors.New("database is locked")
	tester.Frame()
	require.Empty(t, recorder.shown, "the first page's age does not carry over")

	now = now.Add(2 * time.Second)
	tester.Frame()
	require.Len(t, recorder.shown, 1)
	assert.Equal(t, "token", recorder.shown[0].Page,
		"and the notice still names the topmost failing page")
}

// TestTheApplicationIconCountsFailingPages carries the count to the one surface a
// reader sees with the window completely hidden. The sidebar, the title and the
// palette all need the window to be somewhere visible; the application icon does
// not.
func TestTheApplicationIconCountsFailingPages(t *testing.T) {
	var pushes []int
	app := New(workbenchFakes())
	app.SetActions(Actions{
		Version: "0.0.1", SetPageTitle: func(string) {},
		SetBadgeCount: func(n int) { pushes = append(pushes, n) },
	})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Token")
	tester.Frame()
	require.Equal(t, []int{0}, pushes, "a healthy window clears the count")

	app.token.loadErr = errors.New("database is locked")
	tester.Frame()
	require.Equal(t, []int{0, 1}, pushes)

	// A second failing page leaves the title unchanged — it is already prefixed —
	// so a count that rode on the title's change check would freeze at one.
	app.nft.loadErr = errors.New("database is locked")
	tester.Frame()
	require.Equal(t, []int{0, 1, 2}, pushes,
		"the count follows the failures, not the title")

	for i := 0; i < 3; i++ {
		tester.Frame()
	}
	require.Equal(t, []int{0, 1, 2}, pushes,
		"an unchanged count is not pushed again, however many times the frame runs")

	app.token.loadErr, app.nft.loadErr = nil, nil
	tester.Frame()
	require.Equal(t, []int{0, 1, 2, 0}, pushes, "recovery clears it")
}

// TestTheBadgeIsSilentWithoutAShell keeps the view usable on its own: the hook is
// optional, and a view test (or a window built before the shell is wired) must not
// panic without it.
func TestTheBadgeIsSilentWithoutAShell(t *testing.T) {
	app := New(workbenchFakes())
	app.SetActions(Actions{Version: "0.0.1"})
	tester := ui.NewTester(app.View, 1280, 900)
	app.token.loadErr = errors.New("database is locked")
	app.ShowPage("Dashboard")
	require.NotPanics(t, func() { tester.Frame() })
	require.True(t, tester.HasText("Couldn't read Token"),
		"and the in-window report still works")
}

// TestAReloadDuringAWriteIsNotAPageFailure covers the guard Reload now shares with
// every in-window read path. A background write owns the database, so a reload
// landing on a locked one recorded the failure as a page that could not be read —
// and that failure then shows in the sidebar, the window title and the icon, so a
// transient lock would be reported as a broken page.
func TestAReloadDuringAWriteIsNotAPageFailure(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
	tester := ui.NewTester(app.View, 1280, 900)
	// Start from a page that read cleanly, so a failure recorded below can only
	// come from the reload under test.
	app.ShowPage("Token")
	tester.Frame()
	require.Empty(t, app.loadFailures(), "the baseline page loads")

	fake.loadErrs.tokens = errors.New("database is locked")
	app.busy = "Importing…"
	app.Reload()
	tester.Frame()
	require.Empty(t, app.loadFailures(),
		"a reload during a write must not be recorded as a page failure")

	// Once the write finishes, the same reload does read — and does report.
	app.busy = ""
	app.Reload()
	tester.Frame()
	require.Len(t, app.loadFailures(), 1,
		"a reload with no write running still reads and reports")
}

// TestAnUnreadPageDoesNotClaimTheStoreIsEmpty covers the state a page switch
// creates during a background write. Navigation is allowed while a write owns the
// database and the read is skipped, so the new page has no rows and no error —
// exactly the shape of an empty store. Printing "No tokens yet" there is a
// statement about a store the page never looked at.
func TestAnUnreadPageDoesNotClaimTheStoreIsEmpty(t *testing.T) {
	// A plain fake holds no tokens, so "No tokens yet" is exactly what an unread
	// page would say if the two states were not told apart.
	app := New(&fakeServices{})
	app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Dashboard")
	tester.Frame()

	app.busy = "Creating token…"
	app.ShowPage("Token")
	tester.Frame()

	require.False(t, tester.HasText("No tokens yet"),
		"an unread page must not describe the store it has not read")
	require.True(t, tester.HasText("Waiting for the running write"),
		"and it says why it is empty, which is the write holding the database")
	require.True(t, tester.HasText("Creating token…"),
		"the busy bar names the write that owns the database")

	// Once the write finishes the page reads, and the real state shows.
	app.busy = ""
	app.refresh()
	tester.Frame()
	require.True(t, app.token.loaded, "the read happened")
	require.True(t, tester.HasText("No tokens yet"),
		"and the store really is empty, so the guide is right to say so")
}

// TestAReadEmptyStoreStillSaysSo guards the other side: the change must not hide
// the empty state for a store that really is empty.
func TestAReadEmptyStoreStillSaysSo(t *testing.T) {
	// A plain fake holds no tokens, so the page's first read really is empty.
	app := New(&fakeServices{})
	app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Token")
	tester.Frame()

	require.True(t, app.token.loaded)
	require.True(t, tester.HasText("No tokens yet"),
		"a store that was read and is empty still says so")
}

// TestEveryCollectionPageWaitsInsteadOfClaimingEmpty generalises the token rule
// to the other two pages that have a first-run guide. The state is the same:
// navigation is allowed during a background write, the read is skipped, and the
// new page has neither rows nor an error.
func TestEveryCollectionPageWaitsInsteadOfClaimingEmpty(t *testing.T) {
	pages := []struct {
		name string
		id   string
		text string
	}{
		{"NFT", "NFT", "No NFTs yet"},
		{"Oracle", "Oracle", "No sources yet"},
		// The voting page has its own first-run guide and three lists, so it
		// had three empty sentences to withhold rather than one.
		{"Voting", "Voting", "No voters yet"},
	}
	for _, tc := range pages {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{})
			app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
			tester := ui.NewTester(app.View, 1280, 900)
			app.ShowPage("Dashboard")
			tester.Frame()

			app.busy = "Creating…"
			app.ShowPage(tc.id)
			tester.Frame()
			require.Falsef(t, tester.HasText(tc.text),
				"%s must not describe a store it has not read", tc.name)
			require.False(t, tester.HasText("No voters registered yet."),
				"nor may a list inside it")
			require.True(t, tester.HasText("Waiting for the running write"),
				"and it says why")

			// The store really is empty here, so the guide is right after a read.
			app.busy = ""
			app.refresh()
			tester.Frame()
			require.Truef(t, tester.HasText(tc.text),
				"%s says so once it has actually read", tc.name)
		})
	}
}

// TestASearchDuringAWriteDoesNotRead covers the path that bypassed the guard.
// refresh gates on the write, but the search field reads on its own when a term
// changes — the page has to show the new query's rows — and that read landed on
// the locked database, which reported a page failure and marked the page as read
// when it had not been.
func TestASearchDuringAWriteDoesNotRead(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	app.SetActions(Actions{Version: "0.0.1", SetPageTitle: func(string) {}, SetBadgeCount: func(int) {}})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Dashboard")
	tester.Frame()

	app.busy = "Creating…"
	app.token.search = "gold"
	app.ShowPage("Token")
	tester.Frame()

	require.False(t, app.token.loaded,
		"the page did not read, so it must not claim it did")
	require.Empty(t, app.loadFailures(),
		"and a read the write forbade is not a failure to report")
	require.True(t, tester.HasText("Waiting for the running write"),
		"the page says what it is waiting for")

	// With no write running the same term reads and applies.
	app.busy = ""
	app.refresh()
	tester.Frame()
	require.True(t, app.token.loaded)
	require.Equal(t, "gold", app.token.appliedSearch, "the pending term is applied")
}
