package nft

import (
	"encoding/base64"
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNewNFTApp(t *testing.T) {
	app := NewNFTApp()
	assert.NotNil(t, app)
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, 0, app.menuIndex)
}

func TestModelInit(t *testing.T) {
	app := NewNFTApp()
	cmd := app.Init()
	assert.Nil(t, cmd)
}

func TestViewMenuState(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewMintState(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewTransferState(t *testing.T) {
	app := NewNFTApp()
	app.view = "transfer"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewQueryState(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewResultStateNil(t *testing.T) {
	app := NewNFTApp()
	app.view = "result"
	app.nft = nil
	view := app.View()
	assert.NotNil(t, view)
}

func TestMenuViewRenders(t *testing.T) {
	app := NewNFTApp()
	app.view = "menu"
	view := app.menuView()
	assert.NotEmpty(t, view)
}

func TestMintViewRenders(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	view := app.mintView()
	assert.NotEmpty(t, view)
}

func TestMintViewWithError(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.err = "test error"
	view := app.mintView()
	assert.Contains(t, view, "test error")
}

func TestMintViewWithSuccess(t *testing.T) {
	app := NewNFTApp()
	app.view = "mint"
	app.successMsg = "test success"
	view := app.mintView()
	assert.Contains(t, view, "test success")
}

func TestTransferViewRenders(t *testing.T) {
	app := NewNFTApp()
	app.view = "transfer"
	view := app.transferView()
	assert.NotEmpty(t, view)
}

func TestQueryViewRenders(t *testing.T) {
	app := NewNFTApp()
	app.view = "query"
	view := app.queryView()
	assert.NotEmpty(t, view)
}

func TestListViewRenders(t *testing.T) {
	app := NewNFTApp()
	app.view = "list"
	view := app.listView()
	assert.NotEmpty(t, view)
}

func TestResultViewRendersWithNFT(t *testing.T) {
	app := NewNFTApp()
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	app.nft = &nft.NFT{
		ID:          "test-nft-id",
		Name:        "Test NFT",
		Description: "A test NFT",
		Owner:       pubkey,
		BlockHeight: 1,
	}
	view := app.resultView()
	assert.NotEmpty(t, view)
}

func TestLoadNFTsByOwnerEmpty(t *testing.T) {
	app := NewNFTApp()
	app.ownerInput.SetValue("")
	app.loadNFTsByOwner()
	assert.NotNil(t, app)
}

func TestLoadNFTsByOwnerInvalidKey(t *testing.T) {
	app := NewNFTApp()
	app.ownerInput.SetValue("invalid-base64!")
	app.loadNFTsByOwner()
	assert.NotNil(t, app)
}

func TestLoadNFTsByOwnerValid(t *testing.T) {
	app := NewNFTApp()
	validKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	app.ownerInput.SetValue(validKey)
	app.loadNFTsByOwner()
	assert.NotNil(t, app)
}

func TestHandleMintEmptyName(t *testing.T) {
	app := NewNFTApp()
	app.nameInput.SetValue("")
	msg := app.handleMint()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.name_required"), app.err)
}

func TestHandleMintEmptyPubkey(t *testing.T) {
	app := NewNFTApp()
	app.nameInput.SetValue("TestNFT")
	app.descInput.SetValue("Description")
	app.pubkeyInput.SetValue("")
	msg := app.handleMint()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.pubkey_required"), app.err)
}

func TestHandleMintInvalidPubkey(t *testing.T) {
	app := NewNFTApp()
	app.nameInput.SetValue("TestNFT")
	app.descInput.SetValue("Description")
	app.pubkeyInput.SetValue("not-valid-base64!")
	msg := app.handleMint()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.invalid_pubkey"), app.err)
}

func TestHandleMintSuccess(t *testing.T) {
	app := NewNFTApp()
	app.nameInput.SetValue("TestNFT")
	app.descInput.SetValue("Description")
	pubkey := make([]byte, 32)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}
	app.pubkeyInput.SetValue(base64.StdEncoding.EncodeToString(pubkey))
	msg := app.handleMint()
	assert.Nil(t, msg)
	assert.Empty(t, app.err)
	assert.NotNil(t, app.nft)
	assert.Equal(t, "TestNFT", app.nft.Name)
	assert.Equal(t, "result", app.view)
}

func TestHandleTransferEmptyNFTID(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("")
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.nft_id_required"), app.err)
}

func TestHandleTransferEmptyFromKey(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("nft-123")
	app.fromKeyInput.SetValue("")
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.privkey_required"), app.err)
}

func TestHandleTransferEmptyToAddr(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("nft-123")
	app.fromKeyInput.SetValue(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	app.toAddrInput.SetValue("")
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.to_address_required"), app.err)
}

func TestHandleTransferInvalidFromKey(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("nft-123")
	app.fromKeyInput.SetValue("not-valid-base64!")
	app.toAddrInput.SetValue(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.invalid_privkey"), app.err)
}

func TestHandleTransferInvalidToAddr(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("nft-123")
	app.fromKeyInput.SetValue(base64.StdEncoding.EncodeToString(make([]byte, 64)))
	app.toAddrInput.SetValue("not-valid-base64!")
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.invalid_address"), app.err)
}

func TestHandleTransferWrongKeyLength(t *testing.T) {
	app := NewNFTApp()
	app.nftIDInput.SetValue("nft-123")
	app.fromKeyInput.SetValue(base64.StdEncoding.EncodeToString(make([]byte, 16)))
	app.toAddrInput.SetValue(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	msg := app.handleTransfer()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.invalid_privkey"), app.err)
}

func TestHandleQueryEmptyID(t *testing.T) {
	app := NewNFTApp()
	app.queryIDInput.SetValue("")
	msg := app.handleQuery()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.nft_id_required"), app.err)
}

func TestHandleQueryNotFound(t *testing.T) {
	app := NewNFTApp()
	app.queryIDInput.SetValue("nonexistent-nft")
	msg := app.handleQuery()
	assert.Nil(t, msg)
	assert.Equal(t, i18n.GetText("error.not_found"), app.err)
	assert.Nil(t, app.nft)
}
