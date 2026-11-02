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
