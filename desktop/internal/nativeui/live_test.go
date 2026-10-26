package nativeui

import (
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
