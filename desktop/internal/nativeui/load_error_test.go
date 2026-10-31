package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestDetailReadFailuresAreVisible covers three reads the fake can fail and no
// test had failed. They are not page loads: each is behind a button, and the
// card warns only once the read for the current id has failed. A card that
// swallowed the error would show an empty readout, which reads as "there is
// nothing there" rather than "the read did not work".
//
// The id fields are assigned inside the view closure on purpose: selecting a
// token resets the subject's inputs, so a value set before the first frame is
// gone by the time the button is drawn, and the button — which is disabled until
// its fields are set — swallows the click.
func TestDetailReadFailuresAreVisible(t *testing.T) {
	boom := errors.New("database is locked")

	t.Run("token info", func(t *testing.T) {
		app := New(&fakeServices{
			tokenRows: []TokenRow{{ID: "GLD", Name: "Gold", Symbol: "GLD", TotalSupply: "1000"}},
			loadErrs:  fakeLoadErrors{tokenInfo: boom},
		})
		app.ShowPage("token")
		tester := ui.NewTester(func(c *ui.Context) {
			app.token.tokenID = "GLD"
			app.View(c)
		}, 1280, 900)
		tester.Frame()
		require.NoError(t, tester.Click("Get Token Info"), "the read starts from its button")
		tester.Frame()
		require.True(t, tester.HasText("Couldn't load token info"), "a failed token read must warn")
	})

	t.Run("token balance", func(t *testing.T) {
		app := New(&fakeServices{
			tokenRows: []TokenRow{{ID: "GLD", Name: "Gold", Symbol: "GLD", TotalSupply: "1000"}},
			loadErrs:  fakeLoadErrors{tokenBalance: boom},
		})
		app.ShowPage("token")
		tester := ui.NewTester(func(c *ui.Context) {
			app.token.tokenID = "GLD"
			app.token.balanceAccount = "owner-key"
			app.View(c)
		}, 1280, 900)
		tester.Frame()
		require.NoError(t, tester.Click("Get Balance"), "the read starts from its button")
		tester.Frame()
		require.True(t, tester.HasText("Couldn't load balance"), "a failed balance read must warn")
	})

	t.Run("nft details", func(t *testing.T) {
		app := New(&fakeServices{
			nftRows:  []NFTRow{{ID: "nft-1", Name: "Star"}},
			loadErrs: fakeLoadErrors{nftDetails: boom},
		})
		app.ShowPage("nft")
		tester := ui.NewTester(func(c *ui.Context) {
			app.nft.id = "nft-1"
			app.View(c)
		}, 1280, 900)
		tester.Frame()
		require.NoError(t, tester.Click("Get NFT"), "the read starts from its button")
		tester.Frame()
		require.True(t, tester.HasText("Couldn't load NFT"), "a failed NFT read must warn")
	})
}

// TestChainStatusFailureIsVisible is the one read that runs on page entry: the
// integrity verdict is the page's headline, so a failure there must not render
// as "not checked yet".
func TestChainStatusFailureIsVisible(t *testing.T) {
	app := New(&fakeServices{loadErrs: fakeLoadErrors{chain: errors.New("database is locked")}})
	app.ShowPage("blockchain")
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	require.True(t, tester.HasText("Couldn't read the chain"), "a failed integrity read must name itself")
}
