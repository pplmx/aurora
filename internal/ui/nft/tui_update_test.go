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

// q must be typable inside the mint form (the letter in name/description),
// not a back-to-menu key there (TASK-161, ISS-154).
func TestUpdate_QIsTypableInForm(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.err = "x"
	app.successMsg = "y"
	app.Update(keyPress("q"))
	assert.Equal(t, "mint", app.view)
	assert.Equal(t, "x", app.err)
	assert.Equal(t, "y", app.successMsg)
}

// ctrl+c is the hard quit in every view (not a back-to-menu key).
func TestUpdate_CtrlCHardQuits(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	_, cmd := app.Update(keyPress("ctrl+c"))
	assert.NotNil(t, cmd, "ctrl+c must quit, not return to the menu")
}

// TestUpdate_JKAndQuestionMarkTypableInForm pins the ISS-164 sweep: j/k and
// "?" are ordinary characters in the mint form and must be typed into the
// focused input (a description like "Jill's ? piece"), not consumed as
// navigation/help. Arrow keys and Tab remain the form-navigation keys.
func TestUpdate_JKAndQuestionMarkTypableInForm(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.inputFocus = 0
	app.updateInputFocus()
	app.nameInput.SetValue("")

	for _, ch := range []string{"j", "k", "?"} {
		app.Update(keyPress(ch))
	}
	assert.Equal(t, "jk?", app.nameInput.Value())
	assert.Equal(t, 0, app.inputFocus, "letter keys must not move form focus")
	assert.False(t, app.showHelp, "? in a form must not open help (ISS-164)")
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
	app.menuIndex = 4 // last item (Exit) since List-by-Owner was added (ISS-169)
	app.Update(keyPress("down"))
	assert.Equal(t, 4, app.menuIndex)
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
	app.menuIndex = 4 // Exit moved after List-by-Owner (ISS-169)
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

// A valid-base64 but wrong-length owner key must be rejected at mint, not
// stored as a permanently-untransferable NFT (TASK-163, ISS-156).
func TestHandleMint_RejectsWrongLengthOwnerKey(t *testing.T) {
	app := NewNFTApp()
	app.nameInput.SetValue("LockedNFT")
	app.descInput.SetValue("x")
	// "AAAA" decodes to 3 bytes, not a 32-byte ed25519 public key.
	app.pubkeyInput.SetValue("AAAA")
	msg := app.handleMint()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.invalid_pubkey"), app.err)
	assert.Nil(t, app.nft, "mint must not produce an NFT from a wrong-length owner key")
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
	assert.NotNil(t, app.nft, "result view must render the transferred NFT (not Not found)")
	assert.Equal(t, app.nft.Owner, toPub, "result view must show the post-transfer owner")
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

// Round-97 (TASK-123): mint/transfer/query forms never received keystrokes.
// These tests pin the fix: keypresses reach the focused input and Tab/up/down
// cycle focus between the fields.

func TestUpdate_MintFormReceivesKeystrokes(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("M"))
	app.Update(keyPress("y"))
	assert.Equal(t, "My", app.nameInput.Value())
}

func TestUpdate_MintFormTabCyclesFocus(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("tab"))
	assert.Equal(t, 1, app.inputFocus)
	assert.True(t, app.descInput.Focused())
	app.Update(keyPress("tab"))
	assert.Equal(t, 2, app.inputFocus)
	assert.True(t, app.pubkeyInput.Focused())
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_TransferFormReceivesKeystrokes(t *testing.T) {
	app := NewNFTApp()
	app.view = "transfer"
	app.inputFocus = 2
	app.updateInputFocus()
	app.Update(keyPress("t"))
	assert.Equal(t, "t", app.toAddrInput.Value())
}

func TestUpdate_QueryFormReceivesKeystrokes(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("n"))
	app.Update(keyPress("f"))
	assert.Equal(t, "nf", app.queryIDInput.Value())
}

func TestUpdate_UpDownCyclesFocusInMint(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.inputFocus = 1
	app.updateInputFocus()
	app.Update(keyPress("down"))
	assert.Equal(t, 2, app.inputFocus)
	app.Update(keyPress("down"))
	assert.Equal(t, 2, app.inputFocus) // bounded
	app.Update(keyPress("up"))
	assert.Equal(t, 1, app.inputFocus)
}

// Round-98 (TASK-127): list view is a viewport without scroll bindings.
func TestUpdate_ListScrolls(t *testing.T) {
	app := NewNFTApp()
	app.view = "list"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("nft0\nnft1\nnft2\nnft3\nnft4")

	y0 := app.viewport.YOffset()
	app.Update(keyPress("down"))
	assert.Greater(t, app.viewport.YOffset(), y0)
	app.Update(keyPress("pgdown"))
	assert.Greater(t, app.viewport.YOffset(), y0)
	app.Update(keyPress("j"))
	assert.Greater(t, app.viewport.YOffset(), y0)
}

// TestUpdate_ListOwnerMenuNavigates pins the ISS-169 wiring: the menu exposes
// "List by Owner" (index 3) and Enter enters the listOwner prompt, where the
// owner public key is typable (part of the same key-typing convention pinned
// by the j/k/? tests).
func TestUpdate_ListOwnerMenuNavigates(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	app.menuIndex = 3
	app.Update(keyPress("enter"))
	require.Equal(t, "listOwner", app.view, "menu item 3 must open the list-by-owner prompt")

	app.Update(keyPress("a"))
	app.Update(keyPress("b"))
	assert.Equal(t, "ab", app.ownerInput.Value(), "owner key must be typable in the prompt")
}

// TestUpdate_ListOwnerEnterLoadsList: Enter on the listOwner prompt runs the
// (previously dead) loadNFTsByOwner+list transition — with an empty owner it
// must land on the list view showing the required-pubkey error, proving the
// Enter->load->list path is reachable from the menu.
func TestUpdate_ListOwnerEnterLoadsList(t *testing.T) {
	app := NewNFTApp()
	app.view = "listOwner"
	app.ownerInput.SetValue("")
	app.Update(keyPress("enter"))
	assert.Equal(t, "list", app.view, "Enter in the prompt must load the owner list")
	require.Contains(t, app.viewport.View(), i18n.GetText("error.pubkey_required"))
}
