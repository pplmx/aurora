package nft

import (
	"encoding/base64"
	"fmt"
	"os"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ui/components"
)

type model struct {
	view      string
	menuIndex int

	nameInput   textinput.Model
	descInput   textinput.Model
	pubkeyInput textinput.Model

	nftIDInput   textinput.Model
	fromKeyInput textinput.Model
	toAddrInput  textinput.Model

	queryIDInput textinput.Model

	ownerInput textinput.Model

	viewport   viewport.Model
	nft        *nft.NFT
	err        string
	successMsg string

	chain      *blockchain.BlockChain
	nftService *nft.NFTService
}

func NewNFTApp() *model {
	chain := blockchain.InitBlockChain()

	repo := nft.NewInmemRepo()
	nftService := nft.NewService(repo)

	nameInput := textinput.New()
	nameInput.Placeholder = i18n.GetText("nft.tui.name")
	nameInput.Focus()
	nameInput.Prompt = "  "

	descInput := textinput.New()
	descInput.Placeholder = i18n.GetText("nft.tui.description")
	descInput.Prompt = "  "

	pubkeyInput := textinput.New()
	pubkeyInput.Placeholder = i18n.GetText("nft.tui.public_key")
	pubkeyInput.Prompt = "  "

	nftIDInput := textinput.New()
	nftIDInput.Placeholder = i18n.GetText("nft.tui.nft_id")
	nftIDInput.Focus()
	nftIDInput.Prompt = "  "

	fromKeyInput := textinput.New()
	fromKeyInput.Placeholder = i18n.GetText("nft.tui.private_key")
	fromKeyInput.Prompt = "  "

	toAddrInput := textinput.New()
	toAddrInput.Placeholder = i18n.GetText("nft.tui.to_address")
	toAddrInput.Prompt = "  "

	queryIDInput := textinput.New()
	queryIDInput.Placeholder = i18n.GetText("nft.tui.nft_id")
	queryIDInput.Focus()
	queryIDInput.Prompt = "  "

	ownerInput := textinput.New()
	ownerInput.Placeholder = i18n.GetText("nft.tui.public_key")
	ownerInput.Focus()
	ownerInput.Prompt = "  "

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))

	return &model{
		view:       "menu",
		menuIndex:  0,
		chain:      chain,
		nftService: nftService,

		nameInput:   nameInput,
		descInput:   descInput,
		pubkeyInput: pubkeyInput,

		nftIDInput:   nftIDInput,
		fromKeyInput: fromKeyInput,
		toAddrInput:  toAddrInput,

		queryIDInput: queryIDInput,

		ownerInput: ownerInput,

		viewport: vp,
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
		case "ctrl+c", "q":
			if m.view == "menu" {
				return m, tea.Quit
			}
			m.view = "menu"
			m.err = ""
			m.successMsg = ""

		case "up", "k":
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.view == "menu" && m.menuIndex < 3 {
				m.menuIndex++
			}

		case "enter":
			switch m.view {
			case "menu":
				switch m.menuIndex {
				case 0:
					m.view = "mint"
					m.err = ""
					m.successMsg = ""
					m.nameInput.SetValue("")
					m.descInput.SetValue("")
					m.pubkeyInput.SetValue("")
				case 1:
					m.view = "transfer"
					m.err = ""
					m.successMsg = ""
					m.nftIDInput.SetValue("")
					m.fromKeyInput.SetValue("")
					m.toAddrInput.SetValue("")
				case 2:
					m.view = "query"
					m.err = ""
					m.successMsg = ""
					m.queryIDInput.SetValue("")
				case 3:
					return m, tea.Quit
				}
			case "mint":
				return m, m.handleMint
			case "transfer":
				return m, m.handleTransfer
			case "query":
				return m, m.handleQuery
			case "list":
				m.loadNFTsByOwner()
				m.view = "list"
			case "result":
				m.view = "menu"
				m.successMsg = ""
				m.nft = nil
			}
		case "1", "2", "3", "4":
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

	return m, cmd
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	switch m.view {
	case "menu":
		v.SetContent(m.menuView())
	case "mint":
		v.SetContent(m.mintView())
	case "transfer":
		v.SetContent(m.transferView())
	case "query":
		v.SetContent(m.queryView())
	case "result":
		v.SetContent(m.resultView())
	case "list":
		v.SetContent(m.listView())
	default:
		v.SetContent(m.menuView())
	}
	v.AltScreen = true
	return v
}

func (m *model) menuView() string {
	s := components.HeaderStyle().Render("🖼️ "+i18n.GetText("nft.tui.title")+" 🖼️") + "\n\n"
	items := []string{
		i18n.GetText("nft.tui.mint"),
		i18n.GetText("nft.tui.transfer"),
		i18n.GetText("nft.tui.query"),
		i18n.GetText("lottery.tui.exit"),
	}
	for i, item := range items {
		if i == m.menuIndex {
			s += components.MenuActiveStyle().Render("▶ " + item + "\n")
		} else {
			s += components.MenuInactiveStyle().Render("  " + item + "\n")
		}
	}
	s += "\n" + components.HelpTextStyle().Render(i18n.GetText("help.nav"))
	return s
}

func (m *model) mintView() string {
	s := components.HeaderStyle().Render("⛏️ "+i18n.GetText("nft.tui.mint")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.name")+":") + "\n"
	s += m.nameInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.description")+":") + "\n"
	s += m.descInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.public_key")+":") + "\n"
	s += m.pubkeyInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render("[Enter] "+i18n.GetText("lottery.tui.create")) + " | " + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) transferView() string {
	s := components.HeaderStyle().Render("🔄 "+i18n.GetText("nft.tui.transfer")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.nft_id")+":") + "\n"
	s += m.nftIDInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.private_key")+":") + "\n"
	s += m.fromKeyInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.to_address")+":") + "\n"
	s += m.toAddrInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render("[Enter] "+i18n.GetText("lottery.tui.confirm")) + " | " + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) queryView() string {
	s := components.HeaderStyle().Render("🔍 "+i18n.GetText("nft.tui.query")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.nft_id")+":") + "\n"
	s += m.queryIDInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	s += components.BorderStyle().Render("[Enter] "+i18n.GetText("lottery.tui.search")) + " | " + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) resultView() string {
	if m.nft == nil {
		s := components.ErrorStyle().Render("⚠ "+i18n.GetText("error.not_found")) + "\n\n"
		s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))
		return s
	}

	s := components.SuccessStyle().Render("🎉 "+i18n.GetText("nft.tui.nft_detail")) + "\n\n"
	s += components.InfoStyle().Render("ID: ") + m.nft.ID + "\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.name")+": ") + m.nft.Name + "\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.description")+": ") + m.nft.Description + "\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.owner")+": ") + base64.StdEncoding.EncodeToString(m.nft.Owner) + "\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.block_height")+": #") + fmt.Sprintf("%d", m.nft.BlockHeight) + "\n"

	s += "\n" + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) listView() string {
	s := components.HeaderStyle().Render("📜 "+i18n.GetText("nft.tui.nft_list")) + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) handleMint() tea.Msg {
	name := m.nameInput.Value()
	description := m.descInput.Value()
	pubkeyStr := m.pubkeyInput.Value()

	if name == "" {
		m.err = i18n.GetText("error.name_required")
		return nil
	}

	if pubkeyStr == "" {
		m.err = i18n.GetText("error.pubkey_required")
		return nil
	}

	pubkey, err := base64.StdEncoding.DecodeString(pubkeyStr)
	if err != nil {
		m.err = i18n.GetText("error.invalid_pubkey")
		return nil
	}

	newNFT := nft.NewNFT(name, description, "", "", pubkey, pubkey)
	result, err := m.nftService.Mint(newNFT, m.chain)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	m.nft = result
	m.successMsg = i18n.GetText("nft.tui.mint_success")
	m.view = "result"

	return nil
}

func (m *model) handleTransfer() tea.Msg {
	nftID := m.nftIDInput.Value()
	fromKeyStr := m.fromKeyInput.Value()
	toAddrStr := m.toAddrInput.Value()

	if nftID == "" {
		m.err = i18n.GetText("error.nft_id_required")
		return nil
	}

	if fromKeyStr == "" {
		m.err = i18n.GetText("error.privkey_required")
		return nil
	}

	if toAddrStr == "" {
		m.err = i18n.GetText("error.to_address_required")
		return nil
	}

	fromKey, err := base64.StdEncoding.DecodeString(fromKeyStr)
	if err != nil {
		m.err = i18n.GetText("error.invalid_privkey")
		return nil
	}

	toAddr, err := base64.StdEncoding.DecodeString(toAddrStr)
	if err != nil {
		m.err = i18n.GetText("error.invalid_address")
		return nil
	}

	if len(fromKey) != 32 {
		m.err = i18n.GetText("error.invalid_privkey")
		return nil
	}

	fromPubKey := fromKey[32:]
	_, err = m.nftService.Transfer(nftID, fromPubKey, toAddr, fromKey, m.chain)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	m.successMsg = i18n.GetText("nft.tui.transfer_success")
	m.view = "result"

	return nil
}

func (m *model) handleQuery() tea.Msg {
	nftID := m.queryIDInput.Value()

	if nftID == "" {
		m.err = i18n.GetText("error.nft_id_required")
		return nil
	}

	result, err := m.nftService.GetNFTByID(nftID)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	if result == nil {
		m.err = i18n.GetText("error.not_found")
		return nil
	}

	m.nft = result
	m.view = "result"

	return nil
}

func (m *model) loadNFTsByOwner() {
	ownerStr := m.ownerInput.Value()
	if ownerStr == "" {
		m.viewport.SetContent(components.ErrorStyle().Render("⚠ "+i18n.GetText("error.pubkey_required")) + "\n\n" + components.HelpTextStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back")))
		return
	}

	owner, err := base64.StdEncoding.DecodeString(ownerStr)
	if err != nil {
		m.viewport.SetContent(components.ErrorStyle().Render("⚠ "+i18n.GetText("error.invalid_pubkey")) + "\n\n" + components.HelpTextStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back")))
		return
	}

	nfts, err := m.nftService.GetNFTsByOwner(owner)
	if err != nil {
		m.viewport.SetContent(components.ErrorStyle().Render("⚠ "+err.Error()) + "\n\n" + components.HelpTextStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back")))
		return
	}

	if len(nfts) == 0 {
		m.viewport.SetContent(i18n.GetText("nft.tui.no_nfts") + "\n\n" + components.HelpTextStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back")))
		return
	}

	var content string
	for _, n := range nfts {
		content += fmt.Sprintf("--- %s ---\n", n.Name)
		content += fmt.Sprintf("ID: %s\n", n.ID)
		content += fmt.Sprintf("Description: %s\n\n", n.Description)
	}
	m.viewport.SetContent(content)
}

func RunNFTUI() error {
	p := tea.NewProgram(NewNFTApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
