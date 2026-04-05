package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ui/components"
)

type model struct {
	view         string
	menuIndex    int
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
		nonces:         make(map[string]uint64),
	}
	tokenService := token.NewService(repo, eventStore, chain)

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
		switch msg.String() {
		case "q", "ctrl+c":
			if m.view == "menu" {
				return m, tea.Quit
			}
			m.view = "menu"
			m.err = ""
			m.successMsg = ""
			return m, nil

		case "up", "k":
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.view == "menu" && m.menuIndex < 4 {
				m.menuIndex++
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

	if m.view == "create" {
		m.createNameInput, _ = m.createNameInput.Update(msg)
		m.createSymbolInput, _ = m.createSymbolInput.Update(msg)
		m.createSupplyInput, _ = m.createSupplyInput.Update(msg)
		m.createDecimalsInput, cmd = m.createDecimalsInput.Update(msg)
	}

	return m, cmd
}

func (m *model) View() tea.View {
	v := tea.NewView("")
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
	case 1:
		m.view = "mint"
		m.err = ""
		m.successMsg = ""
		m.mintToInput.SetValue("")
		m.mintAmountInput.SetValue("")
		m.mintPrivateInput.SetValue("")
	case 2:
		m.view = "transfer"
		m.err = ""
		m.successMsg = ""
		m.transferToInput.SetValue("")
		m.transferAmountInput.SetValue("")
		m.transferPrivateInput.SetValue("")
	case 3:
		m.view = "balance"
		m.err = ""
		m.successMsg = ""
		m.balanceAddressInput.SetValue("")
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
		s += components.WarningStyle().Render("请先创建代币") + "\n\n"
	} else {
		s += components.InfoStyle().Render("代币: "+m.currentToken.Symbol()+" ("+m.currentToken.Name()+")") + "\n\n"
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
		s += components.WarningStyle().Render("请先创建代币") + "\n\n"
	} else {
		ownerB64 := base64.StdEncoding.EncodeToString(m.ownerKey)
		s += components.InfoStyle().Render("从: ") + ownerB64[:min(20, len(ownerB64))] + "...\n\n"
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
		m.err = "名称不能为空"
		return
	}
	if symbol == "" {
		m.err = "符号不能为空"
		return
	}
	if supplyStr == "" {
		m.err = "供应量不能为空"
		return
	}

	supply, ok := new(big.Int).SetString(supplyStr, 10)
	if !ok {
		m.err = "无效的供应量"
		return
	}

	if decimalsStr != "" {
		if _, err := strconv.Atoi(decimalsStr); err != nil {
			m.err = "无效的小数位数"
			return
		}
	}

	req := &token.CreateTokenRequest{
		Name:        name,
		Symbol:      symbol,
		TotalSupply: &token.Amount{Int: supply},
		Owner:       m.ownerKey,
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
		m.err = "请先创建代币"
		return
	}

	toStr := m.mintToInput.Value()
	amountStr := m.mintAmountInput.Value()
	privateStr := m.mintPrivateInput.Value()

	if toStr == "" {
		m.err = "接收地址不能为空"
		return
	}
	if amountStr == "" {
		m.err = "数量不能为空"
		return
	}

	to, err := base64.StdEncoding.DecodeString(toStr)
	if err != nil {
		m.err = "无效的接收地址"
		return
	}

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		m.err = "无效的数量"
		return
	}

	var priv []byte
	if privateStr != "" {
		priv, err = base64.StdEncoding.DecodeString(privateStr)
		if err != nil {
			m.err = "无效的私钥"
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
		m.err = "请先创建代币"
		return
	}

	toStr := m.transferToInput.Value()
	amountStr := m.transferAmountInput.Value()
	privateStr := m.transferPrivateInput.Value()

	if toStr == "" {
		m.err = "接收地址不能为空"
		return
	}
	if amountStr == "" {
		m.err = "数量不能为空"
		return
	}

	to, err := base64.StdEncoding.DecodeString(toStr)
	if err != nil {
		m.err = "无效的接收地址"
		return
	}

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		m.err = "无效的数量"
		return
	}

	var priv []byte
	if privateStr != "" {
		priv, err = base64.StdEncoding.DecodeString(privateStr)
		if err != nil {
			m.err = "无效的私钥"
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
		m.err = "请先创建代币"
		return
	}

	var owner token.PublicKey
	var err error
	if addressStr != "" {
		owner, err = base64.StdEncoding.DecodeString(addressStr)
		if err != nil {
			m.err = "无效的地址"
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
		components.KeyValue("代币", m.currentToken.Symbol()+" ("+m.currentToken.Name()+")") + "\n\n" +
			components.SuccessStyle().Render("余额: "+balance.String()+" "+m.currentToken.Symbol()) + "\n\n" +
			components.KeyValue("地址", addrB64[:min(20, len(addrB64))]+"..."),
	)
}

func (m *model) loadHistory() {
	if m.currentToken == nil {
		m.viewport.SetContent("暂无代币\n\n" + components.HelpTextStyle().Render("使用 '创建代币' 创建代币"))
		return
	}

	events, err := m.tokenService.GetTransferHistory(m.currentToken.ID(), m.ownerKey, 50)
	if err != nil {
		m.viewport.SetContent("加载历史失败: " + err.Error())
		return
	}

	if len(events) == 0 {
		m.viewport.SetContent("暂无转账记录\n\n" + components.HelpTextStyle().Render("进行转账操作后会显示记录"))
		return
	}

	var content string
	for i, e := range events {
		fromB64 := base64.StdEncoding.EncodeToString(e.From())
		toB64 := base64.StdEncoding.EncodeToString(e.To())
		content += fmt.Sprintf("--- 转账 #%d ---\n", i+1)
		content += fmt.Sprintf("从: %s...\n", fromB64[:min(10, len(fromB64))])
		content += fmt.Sprintf("到: %s...\n", toB64[:min(10, len(toB64))])
		content += fmt.Sprintf("数量: %s %s\n\n", e.Amount().String(), m.currentToken.Symbol())
	}
	m.viewport.SetContent(content)
}

type inmemRepo struct {
	tokens    map[token.TokenID]*token.Token
	balances  map[string]*token.Amount
	approvals map[string]*token.Approval
}

func (r *inmemRepo) SaveToken(tok *token.Token) error {
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

type inmemEventStore struct {
	transferEvents []*token.TransferEvent
	mintEvents     []*token.MintEvent
	burnEvents     []*token.BurnEvent
	approveEvents  []*token.ApproveEvent
	nonces         map[string]uint64
}

func (e *inmemEventStore) SaveTransferEvent(event *token.TransferEvent) error {
	e.transferEvents = append(e.transferEvents, event)
	key := string(event.TokenID()) + string(event.From())
	e.nonces[key] = event.Nonce()
	return nil
}

func (e *inmemEventStore) SaveMintEvent(event *token.MintEvent) error {
	e.mintEvents = append(e.mintEvents, event)
	return nil
}

func (e *inmemEventStore) SaveBurnEvent(event *token.BurnEvent) error {
	e.burnEvents = append(e.burnEvents, event)
	return nil
}

func (e *inmemEventStore) SaveApproveEvent(event *token.ApproveEvent) error {
	e.approveEvents = append(e.approveEvents, event)
	return nil
}

func (e *inmemEventStore) GetTransferEventsByToken(tokenID token.TokenID) ([]*token.TransferEvent, error) {
	var result []*token.TransferEvent
	for _, ev := range e.transferEvents {
		if ev.TokenID() == tokenID {
			result = append(result, ev)
		}
	}
	return result, nil
}

func (e *inmemEventStore) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.TransferEvent, error) {
	var result []*token.TransferEvent
	for _, ev := range e.transferEvents {
		if ev.TokenID() == tokenID && (string(ev.From()) == string(owner) || string(ev.To()) == string(owner)) {
			result = append(result, ev)
		}
	}
	return result, nil
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

func (e *inmemEventStore) GetLastNonce(tokenID token.TokenID, owner token.PublicKey) (uint64, error) {
	key := string(tokenID) + string(owner)
	return e.nonces[key], nil
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
