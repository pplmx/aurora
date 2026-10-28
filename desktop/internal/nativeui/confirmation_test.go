package nativeui

import (
	"strconv"
	"strings"
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

// TestDeleteDialogResolvesASourceOutsideTheCurrentPage covers the id a CLI or
// an earlier session leaves in the action field. The source page is bounded,
// so a page-only label lookup showed an opaque id for older storage.
func TestDeleteDialogResolvesASourceOutsideTheCurrentPage(t *testing.T) {
	sources := make([]SourceRow, oraclePageSize+1)
	for i := range sources {
		sources[i] = SourceRow{ID: "src-" + strconv.Itoa(i), Name: "Source " + strconv.Itoa(i)}
	}
	app := New(&fakeServices{sources: sources})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	tester.Frame()

	target := sources[len(sources)-1]
	require.False(t, containsSourceID(app.oracle.sources, target.ID),
		"the target must start outside the visible source page")
	app.oracle.fetchID = target.ID
	tester.Frame()
	require.NoError(t, tester.Click("Delete"))
	tester.Frame()
	require.True(t, tester.HasText("Delete "+target.Name+" ("+shortID(target.ID)+")?"),
		"the confirmation must resolve the name beyond the visible page")

	require.NoError(t, tester.Click("Cancel"))
}

// TestConfirmationDialogHonoursTextScale keeps the irreversible-action gate
// readable when the desktop asks for larger text. The page scales its own
// labels, but MyGo's built-in alert follows the theme, so the dialog used to
// stay at the default size while the window behind it grew.
func TestConfirmationDialogHonoursTextScale(t *testing.T) {
	buttonHeight := func(t *testing.T, scale float32) float32 {
		t.Helper()
		app := New(&fakeServices{})
		app.token.askBurn = true
		app.token.burnAmount = "25"
		app.token.burnFrom = "owner-pk"
		tester := ui.NewTester(app.View, 900, 600)
		if scale != 1 {
			tester.SetPreferences(ui.Preferences{TextScale: scale})
		}
		app.ShowPage("Token")
		tester.Frame()

		cancel, ok := tester.Find(tr("Cancel"))
		require.True(t, ok)
		require.Greater(t, cancel.H, float32(0))
		require.NoError(t, tester.Click(tr("Cancel")))
		tester.Frame()
		require.False(t, app.token.askBurn)
		return cancel.H
	}

	normal := buttonHeight(t, 1)
	large := buttonHeight(t, 1.5)
	require.Greater(t, large, normal*1.25,
		"the confirmation controls must grow with the desktop text scale")
}

// TestLongNFTNameKeepsBurnActionsInView makes the destructive-action gate fit
// the shortest supported window even with the maximum NFT name and a large
// desktop text scale. The full stored name is not needed to identify the asset:
// the question keeps a readable prefix and the shortened unique id.
func TestLongNFTNameKeepsBurnActionsInView(t *testing.T) {
	const id = "5c43a418-ca80-46ba-af63-cc7893d1a5e3"
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	app.ShowPage("NFT")
	app.nft.burnID = id
	app.nft.nft = &NFTRow{ID: id, Name: strings.Repeat("Long NFT ", 25)}
	app.nft.askBurn = true
	tester.Frame()

	cancel, ok := tester.Find(tr("Cancel"))
	require.True(t, ok)
	require.Greater(t, cancel.W, float32(0))
	require.LessOrEqual(t, cancel.Y+cancel.H, float32(600),
		"the dialog's buttons must remain inside the minimum-height window")
	require.True(t, tester.Focused(tr("Burn")), "the default action must remain focusable")
	require.NoError(t, tester.Click(tr("Cancel")))
	tester.Frame()
	require.False(t, app.nft.askBurn)
}

// TestConciseTextKeepsRunesWhole protects the dialog's name bound from
// splitting a multibyte UTF-8 character when Chinese or another script is in
// the stored name.
func TestConciseTextKeepsRunesWhole(t *testing.T) {
	require.Equal(t, "你好…", conciseText(" 你好世界 ", 2))
	require.Equal(t, "owner-pk", conciseText(" owner-pk ", 32))
}

func containsSourceID(sources []SourceRow, id string) bool {
	for _, source := range sources {
		if source.ID == id {
			return true
		}
	}
	return false
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
