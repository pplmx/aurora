package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestGuideNamesControlsThatExist keeps the guidance honest. Every step tells
// the reader where to go, and it does that by naming a card, a tab or a button:
// "Open Ballot", "press Verify", "A spender's remaining allowance is in
// Allowance". A renamed control leaves the sentence pointing at nothing, and
// nothing fails — the guide still reads well, it just no longer matches the
// page it is attached to.
//
// The list is kept by hand rather than parsed out of the sentences: guessing
// which capitalised word is a control name would flag "Aurora", "NFT" and
// "JSON" too, and the test would then be maintained by suppressing its own
// findings. Adding a step that points at a new control means adding it here.
func TestGuideNamesControlsThatExist(t *testing.T) {
	pages := []struct {
		page     string
		controls []string
	}{
		{"Lottery", []string{"History", "Verify", "Count"}},
		{"NFT", []string{"Collection", "Transfer", "Burn", "History", "Inspect"}},
		{"Oracle", []string{"Advanced", "Sources", "Health", "Readings"}},
		// "Use generated owner key" is deliberately absent: it only appears once
		// a key has been generated for this token, and the step says so.
		{"Token", []string{"Tokens", "Mint", "Transfer", "Approve", "Transfer From", "Burn", "Allowance"}},
		{"Voting", []string{"Setup", "Ballot", "Results"}},
		{"Blockchain", []string{"Payload", "Show"}},
	}

	for _, tc := range pages {
		t.Run(tc.page, func(t *testing.T) {
			// Seeded, because a page with no records leads with its first-record
			// path instead of the guide. The guide and the cards it names appear
			// together, which is the state this test is about.
			app := New(&fakeServices{
				blocks:     captureBlocks(),
				draws:      []DrawRow{{ID: "draw-1", BlockHeight: 3}},
				nftRows:    []NFTRow{{ID: "nft-1", Name: "Star", Owner: "owner-pk"}},
				sources:    []SourceRow{{ID: "src-1", Name: "btc", URL: "https://x", Enabled: true}},
				tokenRows:  []TokenRow{{ID: "token-1", Name: "Gold", Symbol: "GLD"}},
				voterRows:  []VoterRow{{PublicKey: "voter-pk", Name: "Ada"}},
				candidates: []CandidateRow{{ID: "cand-1", Name: "Ada"}},
				sessions:   []SessionRow{{ID: "sess-1", Title: "Budget", Status: "active"}},
			})
			tester := ui.NewTester(app.View, 1280, 4000)
			app.ShowPage(tc.page)
			tester.Frame()

			for _, control := range tc.controls {
				require.Truef(t, tester.HasText(control),
					"%s: the guide names %q, but nothing on the page draws it", tc.page, control)
			}
		})
	}
}
