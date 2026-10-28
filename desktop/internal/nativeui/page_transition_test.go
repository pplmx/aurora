package nativeui

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPageSwitchPreservesDestinationLayout pins page identity on the shared
// scroll container. Without it, Oracle's taller tree was reconciled into NFT's
// tree: the first visit to NFT after Oracle clipped the Mint button at 1.5x
// text.
func TestPageSwitchPreservesDestinationLayout(t *testing.T) {
	newApp := func() *App {
		return New(&fakeServices{
			nftRows: []NFTRow{{
				ID: "nft-aurora-111111111111111111111111", Name: "Northern Lights",
				Description: "Aurora over the ledger.",
				Owner:       "pk-owner-11111111111111111111111111111111",
				Creator:     "pk-creator-111111111111111111111111111111",
				ImageURL:    "https://example.com/aurora.png",
				TokenURI:    "ipfs://bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
				BlockHeight: 9,
			}},
			sources: []SourceRow{{
				ID: "source-aurora-111111111111111111111111", Name: "btc-price",
				URL: "https://api.example.com/v1/prices/bitcoin",
			}},
		})
	}
	newTester := func(app *App) *ui.Tester {
		tester := ui.NewTester(app.View, 900, 2200)
		tester.SetPreferences(ui.Preferences{TextScale: 1.5})
		return tester
	}

	directApp := newApp()
	direct := newTester(directApp)
	directApp.ShowPage("NFT")
	direct.Frame()
	direct.Frame()

	switchedApp := newApp()
	switched := newTester(switchedApp)
	switchedApp.ShowPage("Oracle")
	switched.Frame()
	switchedApp.ShowPage("NFT")
	switched.Frame()
	switched.Frame()

	want, got := direct.Image(), switched.Image()
	require.Equal(t, want.Bounds(), got.Bounds())
	// Both frames come from the same test process and theme, so an exact match
	// is stable without committing a platform-dependent golden image.
	require.True(t, bytes.Equal(want.Pix, got.Pix),
		"Oracle -> NFT must render exactly like opening NFT directly")
}
