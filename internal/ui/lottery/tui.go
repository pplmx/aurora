package lottery

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ui/components"
)

type model struct {
	view              string
	chain             *blockchain.BlockChain
	count             string
	result            *lottery.LotteryRecord
	err               string
	successMsg        string
	participantsInput textinput.Model
	seedInput         textinput.Model
	countInput        textinput.Model
	viewport          viewport.Model
	menuIndex         int
	inputFocus        int
	showHelp          bool
}

func NewLotteryApp() *model {
	chain := blockchain.InitBlockChain()

	pInput := textinput.New()
	pInput.Placeholder = i18n.GetText("lottery.tui.participants")
	pInput.Focus()
	pInput.Prompt = "  "

	sInput := textinput.New()
	sInput.Placeholder = i18n.GetText("lottery.tui.seed")
	sInput.Prompt = "  "

	cInput := textinput.New()
	cInput.Placeholder = i18n.GetText("lottery.tui.winners")
	cInput.SetValue("3")
	cInput.Prompt = "  "

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))

	return &model{
		view:              "menu",
		chain:             chain,
		count:             "3",
		participantsInput: pInput,
		seedInput:         sInput,
		countInput:        cInput,
		viewport:          vp,
		menuIndex:         0,
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
			if msg.String() == "esc" || msg.String() == "?" {
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == "menu" {
				return m, tea.Quit
			}
			m.view = "menu"
			m.err = ""
			m.successMsg = ""
			return m, nil

		case "?":
			m.showHelp = true
			return m, nil

		case "enter":
			switch m.view {
			case "menu":
				switch m.menuIndex {
				case 0:
					m.view = "create"
					m.err = ""
					m.successMsg = ""
					m.seedInput.SetValue("")
					m.countInput.SetValue("3")
					m.inputFocus = 0
					m.updateInputFocus()
				case 1:
					m.loadHistory()
					m.view = "history"
				case 2:
					return m, tea.Quit
				}
			case "create":
				m.handleCreate()
			case "history", "result":
				m.view = "menu"
				m.successMsg = ""
			}
		case "1", "2", "3":
			if m.view == "menu" {
				m.menuIndex = int(msg.String()[0] - '1')
			}
		case "up", "k":
			if m.view == "create" {
				if m.inputFocus > 0 {
					m.inputFocus--
					m.updateInputFocus()
				}
				return m, nil
			}
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down", "j":
			if m.view == "create" {
				if m.inputFocus < 2 {
					m.inputFocus++
					m.updateInputFocus()
				}
				return m, nil
			}
			if m.view == "menu" && m.menuIndex < 2 {
				m.menuIndex++
			}
		case "tab":
			if m.view == "create" {
				m.inputFocus = (m.inputFocus + 1) % 3
				m.updateInputFocus()
				return m, nil
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

	// Forward remaining keypresses into the focused input so the create form
	// is typable. Navigation keys (up/down/j/k/tab/esc/enter) were consumed
	// above and must not also reach the textinput.
	if m.view == "create" {
		var c1, c2, c3 tea.Cmd
		m.participantsInput, c1 = m.participantsInput.Update(msg)
		m.seedInput, c2 = m.seedInput.Update(msg)
		m.countInput, c3 = m.countInput.Update(msg)
		cmd = tea.Batch(cmd, c1, c2, c3)
	}

	return m, cmd
}

// updateInputFocus moves focus to the create form's currently-selected input.
func (m *model) updateInputFocus() {
	m.participantsInput.Blur()
	m.seedInput.Blur()
	m.countInput.Blur()

	switch m.inputFocus {
	case 0:
		m.participantsInput.Focus()
	case 1:
		m.seedInput.Focus()
	case 2:
		m.countInput.Focus()
	}
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	if m.showHelp {
		v.SetContent(m.helpView())
	} else {
		switch m.view {
		case "menu":
			v.SetContent(m.menuView())
		case "create":
			v.SetContent(m.createView())
		case "history":
			v.SetContent(m.historyView())
		case "result":
			v.SetContent(m.resultView())
		default:
			v.SetContent("")
		}
	}
	v.AltScreen = true
	return v
}

func (m *model) menuView() string {
	menuItems := []string{
		i18n.GetText("lottery.tui.create"),
		i18n.GetText("lottery.tui.history"),
		i18n.GetText("lottery.tui.exit"),
	}

	s := components.HeaderStyle().Render("🌟 "+i18n.GetText("lottery.tui.title")+" 🌟") + "\n\n"

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
	s := components.HeaderStyle().Render("📝 "+i18n.GetText("lottery.tui.create")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.tui.participants")+":") + "\n"
	s += m.participantsInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.tui.seed")+":") + "\n"
	s += m.seedInput.View() + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.tui.winners")+":") + "\n"
	s += m.countInput.View() + "\n\n"

	if m.err != "" {
		s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.create_btn")) + " | " + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) historyView() string {
	s := components.HeaderStyle().Render("📜 "+i18n.GetText("lottery.tui.history")) + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) resultView() string {
	if m.result == nil {
		return i18n.GetText("error.not_found")
	}

	s := components.SuccessStyle().Render("🎉 "+i18n.GetText("lottery.tui.completed")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.lottery_id")+": ") + m.result.ID + "\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.block_height")+": #") + fmt.Sprintf("%d", m.result.BlockHeight) + "\n\n"
	s += components.SuccessStyle().Render(i18n.GetText("lottery.winners")+":") + "\n"

	for i, w := range m.result.Winners {
		// Guard against mismatched slice lengths (imported data, old
		// schemas, partial writes). Without this, mismatched records
		// panic with index-out-of-range.
		addr := "(no address)"
		if i < len(m.result.WinnerAddresses) {
			addr = m.result.WinnerAddresses[i]
		}
		s += fmt.Sprintf("   %d. %s (%s)\n", i+1, w, addr)
	}

	s += "\n"
	vrfOut := m.result.VRFOutput
	vrfProof := m.result.VRFProof
	if len(vrfOut) > 32 {
		vrfOut = vrfOut[:32]
	}
	if len(vrfProof) > 32 {
		vrfProof = vrfProof[:32]
	}
	s += components.InfoStyle().Render(i18n.GetText("lottery.vrf_output")+": ") + vrfOut + "...\n"
	s += components.InfoStyle().Render(i18n.GetText("lottery.vrf_proof")+": ") + vrfProof + "...\n"

	s += "\n" + components.BorderStyle().Render(i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) helpView() string {
	s := components.HeaderStyle().Render("⌨ 键盘快捷键") + "\n\n"
	s += components.InfoStyle().Render("导航:") + "\n"
	s += "  ↑/k  上移\n"
	s += "  ↓/j  下移\n"
	s += "  回车  确认\n"
	s += "  ESC  返回\n"
	s += "  q    退出\n\n"
	s += components.InfoStyle().Render("菜单:") + "\n"
	s += "  1    创建抽奖\n"
	s += "  2    查看历史\n"
	s += "  3    退出\n\n"
	s += components.HelpTextStyle().Render("按 ESC 或 ? 返回")

	return s
}

func (m *model) handleCreate() tea.Msg {
	participants := parseTextArea(m.participantsInput.Value())
	seed := m.seedInput.Value()
	count := 3
	if c, err := strconv.Atoi(m.countInput.Value()); err == nil {
		count = c
	}

	if len(participants) == 0 {
		m.err = i18n.GetText("lottery.tui.participants_required")
		return nil
	}

	if count < 1 {
		m.err = i18n.GetText("lottery.tui.winners_positive")
		return nil
	}

	if len(participants) < count {
		m.err = i18n.GetText("lottery.tui.winners_exceed")
		return nil
	}

	if seed == "" {
		m.err = i18n.GetText("lottery.tui.seed_required")
		return nil
	}

	result := m.runLottery(participants, seed, count)
	if result == nil {
		m.err = i18n.GetText("lottery.tui.create_failed")
		return nil
	}
	m.result = result
	m.view = "result"
	m.successMsg = i18n.GetText("lottery.tui.created_onchain")

	return nil
}

func (m *model) loadHistory() {
	records := m.chain.GetLotteryRecords()
	if len(records) == 0 {
		m.viewport.SetContent("暂无抽奖记录\n\n" + components.HelpTextStyle().Render("使用 'lottery create' 创建抽奖"))
	} else {
		var content string
		for i, data := range records {
			content += fmt.Sprintf("--- 抽奖 #%d ---\n%s\n\n", i+1, data)
		}
		m.viewport.SetContent(content)
	}
}

func (m *model) runLottery(participants []string, seed string, count int) *lottery.LotteryRecord {
	pk, sk, _ := lottery.GenerateKeyPair()
	output, proof, _ := lottery.VRFProve(sk, []byte(seed))

	winners := lottery.SelectWinners(output, participants, count)
	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = lottery.NameToAddress(w)
	}

	record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, lottery.EncodePublicKey(pk), 0)

	jsonData, err := record.ToJSON()
	if err != nil {
		return nil
	}
	height, err := m.chain.AddLotteryRecord(jsonData)
	if err != nil {
		return nil
	}
	record.BlockHeight = height

	return record
}

// parseTextArea splits a participant list into individual names. The create
// form's single-line text input cannot hold newlines, so names are accepted
// comma-separated (matching the `lottery create -p "A,B,C"` CLI convention) —
// newlines are also honoured for any legacy multiline content.
func parseTextArea(text string) []string {
	var result []string
	for _, chunk := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			result = append(result, chunk)
		}
	}
	return result
}

func RunLotteryTUI() error {
	p := tea.NewProgram(NewLotteryApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
