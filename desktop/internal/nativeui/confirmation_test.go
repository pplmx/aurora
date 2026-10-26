package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestBurnDialogNamesWhatItDestroys pins the confirmation's detail. The bare
// question "Burn tokens?" asked the operator to trust whichever amount the
// form happened to hold; the amount and the account are now in the sentence,
// so a wrong value can be caught before an irreversible write.
func TestBurnDialogNamesWhatItDestroys(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.burnFrom = "owner-pk"
	app.token.burnAmount = "25"
	require.NoError(t, tester.Click("Burn"), "select the Burn operation")
	tester.Frame()
	require.NoError(t, tester.Click("Burn…"))
	tester.Frame()
	require.True(t, tester.HasText("Burn 25 from owner-pk?"),
		"the confirmation must name the amount and the account")

	require.NoError(t, tester.Click("Cancel"))
	tester.Frame()
	require.False(t, app.token.askBurn, "Cancel must close the dialog")
}

// TestNFTBurnDialogNamesTheAsset covers the same contract for an asset: its
// name when the Inspect card loaded it, the shortened id otherwise.
func TestNFTBurnDialogNamesTheAsset(t *testing.T) {
	const id = "5c43a418-ca80-46ba-af63-cc7893d1a5e3"
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.burnID = id
	app.nft.nft = &NFTRow{ID: id, Name: "Star"}
	require.NoError(t, tester.Click("Burn"), "select the Burn operation")
	tester.Frame()
	require.NoError(t, tester.Click("Burn NFT…"))
	tester.Frame()
	require.True(t, tester.HasText("Burn Star"), "the NFT's name must be in the question")
	require.True(t, tester.HasText(shortID(id)), "the shortened id must identify it too")

	require.NoError(t, tester.Click("Cancel"))
	tester.Frame()
	require.False(t, app.nft.askBurn)
}

// TestDeleteDialogNamesTheSource pins the oracle confirmation: the actions
// share one id field, so the question must say which source the id names.
func TestDeleteDialogNamesTheSource(t *testing.T) {
	fake := &fakeServices{sources: []SourceRow{{ID: "src-1", Name: "btc-price"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	tester.Frame()

	app.oracle.fetchID = "src-1"
	tester.Frame()
	require.NoError(t, tester.Click("Delete"))
	tester.Frame()
	require.True(t, tester.HasText("Delete btc-price (src-1)?"),
		"the question must name the source, not just “this source”")

	require.NoError(t, tester.Click("Cancel"))
	tester.Frame()
	require.False(t, app.oracle.askDelete)
}

// TestCopyConfirmsWithAToast pins where a copy is acknowledged. The status
// line sits above the cards and is often scrolled away when the Copy button is
// clicked, so the confirmation is a toast at the pointer.
func TestCopyConfirmsWithAToast(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	app.voting.voterName = "Alice"
	app.registerVoter()
	tester.Frame()

	require.NoError(t, tester.Click("Copy Private key"))
	tester.Frame()
	require.Equal(t, "sk-Alice", tester.Clipboard())
	require.True(t, tester.HasText("Private key is on the clipboard"), "a toast must confirm the copy")
}
