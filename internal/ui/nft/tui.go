package nft

import (
	"crypto/ed25519"
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
	view       string
	menuIndex  int
	inputFocus int
	showHelp   bool

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
	nftService := nft.NewServiceWithoutTx(repo)

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
			// name/description/id/owner inputs (the help screen scopes q to
			// the menu); read-only views (result, list) keep back-to-menu.
			if !m.isFormView() {
				m.view = "menu"
				m.err = ""
				m.successMsg = ""
				return m, nil
			}

		case "?":
			// scoped to non-form views: in the mint form "?" is a typable
			// character (descriptions with "?"), mirroring the q
			// fall-through convention (TASK-161, ISS-154; ISS-164).
			if !m.isFormView() {
				m.showHelp = true
				return m, nil
			}

		case "up":
			if m.view == "menu" {
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			} else if m.isFormView() && m.inputFocus > 0 {
				m.inputFocus--
				m.updateInputFocus()
				return m, nil
			}
		case "k":
			// A typable letter in the mint form (falls through below);
			// arrow keys and Tab are the form-navigation keys, the menu
			// still navigates on the bare letter (ISS-164).
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down":
			if m.view == "menu" {
				if m.menuIndex < 4 {
					m.menuIndex++
				}
			} else if m.isFormView() && m.inputFocus < m.formInputCount()-1 {
				m.inputFocus++
				m.updateInputFocus()
				return m, nil
			}
		case "j":
			if m.view == "menu" && m.menuIndex < 4 {
				m.menuIndex++
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
				switch m.menuIndex {
				case 0:
					m.view = "mint"
					m.err = ""
					m.successMsg = ""
					m.nameInput.SetValue("")
					m.descInput.SetValue("")
					m.pubkeyInput.SetValue("")
					m.inputFocus = 0
					m.updateInputFocus()
					return m, nil
				case 1:
					m.view = "transfer"
					m.err = ""
					m.successMsg = ""
					m.nftIDInput.SetValue("")
					m.fromKeyInput.SetValue("")
					m.toAddrInput.SetValue("")
					m.inputFocus = 0
					m.updateInputFocus()
					return m, nil
				case 2:
					m.view = "query"
					m.err = ""
					m.successMsg = ""
					m.queryIDInput.SetValue("")
					m.inputFocus = 0
					m.updateInputFocus()
					return m, nil
				case 3:
					m.view = "listOwner"
					m.err = ""
					m.successMsg = ""
					m.ownerInput.SetValue("")
					m.inputFocus = 0
					m.updateInputFocus()
					return m, nil
				case 4:
					return m, tea.Quit
				}
			case "mint":
				return m, m.handleMint
			case "transfer":
				return m, m.handleTransfer
			case "query":
				return m, m.handleQuery
			case "listOwner":
				// Type an owner public key here, Enter loads the list (the
				// previously-dead case "list" kept the same call; ISS-169).
				m.loadNFTsByOwner()
				m.view = "list"
			case "list":
				m.loadNFTsByOwner()
				m.view = "list"
			case "result":
				m.view = "menu"
				m.successMsg = ""
				m.nft = nil
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

	// Forward remaining keypresses into the focused input so the form views
	// are typable (round-97: keystrokes never reached the textinput models).
	cmd = tea.Batch(cmd, m.forwardToActiveInput(msg))

	// The list view is a viewport; let it handle scrolling keys
	// (up/down/j/k/pgup/pgdn/space/b/f/u/d) so long NFT lists are reachable
	// instead of clipped at the 15-row viewport (TASK-127, ISS-119).
	if m.view == "list" {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

// isFormView reports whether the current view edits NFT fields.
func (m *model) isFormView() bool {
	switch m.view {
	case "mint", "transfer", "query", "listOwner":
		return true
	}
	return false
}

// formInputCount returns how many text inputs the current view has.
func (m *model) formInputCount() int {
	switch m.view {
	case "mint", "transfer":
		return 3
	case "query", "listOwner":
		return 1
	}
	return 0
}

// updateInputFocus focuses the current view's input at m.inputFocus.
func (m *model) updateInputFocus() {
	switch m.view {
	case "mint":
		m.nameInput.Blur()
		m.descInput.Blur()
		m.pubkeyInput.Blur()
		switch m.inputFocus {
		case 0:
			m.nameInput.Focus()
		case 1:
			m.descInput.Focus()
		case 2:
			m.pubkeyInput.Focus()
		}
	case "transfer":
		m.nftIDInput.Blur()
		m.fromKeyInput.Blur()
		m.toAddrInput.Blur()
		switch m.inputFocus {
		case 0:
			m.nftIDInput.Focus()
		case 1:
			m.fromKeyInput.Focus()
		case 2:
			m.toAddrInput.Focus()
		}
	case "query":
		m.queryIDInput.Blur()
		m.queryIDInput.Focus()
	case "listOwner":
		m.ownerInput.Blur()
		m.ownerInput.Focus()
	}
}

// forwardToActiveInput pipes a message into the active view's text inputs.
func (m *model) forwardToActiveInput(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch m.view {
	case "mint":
		var c1, c2, c3 tea.Cmd
		m.nameInput, c1 = m.nameInput.Update(msg)
		m.descInput, c2 = m.descInput.Update(msg)
		m.pubkeyInput, c3 = m.pubkeyInput.Update(msg)
		cmds = append(cmds, c1, c2, c3)
	case "transfer":
		var c1, c2, c3 tea.Cmd
		m.nftIDInput, c1 = m.nftIDInput.Update(msg)
		m.fromKeyInput, c2 = m.fromKeyInput.Update(msg)
		m.toAddrInput, c3 = m.toAddrInput.Update(msg)
		cmds = append(cmds, c1, c2, c3)
	case "query":
		var c1 tea.Cmd
		m.queryIDInput, c1 = m.queryIDInput.Update(msg)
		cmds = append(cmds, c1)
	case "listOwner":
		var c1 tea.Cmd
		m.ownerInput, c1 = m.ownerInput.Update(msg)
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
		case "mint":
			v.SetContent(m.mintView())
		case "transfer":
			v.SetContent(m.transferView())
		case "query":
			v.SetContent(m.queryView())
		case "listOwner":
			v.SetContent(m.listOwnerView())
		case "result":
			v.SetContent(m.resultView())
		case "list":
			v.SetContent(m.listView())
		default:
			v.SetContent(m.menuView())
		}
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
		i18n.GetText("nft.tui.list_owner"),
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
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.nft_id")+": ") + m.nft.ID + "\n"
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

// listOwnerView prompts for an owner public key; Enter on this view loads the
// owner's NFTs into the scrollable list view (ISS-169 wired the previously
// dead list machinery to this surface).
func (m *model) listOwnerView() string {
	s := components.HeaderStyle().Render("📜 "+i18n.GetText("nft.tui.list_owner")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("nft.tui.owner")+": ") + "\n"
	s += m.ownerInput.View() + "\n\n"
	s += components.HelpTextStyle().Render(i18n.GetText("nft.tui.enter_owner_hint")) + "\n"
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
	// The owner key must be a full 32-byte Ed25519 public key; a valid-base64
	// but wrong-length value ("AAAA" -> 3 bytes) would otherwise mint an NFT
	// that no real 32-byte key can ever match — permanently owner-locked with
	// no error (mirrors the transfer fromKey length check + domain key-length
	// enforcement, TASK-163, ISS-156).
	if len(pubkey) != ed25519.PublicKeySize {
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

	// fromKey is the full 64-byte Ed25519 private key (seed|public). The
	// service derives the from-pubkey as fromKey[32:], so a 32-byte seed-only
	// key would yield an empty owner and always fail the transfer.
	if len(fromKey) != ed25519.PrivateKeySize {
		m.err = i18n.GetText("error.invalid_privkey")
		return nil
	}

	fromPubKey := fromKey[32:]
	_, err = m.nftService.Transfer(nftID, fromPubKey, toAddr, fromKey, m.chain)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	// Refresh m.nft to the transferred NFT so the result view shows its real
	// post-transfer state (new owner) instead of "⚠ Not found" when m.nft is
	// nil (fresh session) or an unrelated previously-queried NFT. The chain
	// write has already committed, so the stale/nil render would misleadingly
	// suggest the transfer failed (TASK-163, ISS-156).
	if nft, nerr := m.nftService.GetNFTByID(nftID); nerr == nil && nft != nil {
		m.nft = nft
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

	// The TUI shows the full sandbox collection (no paging caps — only the
	// REST list is bounded, TASK-101, ISS-093).
	nfts, err := m.nftService.GetNFTsByOwner(owner, 0, 0)
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
		content += fmt.Sprintf("%s: %s\n", i18n.GetText("nft.tui.nft_id"), n.ID)
		content += fmt.Sprintf("%s: %s\n\n", i18n.GetText("nft.tui.description"), n.Description)
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
