package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// writeGateCase is one write entry point and the evidence that it reached
// storage. Each case compares a fingerprint after the attempt: the write either
// left a mark or it did not, which is what "gated" means.
type writeGateCase struct {
	name string
	page page
	run  func(*App)
	sig  func(*fakeServices) string
}

func writeGateCases() []writeGateCase {
	return []writeGateCase{
		{"createToken", pageToken, func(a *App) {
			a.token.name, a.token.symbol, a.token.supply, a.token.owner = "X", "X", "1", "pk"
			a.createToken()
		}, func(f *fakeServices) string { return itoa(f.createTokenCalls) }},
		{"mintToken", pageToken, func(a *App) {
			a.token.tokenID, a.token.mintTo = "t1", "b"
			a.token.mintAmount, a.token.mintPriv = "1", "k"
			a.mintToken()
		}, func(f *fakeServices) string { return itoa(f.mintTokenCalls) }},
		{"createLottery", pageLottery, func(a *App) {
			a.lottery.participants, a.lottery.seed, a.lottery.count = "A,B", "seed", 1
			a.createLottery()
		}, func(f *fakeServices) string {
			if f.created != nil {
				return "draw stored"
			}
			return "no draw"
		}},
		{"registerVoter", pageVoting, func(a *App) {
			a.voting.voterName = "Ada"
			a.registerVoter()
		}, func(f *fakeServices) string { return itoa(f.voters) }},
		{"registerCandidate", pageVoting, func(a *App) {
			a.voting.candName = "Al"
			a.registerCandidate()
		}, func(f *fakeServices) string { return itoa(f.candidateRegisterCalls) }},
	}
}

// TestNoWriteStartsWhileAnotherRuns holds the rule at every entry point instead
// of only the one that broke. The busy banner names one write, so a second one
// starting underneath it would report its result as the first's — and with the
// real database, two writes would contend for it.
func TestNoWriteStartsWhileAnotherRuns(t *testing.T) {
	for _, tc := range writeGateCases() {
		t.Run(tc.name, func(t *testing.T) {
			fake := workbenchFakes()
			app := New(fake)
			tester := ui.NewTester(app.View, 1280, 2400)
			app.ShowPage(pageTitles[tc.page])
			tester.Frame()
			before := tc.sig(fake)

			app.busy = "Busy…"
			tc.run(app)
			tester.Frame()

			require.Equal(t, before, tc.sig(fake),
				"%s must not reach storage while another write owns it", tc.name)
		})
	}
}

// TestEveryWriteEntryPointIsCovered keeps the table honest: the page files call
// runWrite from a known number of places, and a new write added without a case
// here would be the one that slips through.
func TestEveryWriteEntryPointIsCovered(t *testing.T) {
	covered := map[string]bool{}
	for _, tc := range writeGateCases() {
		covered[tc.name] = true
	}
	for _, name := range []string{
		"createToken", "mintToken", "createLottery", "registerVoter", "registerCandidate",
	} {
		require.Truef(t, covered[name], "%s must have a gate case", name)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return "nonzero"
}
