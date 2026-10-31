package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEveryTokenTabRenders covers the five write forms behind the Token page's
// tabs. Only Burn (index 4) was ever rendered — the confirmation tests click it
// — so a form that panicked, or drew nothing, on Mint, Transfer, Approve or
// Transfer From would have shipped.
//
// The assertion is each form's own purpose line, not its tab label: the label
// is on screen whatever tab is selected, so asserting it would pass even if the
// body stopped drawing. The fake carries one token, because with none the page
// shows the first-record guide and every tab would look alike.
func TestEveryTokenTabRenders(t *testing.T) {
	for _, tc := range []struct {
		index   int
		purpose string
	}{
		{0, "Only the token owner can mint, and only for a mintable token."},
		{1, "Move tokens from one account to another."},
		{2, "Let a spender move up to an amount on the owner's behalf."},
		{3, "Spend an allowance the owner granted."},
		{4, "Destroy tokens from an account. This cannot be undone."},
	} {
		app := New(&fakeServices{tokenRows: []TokenRow{{ID: "GLD", Name: "Gold", Symbol: "GLD", TotalSupply: "1000"}}})
		app.ShowPage("token")
		app.token.opTab = tc.index
		tester := ui.NewTester(app.View, 1280, 900)
		tester.Frame()
		require.Truef(t, tester.HasText(tr(tc.purpose)),
			"token tab %d must draw its own form, not another tab's", tc.index)
	}
}

// TestEveryNFTTabRenders is the same for the NFT page's three forms, asserted on
// the card purpose. Index 0 is the transfer form, which no test had opened.
func TestEveryNFTTabRenders(t *testing.T) {
	for _, tc := range []struct {
		index   int
		purpose string
	}{
		{0, "Move an NFT to a new owner; the transfer is signed by the current one."},
		{1, "Destroy an NFT permanently. This cannot be undone."},
		{2, "Every mint, transfer and burn recorded for this NFT."},
	} {
		app := New(&fakeServices{nftRows: []NFTRow{{ID: "nft-1", Name: "Star"}}})
		app.ShowPage("nft")
		app.nft.opTab = tc.index
		tester := ui.NewTester(app.View, 1280, 900)
		tester.Frame()
		require.Truef(t, tester.HasText(tr(tc.purpose)),
			"nft tab %d must draw its own form", tc.index)
	}
}
