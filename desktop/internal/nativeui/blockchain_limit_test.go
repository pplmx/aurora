package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestBlockchainBlockLimitClampsThePageSize pins the page size the ledger card
// asks for. The bound is not cosmetic: the value comes from a control the
// operator moves, and it is passed straight to a query, so an unbounded value
// would fetch the whole chain into a frame.
func TestBlockchainBlockLimitClampsThePageSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  float64
		want int
	}{
		{"zero falls back to the floor", 0, 10},
		{"a negative value is the floor", -5, 10},
		{"just under the floor", 9, 10},
		{"the floor itself", 10, 10},
		{"a value inside the range", 50, 50},
		{"the ceiling itself", 10000, 10000},
		{"just over the ceiling", 10001, 10000},
		{"far over the ceiling", 1e9, 10000},
		{"a fractional value truncates", 42.9, 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equalf(t, tc.want, blockchainBlockLimit(tc.raw),
				"blockchainBlockLimit(%v)", tc.raw)
		})
	}
}

// TestLedgerCardTellsUncheckedFromBroken keeps three states apart that would
// otherwise look alike. A page that was never verified must ask for a refresh
// rather than imply the chain is fine, and a broken chain must say so instead of
// leaving the counters to stand alone as if they were a verdict.
func TestLedgerCardTellsUncheckedFromBroken(t *testing.T) {
	t.Run("never checked", func(t *testing.T) {
		// The page loads as soon as it is shown, so an unverified ledger is the
		// state a window is in before its services are wired. Building the view
		// that way is what exercises the branch.
		app := New(nil)
		tester := ui.NewTester(app.View, 1280, 1200)
		app.ShowPage("Blockchain")
		tester.Frame()

		require.False(t, app.blockchain.checked, "the probe must reach the unverified state")
		require.True(t, tester.HasText("Press Refresh to verify the chain."),
			"an unverified page must ask for the check it has not run")
		require.False(t, tester.HasText("Hash chain verified"),
			"nothing may claim the chain is intact before it was checked")
	})

	t.Run("verified", func(t *testing.T) {
		app := New(&fakeServices{blocks: captureBlocks(), ledgerHead: "009c4a0820ab"})
		tester := ui.NewTester(app.View, 1280, 1200)
		app.ShowPage("Blockchain")
		tester.Frame()

		require.True(t, tester.HasText("Hash chain verified"))
		require.True(t, tester.HasText("009c4a0820ab"), "the head is the chain's identity")
	})

	t.Run("broken", func(t *testing.T) {
		app := New(&fakeServices{blocks: captureBlocks()})
		tester := ui.NewTester(app.View, 1280, 1200)
		app.ShowPage("Blockchain")
		app.blockchain.valid = false
		app.blockchain.checked = true
		tester.Frame()

		require.True(t, tester.HasText("Hash chain is broken"),
			"a broken chain must say so, not leave the counters to imply health")
		require.False(t, tester.HasText("Hash chain verified"))
	})
}
