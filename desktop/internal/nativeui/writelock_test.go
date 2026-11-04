package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestABusyWindowStillLetsTheReaderNavigate holds the arrangement the busy bar
// exists for. A write can take seconds, and the operator who started it may need
// to check something else meanwhile; the page body is disabled, but the sidebar is
// deliberately not, and the result reports on whatever page they are looking at.
func TestABusyWindowStillLetsTheReaderNavigate(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()

	app.busy = "Creating token…"
	app.busyLabel = "Creating token…"
	tester.Frame()

	require.NoError(t, tester.Click("Dashboard"), "the sidebar stays usable during a write")
	tester.Frame()
	require.Equal(t, pageDashboard, app.page)
}

// TestASecondWriteWaitsForTheFirst: one write owns the database at a time. The
// second is dropped rather than queued, because the busy bar names the write that
// is running and a queued one would run behind a result the operator already read
// as final.
func TestASecondWriteWaitsForTheFirst(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()
	before := fake.createTokenCalls

	app.busy = "Creating token…"
	app.busyLabel = "Creating token…"
	tester.Frame()
	app.token.name, app.token.symbol, app.token.supply = "X", "X", "1"
	app.token.owner = "pk"
	app.createToken()
	tester.Frame()

	require.Equal(t, before, fake.createTokenCalls,
		"a write attempted during another must not reach storage")
	require.NotEmpty(t, app.busy, "and must not clear the running write's banner")
}

// TestTheBusyBarFollowsTheReaderAcrossEveryPage generalises the arrangement to
// where the reader can go. A write can take seconds and the sidebar stays usable
// during it, so the banner must be a fact about the window rather than about one
// page — otherwise moving away from the form looks like the write ended.
func TestTheBusyBarFollowsTheReaderAcrossEveryPage(t *testing.T) {
	for _, target := range pageOrder {
		t.Run(pageTitles[target], func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, 1280, 2400)
			app.ShowPage("Token")
			tester.Frame()

			app.busy = "Creating token…"
			app.busyLabel = "Creating token…"
			tester.Frame()

			app.ShowPage(pageTitles[target])
			tester.Frame()

			require.Equal(t, target, app.page, "the write must not block navigation")
			require.Truef(t, tester.HasText("Creating token…"),
				"the busy bar must still name the write on %s", pageTitles[target])
		})
	}
}
