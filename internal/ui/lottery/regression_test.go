package lottery

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/stretchr/testify/assert"
)

// TestResultView_HandlesMismatchedWinnerAddresses is a regression test.
//
// Before the fix, resultView() indexed `m.result.WinnerAddresses[i]`
// without checking slice length. A record with more Winners than
// WinnerAddresses (which can happen with imported data, older DB schemas,
// or partial writes) would panic with index-out-of-range.
//
// This test exercises both:
//  1. The happy path (equal lengths)
//  2. The regression: mismatched lengths — must NOT panic.
func TestResultView_HandlesMismatchedWinnerAddresses(t *testing.T) {
	t.Run("equal lengths", func(t *testing.T) {
		app := NewLotteryApp()
		app.result = &lottery.LotteryRecord{
			ID:              "happy",
			Winners:         []string{"alice", "bob"},
			WinnerAddresses: []string{"addr1", "addr2"},
			VRFOutput:       "vrf-out",
			VRFProof:        "vrf-proof",
		}
		assert.NotPanics(t, func() {
			_ = app.resultView()
		})
	})

	t.Run("more winners than addresses (regression)", func(t *testing.T) {
		app := NewLotteryApp()
		app.result = &lottery.LotteryRecord{
			ID:              "broken",
			Winners:         []string{"a", "b", "c", "d"},
			WinnerAddresses: []string{"addr1"}, // short!
			VRFOutput:       "vrf-out",
			VRFProof:        "vrf-proof",
		}
		var view string
		assert.NotPanics(t, func() {
			view = app.resultView()
		})
		// Must still render something — i.e. the placeholder "no address"
		// path works. And all 4 winners must show up, not just the first.
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "a")
		assert.Contains(t, view, "d")
	})

	t.Run("nil winner addresses", func(t *testing.T) {
		app := NewLotteryApp()
		app.result = &lottery.LotteryRecord{
			ID:              "nil-addr",
			Winners:         []string{"only"},
			WinnerAddresses: nil,
			VRFOutput:       "vrf-out",
			VRFProof:        "vrf-proof",
		}
		assert.NotPanics(t, func() {
			_ = app.resultView()
		})
	})
}
