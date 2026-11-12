package nativeui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestLedgerBuildsOnlyTheRowsInView keeps the ledger on MyGo's virtual list.
// The Show control allows 10,000 rows; laying every record out each frame took
// about 450ms, so the page answered a keystroke half a second later. The list
// must build the rows in view, and scrolling it must bring the far end in.
func TestLedgerBuildsOnlyTheRowsInView(t *testing.T) {
	blocks := make([]BlockRow, 5000)
	for i := range blocks {
		blocks[i] = BlockRow{
			Height:  int64(len(blocks) - i),
			Module:  "token",
			Action:  "Transfer",
			Summary: fmt.Sprintf("entry-%05d", i),
		}
	}
	app := New(&fakeServices{blocks: blocks})
	tester := ui.NewTester(app.View, 1280, 860)
	app.ShowPage("Blockchain")
	app.blockchain.blockLimit = 10000
	app.loadBlockchain()
	tester.Frame()

	require.True(t, tester.HasText("entry-00000"), "the first record must be built")
	require.False(t, tester.HasText("entry-04999"),
		"a record far below the viewport must not be built")

	app.blockchain.blockList.ScrollTo(len(blocks)-1, ui.End)
	tester.Frame()
	require.True(t, tester.HasText("entry-04999"), "the list must scroll to its last record")
	require.False(t, tester.HasText("entry-00000"),
		"a record scrolled far above the viewport must not stay built")
}
