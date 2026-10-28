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

	require.NoError(t, tester.Click("Dismiss key"))
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

// TestDismissingGeneratedKeyRestoresFocus covers both key cards: the focused
// Dismiss control removes itself with the private half, so focus must return to
// the Generate action that remains rather than falling to the window root.
func TestDismissingGeneratedKeyRestoresFocus(t *testing.T) {
	for _, tc := range []struct {
		name    string
		page    page
		private func(*App) string
	}{
		{"token owner", pageToken, func(app *App) string { return app.token.ownerPriv }},
		{"NFT creator", pageNFT, func(app *App) string { return app.nft.creatorPriv }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, 1280, 4000)
			app.ShowPage(pageIDs[tc.page])
			tester.Frame()

			for range 60 {
				if tester.Focused(tr("Generate key pair")) {
					break
				}
				tester.Key(0, ui.KeyTab)
				tester.Frame()
			}
			require.True(t, tester.Focused(tr("Generate key pair")))
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			require.Equal(t, "sk-generated", tc.private(app))
			require.True(t, tester.Focused(tr("Generate key pair")),
				"inserting the key card must keep focus on Generate")

			tester.Key(0, ui.KeyTab)
			tester.Frame()
			require.True(t, tester.Focused(tr("Dismiss key")))
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			require.Empty(t, tc.private(app))
			require.True(t, tester.Focused(tr("Generate key pair")),
				"removing the key card must return focus to Generate")
		})
	}
}
