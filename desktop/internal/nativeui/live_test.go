package nativeui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestLiveUpdatesRefreshTheOpenPage pins the opt-in timer: nothing runs while
// it is off, and an open dashboard reloads its counters when the interval due.
func TestLiveUpdatesRefreshTheOpenPage(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.liveNow = tester.Now
	app.ShowPage("Dashboard")
	tester.Frame()

	before := fake.dashboardCalls
	tester.Advance(liveRefreshInterval + time.Second)
	require.Equal(t, before, fake.dashboardCalls, "live updates are off by default")

	app.setLiveUpdates(true)
	tester.Frame()
	before = fake.dashboardCalls
	tester.Advance(liveRefreshInterval + time.Second)
	require.Greater(t, fake.dashboardCalls, before, "the due timer must reload the dashboard")
}

// TestLiveUpdatesRefreshOracleHealth covers the second page that opts in, and
// proves the timer does not fire while the user is on Settings.
func TestLiveUpdatesRefreshOracleHealth(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.liveNow = tester.Now
	app.setLiveUpdates(true)
	app.ShowPage("Oracle")
	tester.Frame()

	before := fake.oracleHealthCalls
	tester.Advance(liveRefreshInterval + time.Second)
	require.Greater(t, fake.oracleHealthCalls, before)

	// The same tick also re-reads a reading card the operator opened.
	app.oracle.tab = 2
	app.oracle.queryID = "source-1"
	tester.Frame()
	beforeData := fake.oracleDataCalls
	tester.Advance(liveRefreshInterval + time.Second)
	require.Greater(t, fake.oracleDataCalls, beforeData)

	app.ShowPage("Settings")
	tester.Frame()
	before = fake.oracleHealthCalls
	tester.Advance(liveRefreshInterval + time.Second)
	require.Equal(t, before, fake.oracleHealthCalls, "Settings must not start a background refresh")
}

// TestTheLiveUpdatesDescriptionNamesWhatItRefreshes keeps the Settings copy honest.
// The description listed the Dashboard, Oracle health and open readings, while the
// Oracle reload also refreshes the source list and the preset picker — a reader who
// turned it on to watch the health counters had no way to learn that their source
// list also updates.
//
// The check ties the sentence to the reload it describes, so widening the reload
// without widening the sentence fails here.
func TestTheLiveUpdatesDescriptionNamesWhatItRefreshes(t *testing.T) {
	reload := functionSource(t, readFile(t, "oracle.go"), "func (a *App) refreshOracleLive")
	require.Contains(t, reload, "a.loadOracle()",
		"the Oracle reload must refresh the source list")
	require.Contains(t, reload, "a.refreshOracleReads()",
		"and the readings that are open")

	settings := readFile(t, "settings.go")
	require.Contains(t, settings, "the Oracle source list, health and open readings",
		"the description must name the source list the reload refreshes")
}

// readFile reads a sibling source file. live_test.go works with a few of them and
// every helper takes a name rather than a path.
func readFile(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(name)
	require.NoError(t, err)
	return string(source)
}

// functionSource returns the body of one function, from its declaration to the next.
func functionSource(t *testing.T, src, declaration string) string {
	t.Helper()
	start := strings.Index(src, declaration)
	require.NotEqualf(t, -1, start, "%s must exist", declaration)
	rest := src[start+len(declaration):]
	if end := strings.Index(rest, "\nfunc "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

// TestSwitchingPagesRestartsTheLiveTimer keeps a page switch from reading a page
// twice. The timer used to carry its due time across the switch, so a reader who
// changed pages after ten seconds of waiting made the new page's first frame find
// it expired: the page's own load ran, then the timer fired immediately for the
// same data, costing a second query and a visible repaint for a refresh nobody
// asked for.
//
// Restarting on the switch matches setLiveUpdates, which resets the same field
// whenever the setting changes.
func TestSwitchingPagesRestartsTheLiveTimer(t *testing.T) {
	now := time.Now()
	fake := &fakeServices{
		sources: []SourceRow{{ID: "s1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true}},
	}
	app := New(fake)
	app.liveUpdates = true
	app.liveNow = func() time.Time { return now }
	tester := ui.NewTester(app.View, 1400, 3000)
	app.ShowPage("Dashboard")
	tester.Frame()

	// Past the interval, then switch without a frame in between: the switch is
	// what has to restart the timer.
	now = now.Add(liveRefreshInterval + time.Second)
	app.liveNow = func() time.Time { return now }
	app.ShowPage("Oracle")
	tester.Frame()

	require.Equal(t, 1, fake.sourceListCalls,
		"the page's own load must not be followed by a timer reload on the same frame")
	require.Greaterf(t, app.liveNext.Sub(now), time.Duration(0),
		"the timer must be armed for the new page, not already due")

	// And it still fires on time afterwards.
	before := fake.sourceListCalls
	now = now.Add(liveRefreshInterval)
	app.liveNow = func() time.Time { return now }
	tester.Frame()
	require.Greater(t, fake.sourceListCalls, before,
		"the timer must still fire once its interval passes on the new page")
}

// TestALiveRefreshLeavesTheOpenFormAlone covers the promise the reload's comment makes
// ("does not touch the add form or a read card the user has not opened"). A refresh runs
// every ten seconds, so a reload that reset the add form would erase a URL the reader was
// halfway through typing — and the erasure would look like a glitch, not a rule.
//
// The reload is the real one, not a stub: the point is what loadOracle does to state the
// form owns.
func TestALiveRefreshLeavesTheOpenFormAlone(t *testing.T) {
	fake := &fakeServices{
		sources: []SourceRow{{ID: "s1", Name: "feed", URL: "https://example.com/feed", Enabled: true}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Oracle")
	tester.Frame()

	// A form in progress, and a source the read cards are pointed at.
	app.oracle.addName = "half typed"
	app.oracle.addURL = "https://example.com/partial"
	app.oracle.addPath = "a.b"
	app.oracle.addHeaders = `{"X-Key":"v"}`
	app.oracle.fetchID, app.oracle.queryID = "s1", "s1"

	app.refreshOracleLive()
	tester.Frame()

	require.Equal(t, "half typed", app.oracle.addName, "the add form must survive a refresh")
	require.Equal(t, "https://example.com/partial", app.oracle.addURL)
	require.Equal(t, "a.b", app.oracle.addPath)
	require.Equal(t, `{"X-Key":"v"}`, app.oracle.addHeaders)
	require.Equal(t, "s1", app.oracle.fetchID, "and so must the chosen source")
	require.Equal(t, "s1", app.oracle.queryID)

	// The refresh did run: the counters were re-read.
	require.Greater(t, fake.oracleHealthCalls, 0, "the refresh must actually read health")
}

// TestTurningLiveUpdatesOnStartsAFullInterval covers the switch's timing, which the
// page-switch test covers for navigation but not for the setting itself. Both reset the
// same field, so both start a full interval: turning the setting on does not read the
// page at once, and turning it off and on again does not either.
//
// Waiting is the intended behaviour — the switch says "every 10 seconds", and a reload
// the moment it is flipped would read a page the reader has not looked at yet. The check
// exists because "arm for now" and "arm for now + interval" differ by one frame and
// nothing else, so a change to either would not be visible in the UI.
func TestTurningLiveUpdatesOnStartsAFullInterval(t *testing.T) {
	now := time.Now()
	fake := &fakeServices{
		sources: []SourceRow{{ID: "s1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true}},
	}
	app := New(fake)
	app.liveNow = func() time.Time { return now }
	tester := ui.NewTester(app.View, 1400, 3000)
	app.ShowPage("Oracle")
	tester.Frame()

	before := fake.oracleHealthCalls
	app.setLiveUpdates(true)
	tester.Frame()
	require.Equal(t, before, fake.oracleHealthCalls,
		"enabling the setting must not read the page on the same frame")
	require.Greater(t, app.liveNext.Sub(now), time.Duration(0),
		"the timer must be armed for later, not already due")

	// One full interval later it runs, and re-arms rather than firing every frame.
	now = now.Add(liveRefreshInterval)
	tester.Frame()
	require.Greater(t, fake.oracleHealthCalls, before, "the first interval must refresh")
	require.Greater(t, app.liveNext.Sub(now), time.Duration(0), "and re-arm")

	// Flipping it off and on waits again instead of refreshing immediately.
	app.setLiveUpdates(false)
	app.setLiveUpdates(true)
	tester.Frame()
	before = fake.oracleHealthCalls
	now = now.Add(time.Second)
	tester.Frame()
	require.Equal(t, before, fake.oracleHealthCalls,
		"a fresh enable starts its own interval")
}

// TestLiveRefreshWaitsForABackgroundWrite covers the one read path that ignored
// the busy gate. Refresh and the file commands are disabled while a write owns
// the database, and the palette drops them; the timer fired regardless, so a
// write that landed at the same moment was followed by a read of half-written
// state and then, when the write applied its own reload, by a second one.
//
// The tick is pushed out rather than dropped: the page is meant to stay current,
// and the write's own reload usually covers the interval anyway.
func TestLiveRefreshWaitsForABackgroundWrite(t *testing.T) {
	fake := &fakeServices{
		sources: []SourceRow{{ID: "s1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	app.liveNow = tester.Now
	app.setLiveUpdates(true)
	app.ShowPage("Oracle")
	tester.Frame()

	// A write owns the database, as runWrite leaves it.
	app.busy = "Fetching source…"
	tester.Frame()

	before := fake.sourceListCalls
	tester.Advance(3 * liveRefreshInterval)
	tester.Frame()
	require.Equal(t, before, fake.sourceListCalls,
		"the live timer must not read the database while a write owns it")

	app.busy = ""
	tester.Frame()
	before = fake.sourceListCalls
	tester.Advance(liveRefreshInterval + time.Second)
	tester.Frame()
	require.Greater(t, fake.sourceListCalls, before,
		"and it must resume once the write is done")
}
