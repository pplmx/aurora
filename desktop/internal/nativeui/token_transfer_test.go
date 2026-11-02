package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// transferApp fills the Transfer form and returns the app, so each test only
// describes what it is about.
func transferApp(t *testing.T, err error) (*App, *fakeServices) {
	t.Helper()
	fake := &fakeServices{transferErr: err}
	app := New(fake)
	ui.NewTester(app.View, 1280, 1400).Frame()
	app.ShowPage("Token")
	app.token.tokenID = "GLD"
	app.token.xFrom = "sender-pk"
	app.token.xTo = "recipient-pk"
	app.token.xAmount = "40"
	app.token.xPriv = "sender-priv"
	return app, fake
}

// TestTransferPassesTheFormToTheService pins what the button sends: the four
// fields and the key, trimmed, in the order the use case expects.
func TestTransferPassesTheFormToTheService(t *testing.T) {
	app, fake := transferApp(t, nil)
	app.token.tokenID = "  GLD  "
	app.token.xFrom = " sender-pk "

	app.transferToken()

	require.Equal(t, 1, fake.transferTokenCalls)
	require.Equal(t, [5]string{"GLD", "sender-pk", "recipient-pk", "40", "sender-priv"}, fake.lastTransfer,
		"the service gets trimmed values, and the key it must sign with")
}

// TestSuccessfulTransferClearsTheKey is the security half: a signing key that
// worked is not kept on screen, because the next write needs its own and a
// shared display should not retain the secret.
func TestSuccessfulTransferClearsTheKey(t *testing.T) {
	app, _ := transferApp(t, nil)

	app.transferToken()

	require.Empty(t, app.token.xPriv, "the key is cleared after a successful write")
	require.False(t, app.token.status.isErr, "a completed write is not an error")
	require.Equal(t, "Done.", app.token.status.text, "the operator is told the write landed")
}

// TestFailedTransferKeepsTheKeyForARetry is the other half of the same rule: a
// rejected transfer must leave the form as the operator typed it, or the retry
// costs a re-paste of the key.
func TestFailedTransferKeepsTheKeyForARetry(t *testing.T) {
	app, _ := transferApp(t, errors.New("insufficient balance"))

	app.transferToken()

	require.Equal(t, "sender-priv", app.token.xPriv,
		"a failed transfer keeps the key, so the retry does not need it re-entered")
	require.True(t, app.token.status.isErr)
	require.Contains(t, app.token.status.text, "insufficient balance")
}

// TestTransferRefusesAnIncompleteForm covers the guard: a missing field is a
// refusal with a sentence naming what is needed, not a request with blanks.
func TestTransferRefusesAnIncompleteForm(t *testing.T) {
	app, fake := transferApp(t, nil)
	app.token.xAmount = "   "

	app.transferToken()

	require.Zero(t, fake.transferTokenCalls, "an incomplete form must not reach the service")
	require.True(t, app.token.status.isErr)
	require.Contains(t, app.token.status.text, "Token ID, sender, recipient, amount, and private key are required.")
}

// TestEveryTokenWriteRefusesAnIncompleteForm covers the four write forms on the
// token page. Each has its own fields and its own guard, and the guard is the only
// thing between an incomplete form and a signed write. The assertion is the call
// counter: a message drawn beside a call that went out anyway still reads as "I
// was stopped", so the status text alone cannot show the guard held.
//
// Each case fills every field but one, so a guard that checked only some of them
// still fails here.
func TestEveryTokenWriteRefusesAnIncompleteForm(t *testing.T) {
	cases := []struct {
		name  string
		calls func(*fakeServices) int
		fill  func(*App)
	}{
		{
			name: "create", calls: func(f *fakeServices) int { return f.createTokenCalls },
			fill: func(a *App) {
				// The owner is left out; the rest of the form is filled. Its guard
				// reports through the page status, unlike the field-level checks.
				a.token.name, a.token.symbol, a.token.supply = "Gold", "GLD", "1000"
				a.token.owner = ""
				a.createToken()
			},
		},
		{
			name: "mint", calls: func(f *fakeServices) int { return f.mintTokenCalls },
			fill: func(a *App) {
				a.token.tokenID, a.token.mintTo = "t1", "recipient"
				a.token.mintAmount, a.token.mintPriv = "10", ""
				a.mintToken()
			},
		},
		{
			name: "approve", calls: func(f *fakeServices) int { return f.approveTokenCalls },
			fill: func(a *App) {
				a.token.tokenID, a.token.approveOwner = "t1", "owner"
				a.token.approveTo, a.token.approveAmt = "", "5"
				a.token.approvePriv = "priv"
				a.approveToken()
			},
		},
		{
			name: "transfer from", calls: func(f *fakeServices) int { return f.transferFromTokenCalls },
			fill: func(a *App) {
				a.token.tokenID, a.token.fromOwner = "t1", "owner"
				a.token.fromTo, a.token.fromAmount = "recipient", "5"
				a.token.fromPriv = ""
				a.transferFromToken()
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeServices{}
			app := New(fake)
			tc.fill(app)
			require.False(t, app.token.status.empty(), "the form must report what is missing")
			require.True(t, app.token.status.isErr, "as a failure, not a success")
			require.Zero(t, tc.calls(fake), "an incomplete form must not reach the service")
		})
	}
}

// TestEveryNFTWriteRefusesAnIncompleteForm is the same contract for the asset
// page: minting without a creator and transferring without a key are refused
// before the service is asked.
func TestEveryNFTWriteRefusesAnIncompleteForm(t *testing.T) {
	t.Run("mint", func(t *testing.T) {
		fake := &fakeServices{}
		app := New(fake)
		app.nft.name, app.nft.creator = "Star", ""
		app.mintNFT()
		require.True(t, app.nft.status.isErr)
		require.Zero(t, fake.mintNFTCalls)
	})

	t.Run("transfer", func(t *testing.T) {
		fake := &fakeServices{}
		app := New(fake)
		app.nft.transferID, app.nft.from = "n1", "owner"
		app.nft.to, app.nft.priv = "recipient", ""
		app.transferNFT()
		require.True(t, app.nft.status.isErr)
		require.Zero(t, fake.transferNFTCalls)
	})
}
