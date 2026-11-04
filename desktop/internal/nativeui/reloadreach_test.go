package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestReloadReachesEveryDataPage covers what Reload() promises. The File menu's
// import calls it, and a page it skipped would keep showing the rows it read
// before the import — the data would be stored and invisible, which reads as a
// failed import.
//
// Settings is the exception, and it is deliberate: it reads no collection, so
// there is nothing for a reload to refresh there.
func TestReloadReachesEveryDataPage(t *testing.T) {
	for _, p := range pageOrder {
		if p == pageSettings {
			continue
		}
		t.Run(pageTitles[p], func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			tester := ui.NewTester(app.View, 1280, 2200)
			app.ShowPage(pageTitles[p])
			tester.Frame()

			before := storageReadCounters(fake)
			app.Reload()
			tester.Frame()

			read := 0
			for name, count := range storageReadCounters(fake) {
				read += count - before[name]
			}
			require.Greaterf(t, read, 0,
				"Reload reads nothing on %s, so an import there would store rows the page never shows",
				pageTitles[p])
		})
	}
}

// TestImportIntoAnUnopenedPageStillAppears covers the path where the operator
// imports while another page is open. The importing page's reload cannot help
// them, so the data has to be there when they navigate to it — which it is,
// because opening a page reads it.
func TestImportIntoAnUnopenedPageStillAppears(t *testing.T) {
	fake := workbenchFakes()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2200)
	app.ShowPage("Settings")
	tester.Frame()
	require.Empty(t, app.lottery.rows)

	// The import lands in the store while Settings is open.
	fake.draws = []DrawRow{{
		ID: "d-1", BlockHeight: 3,
		Participants: []string{"Ada"}, Winners: []string{"Ada"},
	}}
	app.Reload()
	tester.Frame()

	app.ShowPage("Lottery")
	tester.Frame()
	require.Len(t, app.lottery.rows, 1,
		"opening the page after an import must show what was imported")
	require.True(t, tester.HasText("Ada"))
}
