package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEveryPageReportsAFailedRead covers the state a page is in when storage
// refuses. A page that renders its empty state instead would tell the operator
// there is nothing stored — a different and much worse claim than "the read
// failed", because the one fact they came to check is being withheld.
//
// Every page is exercised rather than a sample: each one owns its own error
// field, so a page added without a display path would otherwise look fine.
func TestEveryPageReportsAFailedRead(t *testing.T) {
	failure := errors.New("disk I/O error")

	for _, tc := range []struct {
		page  page
		arm   func(*fakeServices)
		empty string
	}{
		{pageToken, func(f *fakeServices) { f.loadErrs.tokens = failure }, "No tokens yet"},
		{pageNFT, func(f *fakeServices) { f.loadErrs.nfts = failure }, "No NFTs yet"},
		{pageVoting, func(f *fakeServices) { f.loadErrs.voters = failure }, "No voters registered yet"},
		{pageOracle, func(f *fakeServices) { f.loadErrs.sources = failure }, "No sources registered"},
		{pageBlockchain, func(f *fakeServices) { f.loadErrs.blocks = failure }, "No module records yet"},
		{pageDashboard, func(f *fakeServices) { f.loadErrs.stats = failure }, ""},
		{pageLottery, func(f *fakeServices) { f.loadErrs.history = failure }, "No draws recorded yet"},
	} {
		t.Run(pageTitles[tc.page], func(t *testing.T) {
			fake := workbenchFakes()
			tc.arm(fake)
			app := New(fake)
			tester := ui.NewTester(app.View, 1280, 2200)
			app.ShowPage(pageTitles[tc.page])
			tester.Frame()

			require.Truef(t, tester.HasText(failure.Error()),
				"%s must report the failure the operator can act on", pageTitles[tc.page])
			if tc.empty != "" {
				require.Falsef(t, tester.HasText(tc.empty),
					"%s must not claim the store is empty when the read failed", pageTitles[tc.page])
			}
		})
	}
}
