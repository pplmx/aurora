package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTokenFormGeneratesAnOwnerKey covers the gap the CLI did not have: it
// generates an owner key pair when the flag is omitted, while the window used
// to require a key from somewhere else before a token could be created at all.
// The public half fills the field; the private half is offered once with Copy.
func TestTokenFormGeneratesAnOwnerKey(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Token")
	tester.Frame()

	require.NoError(t, tester.Click("Generate key pair"))
	tester.Frame()
	require.Equal(t, "pk-generated", app.token.owner)
	require.Equal(t, "sk-generated", app.token.ownerPriv)

	require.NoError(t, tester.Click("Copy Owner private key"))
	tester.Frame()
	require.Equal(t, "sk-generated", tester.Clipboard())

	require.NoError(t, tester.Click("Dismiss"))
	tester.Frame()
	require.Empty(t, app.token.ownerPriv, "Dismiss must remove the private key from the window")
}

// TestNFTFormGeneratesACreatorKey covers the same handover on the mint form.
func TestNFTFormGeneratesACreatorKey(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	require.NoError(t, tester.Click("Generate key pair"))
	tester.Frame()
	require.Equal(t, "pk-generated", app.nft.creator)
	require.Equal(t, "sk-generated", app.nft.creatorPriv)

	require.NoError(t, tester.Click("Copy Creator private key"))
	tester.Frame()
	require.Equal(t, "sk-generated", tester.Clipboard())
}
