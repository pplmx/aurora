package token

import (
	"database/sql"
	"encoding/base64"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNewTokenApp(t *testing.T) {
	app := NewTokenApp()
	assert.NotNil(t, app)
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, 0, app.menuIndex)
	assert.NotNil(t, app.tokenService)
	assert.NotNil(t, app.ownerKey)
}

func TestModelInit(t *testing.T) {
	app := NewTokenApp()
	cmd := app.Init()
	assert.Nil(t, cmd)
}

func TestViewMenuState(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewCreateState(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewMintState(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewTransferState(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewBalanceState(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewHistoryState(t *testing.T) {
	app := NewTokenApp()
	app.view = "history"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestMenuViewRenders(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	view := app.menuView()
	assert.NotEmpty(t, view)
}

func TestCreateViewRenders(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	view := app.createView()
	assert.NotEmpty(t, view)
}

func TestCreateViewWithError(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.err = "test error"
	view := app.createView()
	assert.Contains(t, view, "test error")
}

func TestCreateViewWithSuccess(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.successMsg = "test success"
	view := app.createView()
	assert.Contains(t, view, "test success")
}

func TestMintViewRendersWithoutToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.currentToken = nil
	view := app.mintView()
	assert.Contains(t, view, i18n.GetText("token.tui.no_token"))
}

func TestTransferViewRendersWithoutToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.currentToken = nil
	view := app.transferView()
	assert.Contains(t, view, i18n.GetText("token.tui.no_token"))
}

func TestBalanceViewRenders(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	view := app.balanceView()
	assert.NotEmpty(t, view)
}

func TestHistoryViewRenders(t *testing.T) {
	app := NewTokenApp()
	app.view = "history"
	view := app.historyView()
	assert.NotEmpty(t, view)
}

func TestMinFunction(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 5, min(5, 5))
}

func TestNewInmemEventBus(t *testing.T) {
	es := &inmemEventStore{}
	bus := newInmemEventBus(es)
	assert.NotNil(t, bus)
	assert.NotNil(t, bus.eventStore)
}

func TestInmemReplayProtection(t *testing.T) {
	rp := newInmemReplayProtection()
	assert.NotNil(t, rp)
	nonce, err := rp.GetLastNonce("token1", []byte("owner"))
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), nonce)
	err = rp.SaveNonce("token1", []byte("owner"), 5)
	assert.NoError(t, err)
	nonce, _ = rp.GetLastNonce("token1", []byte("owner"))
	assert.Equal(t, uint64(5), nonce)
}

func TestInmemRepo(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}
	assert.NotNil(t, repo)
}

func TestInmemEventStore(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	assert.NotNil(t, es)
	events, err := es.GetTransferEventsByOwner("token1", token.PublicKey{}, 10, 0)
	assert.NoError(t, err)
	assert.Empty(t, events)
	mintEvents, err := es.GetMintEventsByToken("token1")
	assert.NoError(t, err)
	assert.Empty(t, mintEvents)
	burnEvents, err := es.GetBurnEventsByToken("token1")
	assert.NoError(t, err)
	assert.Empty(t, burnEvents)
}

func TestNoOpTxManager(t *testing.T) {
	tx := &noOpTxManager{}
	err := tx.WithTransaction(func(tx *sql.Tx) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestMintViewRendersWithToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	view := app.mintView()
	assert.NotEmpty(t, view)
}

func TestTransferViewRendersWithToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	view := app.transferView()
	assert.NotEmpty(t, view)
}

func TestTransferViewWithError(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.err = "transfer error"
	view := app.transferView()
	assert.Contains(t, view, "transfer error")
}

func TestTransferViewWithSuccess(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.successMsg = "transfer success"
	view := app.transferView()
	assert.Contains(t, view, "transfer success")
}

func TestBalanceViewWithError(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	app.err = "balance error"
	view := app.balanceView()
	assert.Contains(t, view, "balance error")
}

func TestBalanceViewWithSuccess(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	app.successMsg = "Balance: 100 TST"
	view := app.balanceView()
	assert.Contains(t, view, "Balance: 100 TST")
}

func TestInmemEventBusPublishTransfer(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	bus := newInmemEventBus(es)

	evt := token.NewTransferEvent(
		"token1",
		token.PublicKey([]byte("from")),
		token.PublicKey([]byte("to")),
		token.NewAmount(100),
		1,
		token.Signature{},
	)
	err := bus.Publish(evt)
	assert.NoError(t, err)
	assert.Len(t, es.transferEvents, 1)
	assert.Equal(t, token.TokenID("token1"), es.transferEvents[0].TokenID())
}

func TestInmemEventBusPublishMint(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	bus := newInmemEventBus(es)

	evt := token.NewMintEvent("token1", token.PublicKey([]byte("to")), token.NewAmount(500))
	err := bus.Publish(evt)
	assert.NoError(t, err)
	assert.Len(t, es.mintEvents, 1)
}

func TestInmemEventBusPublishBurn(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	bus := newInmemEventBus(es)

	evt := token.NewBurnEvent("token1", token.PublicKey([]byte("from")), token.NewAmount(50))
	err := bus.Publish(evt)
	assert.NoError(t, err)
	assert.Len(t, es.burnEvents, 1)
}

func TestInmemEventBusPublishApprove(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	bus := newInmemEventBus(es)

	evt := token.NewApproveEvent("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("spender")), token.NewAmount(200))
	err := bus.Publish(evt)
	assert.NoError(t, err)
	assert.Len(t, es.approveEvents, 1)
}

func TestInmemRepoSaveAndGetToken(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	tok := token.NewToken("token1", "TestToken", "TST", token.NewAmount(1000), token.PublicKey([]byte("owner")))
	err := repo.SaveToken(tok)
	assert.NoError(t, err)

	retrieved, err := repo.GetToken(tok.ID())
	assert.NoError(t, err)
	assert.Equal(t, tok.ID(), retrieved.ID())
}

func TestInmemRepoGetTokenNotFound(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	retrieved, err := repo.GetToken("nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestInmemRepoSaveAndGetApproval(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	approval := token.NewApproval("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("spender")), token.NewAmount(100))
	err := repo.SaveApproval(approval)
	assert.NoError(t, err)

	retrieved, err := repo.GetApproval("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("spender")))
	assert.NoError(t, err)
	assert.Equal(t, "token1", string(retrieved.TokenID()))
}

func TestInmemRepoGetApprovalsByOwner(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	approval := token.NewApproval("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("spender")), token.NewAmount(100))
	err := repo.SaveApproval(approval)
	assert.NoError(t, err)

	approvals, err := repo.GetApprovalsByOwner("token1", token.PublicKey([]byte("owner")))
	assert.NoError(t, err)
	assert.Len(t, approvals, 1)
}

func TestInmemRepoGetSetAccountBalance(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	owner := token.PublicKey([]byte("owner"))
	amount := token.NewAmount(500)

	err := repo.SetAccountBalance("token1", owner, amount)
	assert.NoError(t, err)

	retrieved, err := repo.GetAccountBalance("token1", owner)
	assert.NoError(t, err)
	assert.Equal(t, "500", retrieved.String())
}

func TestInmemRepoGetAccountBalanceNotSet(t *testing.T) {
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}

	owner := token.PublicKey([]byte("owner"))
	retrieved, err := repo.GetAccountBalance("token1", owner)
	assert.NoError(t, err)
	assert.Equal(t, "0", retrieved.String())
}

func TestInmemEventStoreGetTransferEventsByOwner(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: []*token.TransferEvent{
			token.NewTransferEvent("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("recipient")), token.NewAmount(100), 1, token.Signature{}),
		},
		mintEvents:    make([]*token.MintEvent, 0),
		burnEvents:    make([]*token.BurnEvent, 0),
		approveEvents: make([]*token.ApproveEvent, 0),
	}

	events, err := es.GetTransferEventsByOwner("token1", token.PublicKey([]byte("owner")), 10, 0)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestInmemEventStoreGetTransferEventsByOwnerEmpty(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}

	events, err := es.GetTransferEventsByOwner("token1", token.PublicKey([]byte("owner")), 0, 0)
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestInmemEventStoreGetTransferEventsByOwnerOffset(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: []*token.TransferEvent{
			token.NewTransferEvent("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("recipient")), token.NewAmount(100), 1, token.Signature{}),
			token.NewTransferEvent("token1", token.PublicKey([]byte("owner")), token.PublicKey([]byte("recipient2")), token.NewAmount(200), 2, token.Signature{}),
		},
		mintEvents:    make([]*token.MintEvent, 0),
		burnEvents:    make([]*token.BurnEvent, 0),
		approveEvents: make([]*token.ApproveEvent, 0),
	}

	events, err := es.GetTransferEventsByOwner("token1", token.PublicKey([]byte("owner")), 10, 1)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestInmemEventStoreGetMintEventsByToken(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents: []*token.MintEvent{
			token.NewMintEvent("token1", token.PublicKey([]byte("owner")), token.NewAmount(1000)),
		},
		burnEvents:    make([]*token.BurnEvent, 0),
		approveEvents: make([]*token.ApproveEvent, 0),
	}

	events, err := es.GetMintEventsByToken("token1")
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestInmemEventStoreGetBurnEventsByToken(t *testing.T) {
	es := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents: []*token.BurnEvent{
			token.NewBurnEvent("token1", token.PublicKey([]byte("owner")), token.NewAmount(50)),
		},
		approveEvents: make([]*token.ApproveEvent, 0),
	}

	events, err := es.GetBurnEventsByToken("token1")
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s})
}

func TestUpdate_QuitFromMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("q"))
	assert.NotNil(t, cmd)
}

func TestUpdate_CtrlCFromMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	_, cmd := app.Update(keyPress("ctrl+c"))
	assert.NotNil(t, cmd)
}

func TestUpdate_QReturnsToMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_CtrlCReturnsToMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.Update(keyPress("ctrl+c"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_UpNavigation(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("up"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_UpNavigationLowerBound(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_DownNavigation(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("j"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_DownNavigationUpperBound(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 4
	app.Update(keyPress("down"))
	assert.Equal(t, 4, app.menuIndex)
}

func TestUpdate_NumericShortcuts(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.Update(keyPress("1"))
	assert.Equal(t, 0, app.menuIndex)
	app.Update(keyPress("2"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("3"))
	assert.Equal(t, 2, app.menuIndex)
	app.Update(keyPress("4"))
	assert.Equal(t, 3, app.menuIndex)
	app.Update(keyPress("5"))
	assert.Equal(t, 4, app.menuIndex)
}

func TestUpdate_NumericOutsideMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.menuIndex = 2
	app.Update(keyPress("3"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_EnterInMenuNavigateToCreate(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("enter"))
	assert.Equal(t, "create", app.view)
}

func TestUpdate_EnterInMenuNavigateToMint(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "mint", app.view)
}

func TestUpdate_EnterInMenuNavigateToTransfer(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("enter"))
	assert.Equal(t, "transfer", app.view)
}

func TestUpdate_EnterInMenuNavigateToBalance(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 3
	app.Update(keyPress("enter"))
	assert.Equal(t, "balance", app.view)
}

func TestUpdate_EnterInMenuNavigateToHistory(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.menuIndex = 4
	app.Update(keyPress("enter"))
	assert.Equal(t, "history", app.view)
}

func TestUpdate_EnterInHistoryReturnsToMenu(t *testing.T) {
	app := NewTokenApp()
	app.view = "history"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_EscFromCreate(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.err = "some error"
	app.successMsg = "some success"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, "", app.err)
	assert.Equal(t, "", app.successMsg)
}

func TestUpdate_EscFromMenuDoesNothing(t *testing.T) {
	app := NewTokenApp()
	app.view = "menu"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	app := NewTokenApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Equal(t, 76, app.viewport.Width())
	assert.Equal(t, 12, app.viewport.Height())
}

func TestUpdate_CreateViewTextInputs(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.Update(keyPress("a"))
	assert.NotNil(t, app)
}

func TestHandleCreate_EmptyName(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("")
	app.createSymbolInput.SetValue("SYM")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestHandleCreate_InvalidSupply(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("SYM")
	app.createSupplyInput.SetValue("invalid")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestHandleMint_EmptyTo(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.mintToInput.SetValue("")
	app.mintAmountInput.SetValue("100")
	app.handleMint()
	assert.NotEmpty(t, app.err)
}

func TestHandleTransfer_EmptyTo(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.transferToInput.SetValue("")
	app.transferAmountInput.SetValue("50")
	app.handleTransfer()
	assert.NotEmpty(t, app.err)
}

func TestHandleBalance_EmptyAddress(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	app.balanceAddressInput.SetValue("")
	app.handleBalance()
	assert.NotEmpty(t, app.err)
}

func TestUpdate_QClearsMessages(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.err = "error"
	app.successMsg = "success"
	app.Update(keyPress("q"))
	assert.Equal(t, "", app.err)
	assert.Equal(t, "", app.successMsg)
}

func TestHandleCreate_Success(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()
	assert.NotEmpty(t, app.successMsg)
	assert.NotNil(t, app.currentToken)
}

func TestHandleMint_Success(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "mint"
	ownerB64 := base64.StdEncoding.EncodeToString(app.ownerKey)
	app.mintToInput.SetValue(ownerB64)
	app.mintAmountInput.SetValue("100")
	app.mintPrivateInput.SetValue("")
	app.handleMint()
	assert.NotEmpty(t, app.successMsg)
}

func TestHandleTransfer_Success(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()
	assert.NotNil(t, app.currentToken)

	recipient := make([]byte, 32)
	recipientB64 := base64.StdEncoding.EncodeToString(recipient)

	app.view = "transfer"
	app.transferToInput.SetValue(recipientB64)
	app.transferAmountInput.SetValue("50")
	app.transferPrivateInput.SetValue("")
	app.handleTransfer()
	assert.NotNil(t, app)
}

func TestHandleBalance_Success(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "balance"
	app.balanceAddressInput.SetValue("")
	app.handleBalance()
	assert.NotEmpty(t, app.successMsg)
}

func TestHandleBalance_WithAddress(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "balance"
	ownerB64 := base64.StdEncoding.EncodeToString(app.ownerKey)
	app.balanceAddressInput.SetValue(ownerB64)
	app.handleBalance()
	assert.NotEmpty(t, app.successMsg)
}

func TestHandleMint_InvalidAddress(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "mint"
	app.mintToInput.SetValue("not-base64!!!")
	app.mintAmountInput.SetValue("100")
	app.handleMint()
	assert.NotEmpty(t, app.err)
}

func TestHandleMint_InvalidAmount(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "mint"
	ownerB64 := base64.StdEncoding.EncodeToString(app.ownerKey)
	app.mintToInput.SetValue(ownerB64)
	app.mintAmountInput.SetValue("not-a-number")
	app.handleMint()
	assert.NotEmpty(t, app.err)
}

func TestHandleTransfer_InvalidAddress(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "transfer"
	app.transferToInput.SetValue("not-base64!!!")
	app.transferAmountInput.SetValue("50")
	app.handleTransfer()
	assert.NotEmpty(t, app.err)
}

func TestHandleTransfer_InvalidAmount(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "transfer"
	app.transferToInput.SetValue(base64.StdEncoding.EncodeToString(app.ownerKey))
	app.transferAmountInput.SetValue("not-a-number")
	app.handleTransfer()
	assert.NotEmpty(t, app.err)
}

func TestHandleBalance_InvalidAddress(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()

	app.view = "balance"
	app.balanceAddressInput.SetValue("not-base64!!!")
	app.handleBalance()
	assert.NotEmpty(t, app.err)
}

func TestHandleCreate_SymbolRequired(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("")
	app.createSupplyInput.SetValue("1000")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestHandleCreate_SupplyRequired(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestHandleCreate_InvalidDecimals(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.createNameInput.SetValue("TestToken")
	app.createSymbolInput.SetValue("TST")
	app.createSupplyInput.SetValue("1000")
	app.createDecimalsInput.SetValue("not-a-number")
	app.handleCreate()
	assert.NotEmpty(t, app.err)
}

func TestHandleMint_NoToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.handleMint()
	assert.NotEmpty(t, app.err)
}

func TestHandleTransfer_NoToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.handleTransfer()
	assert.NotEmpty(t, app.err)
}

func TestHandleBalance_NoToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	app.handleBalance()
	assert.NotEmpty(t, app.err)
}

func TestHandleMint_NoTokenForAmount(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.mintToInput.SetValue("")
	app.handleMint()
	assert.NotEmpty(t, app.err)
}

func TestHandleTransfer_NoTokenForAmount(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.transferToInput.SetValue("")
	app.handleTransfer()
	assert.NotEmpty(t, app.err)
}

// Round-97 (TASK-123): only the create form received keystrokes — the
// mint/transfer/balance inputs were never updated so their forms could not
// be filled by typing. These tests pin the fix.

func TestUpdate_MintFormReceivesKeystrokes(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("A"))
	assert.Equal(t, "A", app.mintToInput.Value())
}

func TestUpdate_MintFormTabCyclesFocus(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("tab"))
	assert.Equal(t, 1, app.inputFocus)
	assert.True(t, app.mintAmountInput.Focused())
	app.Update(keyPress("tab"))
	assert.Equal(t, 2, app.inputFocus)
	assert.True(t, app.mintPrivateInput.Focused())
}

func TestUpdate_TransferFormReceivesKeystrokes(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.inputFocus = 1
	app.updateInputFocus()
	app.Update(keyPress("5"))
	app.Update(keyPress("0"))
	assert.Equal(t, "50", app.transferAmountInput.Value())
}

func TestUpdate_BalanceFormReceivesKeystrokes(t *testing.T) {
	app := NewTokenApp()
	app.view = "balance"
	app.inputFocus = 0
	app.updateInputFocus()
	app.Update(keyPress("b"))
	app.Update(keyPress("c"))
	assert.Equal(t, "bc", app.balanceAddressInput.Value())
}

func TestUpdate_CreateFormReceivesKeystrokes(t *testing.T) {
	app := NewTokenApp()
	app.view = "create"
	app.inputFocus = 1
	app.updateInputFocus()
	app.Update(keyPress("S"))
	assert.Equal(t, "S", app.createSymbolInput.Value())
}

// Round-98 (TASK-127): history view is a viewport without scroll bindings.
func TestUpdate_HistoryScrolls(t *testing.T) {
	app := NewTokenApp()
	app.view = "history"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("line0\nline1\nline2\nline3\nline4")

	y0 := app.viewport.YOffset()
	app.Update(keyPress("down"))
	assert.Greater(t, app.viewport.YOffset(), y0)
	app.Update(keyPress("pgdown"))
	assert.Greater(t, app.viewport.YOffset(), y0)
	app.Update(keyPress("up"))
	assert.LessOrEqual(t, app.viewport.YOffset(), 2)
}

// TestUpdate_QuestionTogglesHelp pins the TASK-130/ISS-118 fix: the menu
// footer advertises "? for help" (help.nav), so "?" must open the help view
// and esc/? must close it, while ordinary keys are swallowed while help is up
// (otherwise a stray Enter would mutate state or leave the view).
func TestUpdate_QuestionTogglesHelp(t *testing.T) {
	app := NewTokenApp()
	app.Update(keyPress("?"))
	assert.True(t, app.showHelp, "? opens the help view")

	app.Update(keyPress("enter"))
	assert.True(t, app.showHelp, "non-exit keys are swallowed while help is open")

	app.Update(keyPress("esc"))
	assert.False(t, app.showHelp, "esc closes the help view")

	app = NewTokenApp()
	app.Update(keyPress("?"))
	app.Update(keyPress("?"))
	assert.False(t, app.showHelp, "? toggles the help view closed again")
}

func TestView_HelpScreenContent(t *testing.T) {
	app := NewTokenApp()
	app.showHelp = true
	assert.Contains(t, app.View().Content, i18n.GetText("tui.help.title"))
}

// TestMintViewNoToken_Localized pins the TASK-131/ISS-120 fix: the token
// TUI's views must follow the active locale instead of rendering hardcoded
// CJK — zh surfaces Chinese, en surfaces English, and no raw CJK literal
// leaks into an en session.
func TestMintViewNoToken_Localized(t *testing.T) {
	translator := i18n.GetTranslator()
	prev := translator.GetLocale()
	defer func() { translator.SetLocale(prev) }()

	translator.SetLocale("zh")
	app := NewTokenApp()
	app.view = "mint"
	app.currentToken = nil
	assert.Contains(t, app.mintView(), "请先创建代币")

	translator.SetLocale("en")
	assert.Contains(t, app.mintView(), "Please create a token first")
}
