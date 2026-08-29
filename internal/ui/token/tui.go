package token

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/ui/components"
)

type inmemEventBus struct {
	eventStore *inmemEventStore
}

func newInmemEventBus(es *inmemEventStore) *inmemEventBus {
	return &inmemEventBus{eventStore: es}
}

func (b *inmemEventBus) Publish(e events.Event) error {
	switch evt := e.(type) {
	case *token.TransferEvent:
		b.eventStore.transferEvents = append(b.eventStore.transferEvents, evt)
	case *token.MintEvent:
		b.eventStore.mintEvents = append(b.eventStore.mintEvents, evt)
	case *token.BurnEvent:
		b.eventStore.burnEvents = append(b.eventStore.burnEvents, evt)
	case *token.ApproveEvent:
		b.eventStore.approveEvents = append(b.eventStore.approveEvents, evt)
	}
	return nil
}

func (b *inmemEventBus) Subscribe(eventType string, handler infraevents.Handler) func() {
	return func() {}
}

func (b *inmemEventBus) SubscribeAll(handler infraevents.Handler) func() {
	return func() {}
}

type inmemReplayProtection struct {
	nonces map[string]uint64
}

func newInmemReplayProtection() *inmemReplayProtection {
	return &inmemReplayProtection{
		nonces: make(map[string]uint64),
	}
}

func (r *inmemReplayProtection) GetLastNonce(tokenID string, owner []byte) (uint64, error) {
	key := tokenID + string(owner)
	return r.nonces[key], nil
}

func (r *inmemReplayProtection) SaveNonce(tokenID string, owner []byte, nonce uint64) error {
	key := tokenID + string(owner)
	r.nonces[key] = nonce
	return nil
}

// ClaimNextNonce for the in-memory TUI mock: the TUI is single-goroutine
// so a non-atomic increment is fine here. Real production replay
// protection lives in SQLite (see internal/infra/events/replay.go).
func (r *inmemReplayProtection) ClaimNextNonce(tokenID string, owner []byte) (uint64, error) {
	key := tokenID + string(owner)
	r.nonces[key]++
	return r.nonces[key], nil
}

type model struct {
	view         string
	menuIndex    int
	inputFocus   int
	showHelp     bool
	err          string
	successMsg   string
	chain        *blockchain.BlockChain
	tokenService token.Service
	currentToken *token.Token
	ownerKey     token.PublicKey
	ownerPriv    []byte

	createNameInput     textinput.Model
	createSymbolInput   textinput.Model
	createSupplyInput   textinput.Model
	createDecimalsInput textinput.Model

	mintToInput      textinput.Model
	mintAmountInput  textinput.Model
	mintPrivateInput textinput.Model

	transferToInput      textinput.Model
	transferAmountInput  textinput.Model
	transferPrivateInput textinput.Model

	balanceAddressInput textinput.Model

	viewport viewport.Model
}

func NewTokenApp() *model {
	chain := blockchain.InitBlockChain()

	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}
	eventStore := &inmemEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
	}
	eventBus := newInmemEventBus(eventStore)
	replay := newInmemReplayProtection()
	txManager := &noOpTxManager{}
	tokenService := token.NewService(repo, txManager, eventBus, eventStore, replay, chain)

	pub, priv, _ := ed25519.GenerateKey(nil)
	ownerKey := token.PublicKey(pub)

	createNameInput := textinput.New()
	createNameInput.Placeholder = i18n.GetText("token.name")
	createNameInput.Focus()
	createNameInput.Prompt = "  "

	createSymbolInput := textinput.New()
	createSymbolInput.Placeholder = "AUR"
	createSymbolInput.Prompt = "  "

	createSupplyInput := textinput.New()
	createSupplyInput.Placeholder = "1000000"
	createSupplyInput.Prompt = "  "

	createDecimalsInput := textinput.New()
	createDecimalsInput.Placeholder = "8"
	createDecimalsInput.SetValue("8")
	createDecimalsInput.Prompt = "  "

	mintToInput := textinput.New()
	mintToInput.Placeholder = i18n.GetText("token.to")
	mintToInput.Prompt = "  "

	mintAmountInput := textinput.New()
	mintAmountInput.Placeholder = i18n.GetText("token.amount")
	mintAmountInput.Prompt = "  "

	mintPrivateInput := textinput.New()
	mintPrivateInput.Placeholder = i18n.GetText("token.private_key")
	mintPrivateInput.EchoMode = textinput.EchoPassword
	mintPrivateInput.Prompt = "  "

	transferToInput := textinput.New()
	transferToInput.Placeholder = i18n.GetText("token.to")
	transferToInput.Prompt = "  "

	transferAmountInput := textinput.New()
	transferAmountInput.Placeholder = i18n.GetText("token.amount")
	transferAmountInput.Prompt = "  "

	transferPrivateInput := textinput.New()
	transferPrivateInput.Placeholder = i18n.GetText("token.private_key")
	transferPrivateInput.EchoMode = textinput.EchoPassword
	transferPrivateInput.Prompt = "  "

	balanceAddressInput := textinput.New()
	balanceAddressInput.Placeholder = i18n.GetText("token.owner")
	balanceAddressInput.Prompt = "  "

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))

	return &model{
		view:                 "menu",
		menuIndex:            0,
		chain:                chain,
		tokenService:         tokenService,
		ownerKey:             ownerKey,
		ownerPriv:            priv,
		createNameInput:      createNameInput,
		createSymbolInput:    createSymbolInput,
		createSupplyInput:    createSupplyInput,
		createDecimalsInput:  createDecimalsInput,
		mintToInput:          mintToInput,
		mintAmountInput:      mintAmountInput,
		mintPrivateInput:     mintPrivateInput,
		transferToInput:      transferToInput,
		transferAmountInput:  transferAmountInput,
		transferPrivateInput: transferPrivateInput,
		balanceAddressInput:  balanceAddressInput,
		viewport:             vp,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.showHelp {
			if s := msg.String(); s == "esc" || s == "?" {
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			// ctrl+c is the hard quit in every view.
			return m, tea.Quit
		case "q":
			if m.view == "menu" {
				return m, tea.Quit
			}
			// In a form view q must fall through so it is typable in the
			// symbol/amount/address inputs (the help screen scopes q to the
			// menu); read-only views (history, etc) keep back-to-menu.
			if !m.isFormView() {
				m.view = "menu"
				m.err = ""
				m.successMsg = ""
				return m, nil
			}

		case "?":
			m.showHelp = true
			return m, nil

		case "up", "k":
			if m.view == "menu" {
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			} else if m.isFormView() && m.inputFocus > 0 {
				m.inputFocus--
				m.updateInputFocus()
				return m, nil
			}

		case "down", "j":
			if m.view == "menu" {
				if m.menuIndex < 4 {
					m.menuIndex++
				}
			} else if m.isFormView() && m.inputFocus < m.formInputCount()-1 {
				m.inputFocus++
				m.updateInputFocus()
				return m, nil
			}

		case "tab":
			if m.isFormView() && m.formInputCount() > 1 {
				m.inputFocus = (m.inputFocus + 1) % m.formInputCount()
				m.updateInputFocus()
				return m, nil
			}

		case "enter":
			switch m.view {
			case "menu":
				m.handleSelect()
			case "create":
				m.handleCreate()
			case "mint":
				m.handleMint()
			case "transfer":
				m.handleTransfer()
			case "balance":
				m.handleBalance()
			case "history":
				m.view = "menu"
			}

		case "1", "2", "3", "4", "5":
			if m.view == "menu" {
				m.menuIndex = int(msg.String()[0] - '1')
			}

		case "esc":
			if m.view != "menu" {
				m.view = "menu"
				m.err = ""
				m.successMsg = ""
			}
		}

	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 12)
	}

	// Forward remaining keypresses into the focused input so the mint /
	// transfer / balance forms are typable, not just the create form
	// (round-97: keystrokes only reached the create inputs).
	cmd = tea.Batch(cmd, m.forwardToActiveInput(msg))

	// The history view is a viewport; let it handle scrolling keys
	// (up/down/j/k/pgup/pgdn/space/b/f/u/d) so long transfer histories are
	// reachable instead of clipped at the 15-row viewport (TASK-127, ISS-119).
	if m.view == "history" {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

// isFormView reports whether the current view edits token fields.
func (m *model) isFormView() bool {
	switch m.view {
	case "create", "mint", "transfer", "balance":
		return true
	}
	return false
}

// formInputCount returns how many text inputs the current view has.
func (m *model) formInputCount() int {
	switch m.view {
	case "create":
		return 4
	case "mint", "transfer":
		return 3
	case "balance":
		return 1
	}
	return 0
}

// updateInputFocus focuses the current view's input at m.inputFocus.
func (m *model) updateInputFocus() {
	switch m.view {
	case "create":
		m.createNameInput.Blur()
		m.createSymbolInput.Blur()
		m.createSupplyInput.Blur()
		m.createDecimalsInput.Blur()
		switch m.inputFocus {
		case 0:
			m.createNameInput.Focus()
		case 1:
			m.createSymbolInput.Focus()
		case 2:
			m.createSupplyInput.Focus()
		case 3:
			m.createDecimalsInput.Focus()
		}
	case "mint":
		m.mintToInput.Blur()
		m.mintAmountInput.Blur()
		m.mintPrivateInput.Blur()
		switch m.inputFocus {
		case 0:
			m.mintToInput.Focus()
		case 1:
			m.mintAmountInput.Focus()
		case 2:
			m.mintPrivateInput.Focus()
		}
	case "transfer":
		m.transferToInput.Blur()
		m.transferAmountInput.Blur()
		m.transferPrivateInput.Blur()
		switch m.inputFocus {
		case 0:
			m.transferToInput.Focus()
		case 1:
			m.transferAmountInput.Focus()
		case 2:
			m.transferPrivateInput.Focus()
		}
	case "balance":
		m.balanceAddressInput.Blur()
		m.balanceAddressInput.Focus()
	}
}

// forwardToActiveInput pipes a message into the active view's text inputs.
func (m *model) forwardToActiveInput(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch m.view {
	case "create":
		var c1, c2, c3, c4 tea.Cmd
		m.createNameInput, c1 = m.createNameInput.Update(msg)
		m.createSymbolInput, c2 = m.createSymbolInput.Update(msg)
		m.createSupplyInput, c3 = m.createSupplyInput.Update(msg)
		m.createDecimalsInput, c4 = m.createDecimalsInput.Update(msg)
		cmds = append(cmds, c1, c2, c3, c4)
	case "mint":
		var c1, c2, c3 tea.Cmd
		m.mintToInput, c1 = m.mintToInput.Update(msg)
		m.mintAmountInput, c2 = m.mintAmountInput.Update(msg)
		m.mintPrivateInput, c3 = m.mintPrivateInput.Update(msg)
		cmds = append(cmds, c1, c2, c3)
	case "transfer":
		var c1, c2, c3 tea.Cmd
		m.transferToInput, c1 = m.transferToInput.Update(msg)
		m.transferAmountInput, c2 = m.transferAmountInput.Update(msg)
		m.transferPrivateInput, c3 = m.transferPrivateInput.Update(msg)
		cmds = append(cmds, c1, c2, c3)
	case "balance":
		var c1 tea.Cmd
		m.balanceAddressInput, c1 = m.balanceAddressInput.Update(msg)
		cmds = append(cmds, c1)
	}
	return tea.Batch(cmds...)
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	if m.showHelp {
		v.SetContent(components.HelpView())
	} else {
		switch m.view {
		case "menu":
			v.SetContent(m.menuView())
		case "create":
			v.SetContent(m.createView())
		case "mint":
			v.SetContent(m.mintView())
		case "transfer":
			v.SetContent(m.transferView())
		case "balance":
			v.SetContent(m.balanceView())
		case "history":
			v.SetContent(m.historyView())
		default:
			v.SetContent(m.menuView())
		}
	}
	v.AltScreen = true
	return v
}

func (m *model) handleSelect() {
	switch m.menuIndex {
	case 0:
		m.view = "create"
		m.err = ""
		m.successMsg = ""
		m.createNameInput.SetValue("")
		m.createSymbolInput.SetValue("")
		m.createSupplyInput.SetValue("")
		m.createDecimalsInput.SetValue("8")
		m.inputFocus = 0
		m.updateInputFocus()
	case 1:
		m.view = "mint"
		m.err = ""
		m.successMsg = ""
		m.mintToInput.SetValue("")
		m.mintAmountInput.SetValue("")
		m.mintPrivateInput.SetValue("")
		m.inputFocus = 0
		m.updateInputFocus()
	case 2:
		m.view = "transfer"
		m.err = ""
		m.successMsg = ""
		m.transferToInput.SetValue("")
		m.transferAmountInput.SetValue("")
		m.transferPrivateInput.SetValue("")
		m.inputFocus = 0
		m.updateInputFocus()
	case 3:
		m.view = "balance"
		m.err = ""
		m.successMsg = ""
		m.balanceAddressInput.SetValue("")
		m.inputFocus = 0
		m.updateInputFocus()
	case 4:
		m.loadHistory()
		m.view = "history"
	}
}

func (m *model) menuView() string {
	s := components.HeaderStyle().Render("🪙 "+i18n.GetText("token.tui.title")) + "\n\n"

	menuItems := []string{
		"📦 " + i18n.GetText("token.tui.create"),
		"✨ " + i18n.GetText("token.tui.mint"),
		"💸 " + i18n.GetText("token.tui.transfer"),
		"💰 " + i18n.GetText("token.tui.query"),
		"📜 " + i18n.GetText("token.history.cmd"),
	}

	for i, item := range menuItems {
		if i == m.menuIndex {
			s += components.MenuActiveStyle().Render("▶ " + item + "\n")
		} else {
			s += components.MenuInactiveStyle().Render("  " + item + "\n")
		}
	}

	s += "\n" + components.HelpTextStyle().Render(i18n.GetText("help.nav"))

	return s
}

func (m *model) createView() string {
	s := components.HeaderStyle().Render("📦 "+i18n.GetText("token.tui.create")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("token.name")+":") + "\n"
	s += m.createNameInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("token.symbol")+":") + "\n"
	s += m.createSymbolInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("token.supply")+":") + "\n"
	s += m.createSupplyInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("token.decimals")+":") + "\n"
	s += m.createDecimalsInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.create_btn")) + " | " + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) mintView() string {
	s := components.HeaderStyle().Render("✨ "+i18n.GetText("token.tui.mint")) + "\n\n"

	if m.currentToken == nil {
		s += components.WarningStyle().Render(i18n.GetText("token.tui.no_token")) + "\n\n"
	} else {
		s += components.InfoStyle().Render(i18n.GetText("token.tui.token_label")+": "+m.currentToken.Symbol()+" ("+m.currentToken.Name()+")") + "\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.to")+":") + "\n"
		s += m.mintToInput.View() + "\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.amount")+":") + "\n"
		s += m.mintAmountInput.View() + "\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.private_key")+":") + "\n"
		s += m.mintPrivateInput.View() + "\n\n"
	}

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.create_btn")) + " | " + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) transferView() string {
	s := components.HeaderStyle().Render("💸 "+i18n.GetText("token.tui.transfer")) + "\n\n"

	if m.currentToken == nil {
		s += components.WarningStyle().Render(i18n.GetText("token.tui.no_token")) + "\n\n"
	} else {
		ownerB64 := base64.StdEncoding.EncodeToString(m.ownerKey)
		s += components.InfoStyle().Render(i18n.GetText("token.tui.from_label")+": ") + ownerB64[:min(20, len(ownerB64))] + "...\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.to")+":") + "\n"
		s += m.transferToInput.View() + "\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.amount")+":") + "\n"
		s += m.transferAmountInput.View() + "\n\n"
		s += components.InfoStyle().Render(i18n.GetText("token.private_key")+":") + "\n"
		s += m.transferPrivateInput.View() + "\n\n"
	}

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.create_btn")) + " | " + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) balanceView() string {
	s := components.HeaderStyle().Render("💰 "+i18n.GetText("token.tui.query")) + "\n\n"

	s += components.InfoStyle().Render(i18n.GetText("token.owner")+":") + "\n"
	s += m.balanceAddressInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += m.successMsg + "\n\n"
	}

	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.create_btn")) + " | " + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) historyView() string {
	s := components.HeaderStyle().Render("📜 "+i18n.GetText("token.history.cmd")) + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) handleCreate() {
	name := m.createNameInput.Value()
	symbol := m.createSymbolInput.Value()
	supplyStr := m.createSupplyInput.Value()
	decimalsStr := m.createDecimalsInput.Value()

	if name == "" {
		m.err = i18n.GetText("error.name_required")
		return
	}
	if symbol == "" {
		m.err = i18n.GetText("error.symbol_required")
		return
	}
	if supplyStr == "" {
		m.err = i18n.GetText("error.supply_required")
		return
	}

	supply, ok := new(big.Int).SetString(supplyStr, 10)
	if !ok {
		m.err = i18n.GetText("error.invalid_supply")
		return
	}

	// Decimals is int8 (0..127 per ValidateTokenDecimals) and 0 is the
	// "unset, use defaultDecimals" sentinel. Parse the field once and clamp it
	// to that range instead of using it only as a validity check — the
	// previous code validated but never assigned, so a create with "18"
	// silently produced an 8-decimal token (TASK-162, ISS-155).
	var decimals int8
	if decimalsStr != "" {
		d, err := strconv.Atoi(decimalsStr)
		if err != nil {
			m.err = i18n.GetText("error.invalid_decimals")
			return
		}
		if d < 0 || d > math.MaxInt8 {
			m.err = i18n.GetText("error.invalid_decimals")
			return
		}
		decimals = int8(d)
	}

	req := &token.CreateTokenRequest{
		Name:        name,
		Symbol:      symbol,
		TotalSupply: &token.Amount{Int: supply},
		Owner:       m.ownerKey,
		Decimals:    decimals,
	}

	tok, err := m.tokenService.CreateToken(req)
	if err != nil {
		m.err = err.Error()
		return
	}

	m.currentToken = tok
	m.successMsg = fmt.Sprintf(i18n.GetText("token.created"), tok.ID(), tok.Name(), tok.Symbol())
}

func (m *model) handleMint() {
	if m.currentToken == nil {
		m.err = i18n.GetText("error.create_token_first")
		return
	}

	toStr := m.mintToInput.Value()
	amountStr := m.mintAmountInput.Value()
	privateStr := m.mintPrivateInput.Value()

	if toStr == "" {
		m.err = i18n.GetText("error.address_required")
		return
	}
	if amountStr == "" {
		m.err = i18n.GetText("error.amount_required")
		return
	}

	to, err := base64.StdEncoding.DecodeString(toStr)
	if err != nil {
		m.err = i18n.GetText("error.invalid_address")
		return
	}

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		m.err = i18n.GetText("error.invalid_amount")
		return
	}

	var priv []byte
	if privateStr != "" {
		priv, err = base64.StdEncoding.DecodeString(privateStr)
		if err != nil {
			m.err = i18n.GetText("error.invalid_privkey")
			return
		}
	} else {
		priv = m.ownerPriv
	}

	req := &token.MintRequest{
		TokenID:    m.currentToken.ID(),
		To:         to,
		Amount:     &token.Amount{Int: amount},
		PrivateKey: priv,
	}

	_, err = m.tokenService.Mint(req)
	if err != nil {
		m.err = err.Error()
		return
	}

	m.successMsg = i18n.GetText("token.minted")
}

func (m *model) handleTransfer() {
	if m.currentToken == nil {
		m.err = i18n.GetText("error.create_token_first")
		return
	}

	toStr := m.transferToInput.Value()
	amountStr := m.transferAmountInput.Value()
	privateStr := m.transferPrivateInput.Value()

	if toStr == "" {
		m.err = i18n.GetText("error.address_required")
		return
	}
	if amountStr == "" {
		m.err = i18n.GetText("error.amount_required")
		return
	}

	to, err := base64.StdEncoding.DecodeString(toStr)
	if err != nil {
		m.err = i18n.GetText("error.invalid_address")
		return
	}

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		m.err = i18n.GetText("error.invalid_amount")
		return
	}

	var priv []byte
	if privateStr != "" {
		priv, err = base64.StdEncoding.DecodeString(privateStr)
		if err != nil {
			m.err = i18n.GetText("error.invalid_privkey")
			return
		}
	} else {
		priv = m.ownerPriv
	}

	req := &token.TransferRequest{
		TokenID:    m.currentToken.ID(),
		From:       m.ownerKey,
		To:         to,
		Amount:     &token.Amount{Int: amount},
		PrivateKey: priv,
	}

	_, err = m.tokenService.Transfer(req)
	if err != nil {
		m.err = err.Error()
		return
	}

	m.successMsg = i18n.GetText("token.transferred")
}

func (m *model) handleBalance() {
	addressStr := m.balanceAddressInput.Value()

	if m.currentToken == nil {
		m.err = i18n.GetText("error.create_token_first")
		return
	}

	var owner token.PublicKey
	var err error
	if addressStr != "" {
		owner, err = base64.StdEncoding.DecodeString(addressStr)
		if err != nil {
			m.err = i18n.GetText("error.invalid_address")
			return
		}
	} else {
		owner = m.ownerKey
	}

	balance, err := m.tokenService.GetBalance(m.currentToken.ID(), owner)
	if err != nil {
		m.err = err.Error()
		return
	}

	addrB64 := base64.StdEncoding.EncodeToString(owner)
	m.successMsg = components.CardStyle().Render(
		components.KeyValue(i18n.GetText("token.tui.token_label"), m.currentToken.Symbol()+" ("+m.currentToken.Name()+")") + "\n\n" +
			components.SuccessStyle().Render(i18n.GetText("token.tui.balance_label")+": "+balance.String()+" "+m.currentToken.Symbol()) + "\n\n" +
			components.KeyValue(i18n.GetText("token.tui.address_label"), addrB64[:min(20, len(addrB64))]+"..."),
	)
}

func (m *model) loadHistory() {
	if m.currentToken == nil {
		m.viewport.SetContent(i18n.GetText("token.tui.no_tokens_view") + "\n\n" +
			components.HelpTextStyle().Render(i18n.GetText("token.tui.create_hint")))
		return
	}

	events, err := m.tokenService.GetTransferHistory(m.currentToken.ID(), m.ownerKey, 50, 0)
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf(i18n.GetText("token.tui.history_failed"), err.Error()))
		return
	}

	if len(events) == 0 {
		m.viewport.SetContent(i18n.GetText("token.tui.no_transfers") + "\n\n" +
			components.HelpTextStyle().Render(i18n.GetText("token.tui.transfer_hint")))
		return
	}

	var content string
	for i, e := range events {
		fromB64 := base64.StdEncoding.EncodeToString(e.From())
		toB64 := base64.StdEncoding.EncodeToString(e.To())
		content += fmt.Sprintf(i18n.GetText("token.tui.transfer_header"), i+1) + "\n"
		content += fmt.Sprintf(i18n.GetText("token.tui.from_b64"), fromB64[:min(10, len(fromB64))]) + "\n"
		content += fmt.Sprintf(i18n.GetText("token.tui.to_b64"), toB64[:min(10, len(toB64))]) + "\n"
		content += fmt.Sprintf(i18n.GetText("token.tui.amount_qty"), e.Amount().String(), m.currentToken.Symbol()) + "\n\n"
	}
	m.viewport.SetContent(content)
}

type noOpTxManager struct{}

func (n *noOpTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

type inmemRepo struct {
	tokens    map[token.TokenID]*token.Token
	balances  map[string]*token.Amount
	approvals map[string]*token.Approval
}

// WithTx satisfies TransactableRepository. The TUI runs single-goroutine and
// uses a no-op transaction manager (tx is always nil), so the tx-scoped
// repository is the repository itself.
func (r *inmemRepo) WithTx(_ *sql.Tx) token.Repository {
	return r
}

// SaveToken insert-only, mirroring the SQLite repo's atomic create semantics
// (ISS-098): a token's ID is its symbol, so an existing ID rejects the create
// instead of silently overwriting the row.
func (r *inmemRepo) SaveToken(tok *token.Token) error {
	if _, exists := r.tokens[tok.ID()]; exists {
		return token.ErrTokenExists
	}
	r.tokens[tok.ID()] = tok
	return nil
}

func (r *inmemRepo) GetToken(id token.TokenID) (*token.Token, error) {
	return r.tokens[id], nil
}

func (r *inmemRepo) SaveApproval(approval *token.Approval) error {
	key := string(approval.TokenID()) + string(approval.Owner()) + string(approval.Spender())
	r.approvals[key] = approval
	return nil
}

func (r *inmemRepo) GetApproval(tokenID token.TokenID, owner, spender token.PublicKey) (*token.Approval, error) {
	key := string(tokenID) + string(owner) + string(spender)
	return r.approvals[key], nil
}

func (r *inmemRepo) GetApprovalsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.Approval, error) {
	var result []*token.Approval
	for _, approval := range r.approvals {
		if approval.TokenID() == tokenID && string(approval.Owner()) == string(owner) {
			result = append(result, approval)
		}
	}
	return result, nil
}

func (r *inmemRepo) GetAccountBalance(tokenID token.TokenID, owner token.PublicKey) (*token.Amount, error) {
	key := string(tokenID) + string(owner)
	if balance, ok := r.balances[key]; ok {
		return balance, nil
	}
	return token.NewAmount(0), nil
}

func (r *inmemRepo) SetAccountBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) error {
	key := string(tokenID) + string(owner)
	r.balances[key] = amount
	return nil
}

// TrySubtractBalance is the TUI's in-memory equivalent of the
// SQLite primitive. The TUI is single-goroutine, so we don't need
// the same locking the SQLite path uses, but the semantics must
// match: return ErrInsufficientAllowance-style sentinel for
// over-spend, so callers can use errors.Is uniformly.
func (r *inmemRepo) TrySubtractBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	key := string(tokenID) + string(owner)
	cur, ok := r.balances[key]
	if !ok {
		cur = token.NewAmount(0)
	}
	if cur.Int.Cmp(amount.Int) < 0 {
		return nil, fmt.Errorf("try subtract balance: %w", token.ErrInsufficientBalance)
	}
	newBal := &token.Amount{Int: new(big.Int).Sub(cur.Int, amount.Int)}
	r.balances[key] = newBal
	return newBal, nil
}

func (r *inmemRepo) TryAddBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	key := string(tokenID) + string(owner)
	cur, ok := r.balances[key]
	if !ok {
		cur = token.NewAmount(0)
	}
	newBal := &token.Amount{Int: new(big.Int).Add(cur.Int, amount.Int)}
	// Mirror the SQLite primitive: refuse to push the balance past MaxInt64.
	if newBal.BitLen() > 63 {
		return nil, fmt.Errorf("try add balance: balance would exceed maximum")
	}
	r.balances[key] = newBal
	return newBal, nil
}

func (r *inmemRepo) TryAddToSupply(id token.TokenID, amount *token.Amount) (*token.Amount, error) {
	tok, ok := r.tokens[id]
	if !ok {
		return nil, token.ErrTokenNotFound
	}
	newSupply := &token.Amount{Int: new(big.Int).Add(tok.TotalSupply().Int, amount.Int)}
	// Mirror the SQLite primitive: refuse to push total_supply past MaxInt64.
	if newSupply.BitLen() > 63 {
		return nil, fmt.Errorf("try add to supply: total supply would exceed maximum")
	}
	r.tokens[id] = token.NewTokenWithDecimals(id, tok.Name(), tok.Symbol(), newSupply, tok.Owner(), tok.Decimals())
	return newSupply, nil
}

// TrySubtractFromSupply is the TUI's in-memory equivalent of the
// SQLite primitive used by Burn: it decrements total_supply, failing
// if the supply is below the burned amount so the ledger invariant
// total_supply == sum of balances holds.
func (r *inmemRepo) TrySubtractFromSupply(id token.TokenID, amount *token.Amount) (*token.Amount, error) {
	tok, ok := r.tokens[id]
	if !ok {
		return nil, token.ErrTokenNotFound
	}
	if tok.TotalSupply().Cmp(amount) < 0 {
		return nil, fmt.Errorf("try subtract supply: total supply below burn amount")
	}
	newSupply := &token.Amount{Int: new(big.Int).Sub(tok.TotalSupply().Int, amount.Int)}
	r.tokens[id] = token.NewTokenWithDecimals(id, tok.Name(), tok.Symbol(), newSupply, tok.Owner(), tok.Decimals())
	return newSupply, nil
}

func (r *inmemRepo) TryDeductApproval(tokenID token.TokenID, owner, spender token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	key := string(tokenID) + string(owner) + string(spender)
	cur, ok := r.approvals[key]
	if !ok {
		return nil, fmt.Errorf("try deduct approval: %w", token.ErrInsufficientAllowance)
	}
	if cur.Amount().Int.Cmp(amount.Int) < 0 {
		return nil, fmt.Errorf("try deduct approval: %w", token.ErrInsufficientAllowance)
	}
	newAmt := &token.Amount{Int: new(big.Int).Sub(cur.Amount().Int, amount.Int)}
	r.approvals[key] = token.NewApproval(tokenID, owner, spender, newAmt)
	return newAmt, nil
}

func (r *inmemRepo) TryAdjustApproval(tokenID token.TokenID, owner, spender token.PublicKey, delta *token.Amount) (*token.Amount, error) {
	key := string(tokenID) + string(owner) + string(spender)
	cur, ok := r.approvals[key]
	var curAmount *token.Amount
	if ok {
		curAmount = cur.Amount()
	} else {
		curAmount = token.NewAmount(0)
	}
	newAmt := &token.Amount{Int: new(big.Int).Add(curAmount.Int, delta.Int)}
	if newAmt.Sign() < 0 {
		newAmt = token.NewAmount(0)
	}
	// Mirror the SQLite primitive's ceiling clamp at MaxInt64.
	if newAmt.BitLen() > 63 {
		newAmt = &token.Amount{Int: new(big.Int).SetInt64(math.MaxInt64)}
	}
	r.approvals[key] = token.NewApproval(tokenID, owner, spender, newAmt)
	return newAmt, nil
}

type inmemEventStore struct {
	transferEvents []*token.TransferEvent
	mintEvents     []*token.MintEvent
	burnEvents     []*token.BurnEvent
	approveEvents  []*token.ApproveEvent
}

func (e *inmemEventStore) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey, limit, offset int) ([]*token.TransferEvent, error) {
	var result []*token.TransferEvent
	for _, ev := range e.transferEvents {
		if ev.TokenID() == tokenID && (string(ev.From()) == string(owner) || string(ev.To()) == string(owner)) {
			result = append(result, ev)
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if offset >= len(result) {
		return []*token.TransferEvent{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (e *inmemEventStore) GetMintEventsByToken(tokenID token.TokenID) ([]*token.MintEvent, error) {
	var result []*token.MintEvent
	for _, ev := range e.mintEvents {
		if ev.TokenID() == tokenID {
			result = append(result, ev)
		}
	}
	return result, nil
}

func (e *inmemEventStore) GetBurnEventsByToken(tokenID token.TokenID) ([]*token.BurnEvent, error) {
	var result []*token.BurnEvent
	for _, ev := range e.burnEvents {
		if ev.TokenID() == tokenID {
			result = append(result, ev)
		}
	}
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RunTokenTUI() error {
	p := tea.NewProgram(NewTokenApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Token TUI: %v\n", err)
		return err
	}
	return nil
}
