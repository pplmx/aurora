package lottery

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/pplmx/aurora/internal/blockchain"

	tea "charm.land/bubbletea/v2"
)

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))
)

type model struct {
	view              string
	chain             *blockchain.BlockChain
	participants      string
	seed              string
	count             string
	result            *LotteryRecord
	history           string
	err               string
	successMsg        string
	participantsInput textinput.Model
	seedInput         textinput.Model
	countInput        textinput.Model
	viewport          viewport.Model
	menuIndex         int
	showHelp          bool
}

func NewLotteryApp() *model {
	chain := blockchain.InitBlockChain()

	pInput := textinput.New()
	pInput.Placeholder = "Alice\nBob\nCharlie\nDavid"
	pInput.Focus()
	pInput.Prompt = "  "

	sInput := textinput.New()
	sInput.Placeholder = "Enter random seed..."
	sInput.Prompt = "  "

	cInput := textinput.New()
	cInput.Placeholder = "3"
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

		case "up", "k":
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}

		case "down", "j":
			if m.view == "menu" && m.menuIndex < 2 {
				m.menuIndex++
			}

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
				case 1:
					m.loadHistory()
					m.view = "history"
				case 2:
					return m, tea.Quit
				}
			case "create":
				return m, m.handleCreate
			case "history", "result":
				m.view = "menu"
				m.successMsg = ""
			}
		case "1", "2", "3":
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
	menuItems := []string{"创建抽奖", "查看历史", "退出"}

	s := headerStyle.Render("🌟 VRF 透明抽奖系统 🌟") + "\n\n"

	for i, item := range menuItems {
		if i == m.menuIndex {
			s += menuSelectedStyle.Render("▶ " + item + "\n")
		} else {
			s += menuItemStyle.Render("  " + item + "\n")
		}
	}

	s += "\n" + helpStyle.Render("按 ↑↓ 选择, 回车确认, ? 查看帮助, q 退出")

	return s
}

func (m *model) createView() string {
	s := headerStyle.Render("📝 创建新抽奖") + "\n\n"
	s += infoStyle.Render("参与者（每行一个）:") + "\n"
	s += m.participantsInput.View() + "\n\n"
	s += infoStyle.Render("随机种子:") + "\n"
	s += m.seedInput.View() + "\n\n"
	s += infoStyle.Render("获奖人数:") + "\n"
	s += m.countInput.View() + "\n\n"

	if m.err != "" {
		s += errorStyle.Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += successStyle.Render("✓ "+m.successMsg) + "\n\n"
	}

	s += borderStyle.Render("[回车] 创建抽奖") + " | " + borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) historyView() string {
	s := headerStyle.Render("📜 抽奖历史") + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) resultView() string {
	if m.result == nil {
		return "无结果"
	}

	s := successStyle.Render("🎉 抽奖完成！") + "\n\n"
	s += infoStyle.Render("📋 抽奖ID: ") + m.result.ID + "\n"
	s += infoStyle.Render("🔢 区块高度: #") + fmt.Sprintf("%d", m.result.BlockHeight) + "\n\n"
	s += successStyle.Render("🎊 中奖者:") + "\n"

	for i, w := range m.result.Winners {
		s += fmt.Sprintf("   %d. %s (%s)\n", i+1, w, m.result.WinnerAddresses[i])
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
	s += infoStyle.Render("🔐 VRF Output: ") + vrfOut + "...\n"
	s += infoStyle.Render("📜 VRF Proof: ") + vrfProof + "...\n"

	s += "\n" + borderStyle.Render("[ESC] 返回主菜单")

	return s
}

func (m *model) helpView() string {
	s := headerStyle.Render("⌨ 键盘快捷键") + "\n\n"
	s += infoStyle.Render("导航:") + "\n"
	s += "  ↑/k  上移\n"
	s += "  ↓/j  下移\n"
	s += "  回车  确认\n"
	s += "  ESC  返回\n"
	s += "  q    退出\n\n"
	s += infoStyle.Render("菜单:") + "\n"
	s += "  1    创建抽奖\n"
	s += "  2    查看历史\n"
	s += "  3    退出\n\n"
	s += helpStyle.Render("按 ESC 或 ? 返回")

	return s
}

func (m *model) handleCreate() tea.Msg {
	participants := parseTextArea(m.participantsInput.Value())
	seed := m.seedInput.Value()
	count := 3
	fmt.Sscanf(m.countInput.Value(), "%d", &count)

	if len(participants) < count {
		m.err = "参与者人数必须多于获奖人数"
		return nil
	}

	if seed == "" {
		m.err = "种子不能为空"
		return nil
	}

	m.result = m.runLottery(participants, seed, count)
	m.view = "result"
	m.successMsg = "抽奖已创建并上链"

	return nil
}

func (m *model) loadHistory() {
	records := m.chain.GetLotteryRecords()
	if len(records) == 0 {
		m.viewport.SetContent("暂无抽奖记录\n\n" + helpStyle.Render("使用 'lottery create' 创建抽奖"))
	} else {
		var content string
		for i, data := range records {
			content += fmt.Sprintf("--- 抽奖 #%d ---\n%s\n\n", i+1, data)
		}
		m.viewport.SetContent(content)
	}
}

func (m *model) runLottery(participants []string, seed string, count int) *LotteryRecord {
	pk, sk, _ := GenerateKeyPair()
	output, proof, _ := VRFProve(sk, []byte(seed))

	winners := SelectWinners(output, participants, count)
	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = NameToAddress(w)
	}

	record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

	jsonData, _ := record.ToJSON()
	height, _ := m.chain.AddLotteryRecord(jsonData)
	record.BlockHeight = height

	_ = pk
	return record
}

func parseTextArea(text string) []string {
	var result []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
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
