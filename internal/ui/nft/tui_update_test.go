package nft

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s})
}

// newTestKeypair returns a real Ed25519 keypair so mint+transfer flows sign
// verifiable messages (the TUI never fabricates keys).
func newTestKeypair() (pub, priv []byte) {
	priv = ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	pub = priv[ed25519.SeedSize:]
	return pub, priv
}

func TestUpdate_QuitFromMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("q"))
	assert.NotNil(t, cmd, "q on the menu quits the TUI")
}

func TestUpdate_CtrlCFromMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("ctrl+c"))
	assert.NotNil(t, cmd, "ctrl+c on the menu quits the TUI")
}

func TestUpdate_QReturnsToMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.err = "x"
	app.successMsg = "y"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
	assert.Empty(t, app.err)
	assert.Empty(t, app.successMsg)
}

func TestUpdate_CtrlCReturnsToMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	app.Update(keyPress("ctrl+c"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_UpNavigation(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("up"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_UpNavigationLowerBound(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_DownNavigation(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("j"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_DownNavigationUpperBound(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.menuIndex = 3
	app.Update(keyPress("down"))
	assert.Equal(t, 3, app.menuIndex)
}

func TestUpdate_EnterInMenu_Navigates(t *testing.T) {
	app := NewNFTApp()
	for idx, want := range []string{"mint", "transfer", "query"} {
		app.menuIndex = idx
		app.Update(keyPress("enter"))
		assert.Equal(t, want, app.view)
		app.view = "menu"
	}
}

func TestUpdate_EnterInMenuAtExit_Quits(t *testing.T) {
	app := NewNFTApp()
	app.menuIndex = 3
	_, cmd := app.Update(keyPress("enter"))
	assert.NotNil(t, cmd, "the exit menu item quits the TUI")
}

func TestUpdate_EnterInMintTransferQuery_ReturnsCmd(t *testing.T) {
	for _, view := range []string{"mint", "transfer", "query"} {
		app := NewNFTApp()
		app.view = view
		_, cmd := app.Update(keyPress("enter"))
		assert.NotNil(t, cmd, "enter in %q dispatches the handler", view)
	}
}

func TestUpdate_EnterInList_LoadsAndStays(t *testing.T) {
	app := NewNFTApp()
	app.view = "list"
	app.Update(keyPress("enter"))
	assert.Equal(t, "list", app.view, "enter in list stays in list (loadNFTsByOwner does not navigate away)")
}

func TestUpdate_EnterInResult_ReturnsToMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "result"
	app.successMsg = "done"
	app.nft = &nft.NFT{ID: "x"}
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
	assert.Empty(t, app.successMsg)
	assert.Nil(t, app.nft)
}

func TestUpdate_NumericShortcuts(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.Update(keyPress("3"))
	assert.Equal(t, 2, app.menuIndex, "3 selects the third menu item (0-based 2)")
}

func TestUpdate_EscFromNonMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.err = "x"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
	assert.Empty(t, app.err)
}

func TestUpdate_EscFromMenuDoesNothing(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.menuIndex = 1
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, 1, app.menuIndex)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	app := NewNFTApp()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.NotNil(t, app.viewport)
}

func TestView_DefaultRendersMenu(t *testing.T) {
	app := NewNFTApp()
	app.view = "bogus-view"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestTransferView_WithError(t *testing.T) {
	app := NewNFTApp()
	app.view = "transfer"
	app.err = "boom"
	assert.Contains(t, app.transferView(), "boom")
}

func TestTransferView_WithSuccess(t *testing.T) {
	app := NewNFTApp()
	app.view = "transfer"
	app.successMsg = "ok"
	assert.Contains(t, app.transferView(), "ok")
}

func TestQueryView_WithError(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	app.err = "nope"
	assert.Contains(t, app.queryView(), "nope")
}

// mintNFT drives the TUI mint handler for a real keypair and returns the app
// plus the keypair, ready for a transfer/query/list follow-up.
func mintNFT(t *testing.T) (*model, []byte, []byte) {
	t.Helper()
	app := NewNFTApp()
	pub, priv := newTestKeypair()
	app.nameInput.SetValue("TestNFT")
	app.descInput.SetValue("A test NFT")
	app.pubkeyInput.SetValue(base64.StdEncoding.EncodeToString(pub))
	msg := app.handleMint()
	require.Nil(t, msg)
	require.NotNil(t, app.nft, "mint should succeed with a valid keypair")
	return app, pub, priv
}

func TestHandleTransfer_Success(t *testing.T) {
	app, _, priv := mintNFT(t)
	toPub := make([]byte, ed25519.PublicKeySize)

	app.nftIDInput.SetValue(app.nft.ID)
	app.fromKeyInput.SetValue(base64.StdEncoding.EncodeToString(priv))
	app.toAddrInput.SetValue(base64.StdEncoding.EncodeToString(toPub))

	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Empty(t, app.err)
	assert.Equal(t, "result", app.view)
	assert.Equal(t, i18n.GetText("nft.tui.transfer_success"), app.successMsg)
}

func TestHandleQuery_Success(t *testing.T) {
	app, _, _ := mintNFT(t)

	app.queryIDInput.SetValue(app.nft.ID)
	msg := app.handleQuery()
	assert.Nil(t, msg)
	assert.Empty(t, app.err)
	assert.Equal(t, "result", app.view)
	assert.NotNil(t, app.nft)
}

func TestLoadNFTsByOwner_Populated(t *testing.T) {
	app, pub, _ := mintNFT(t)

	app.ownerInput.SetValue(base64.StdEncoding.EncodeToString(pub))
	app.loadNFTsByOwner()
	assert.Contains(t, app.viewport.View(), "TestNFT")
}
