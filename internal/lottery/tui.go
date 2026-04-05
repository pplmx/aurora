package lottery

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/pplmx/aurora/internal/blockchain"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))
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
	participantsInput textinput.Model
	seedInput         textinput.Model
	countInput        textinput.Model
	viewport          viewport.Model
}

func NewLotteryApp() *model {
	chain := blockchain.InitBlockChain()

	pInput := textinput.New()
	pInput.Placeholder = "Alice\nBob\nCharlie\nDavid"
	pInput.Focus()

	sInput := textinput.New()
	sInput.Placeholder = "Enter random seed..."

	cInput := textinput.New()
	cInput.Placeholder = "3"
	cInput.SetValue("3")

	vp := viewport.New(60, 20)

	return &model{
		view:              "menu",
		chain:             chain,
		count:             "3",
		participantsInput: pInput,
		seedInput:         sInput,
		countInput:        cInput,
		viewport:          vp,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.view == "menu" {
				return m, tea.Quit
			}
			m.view = "menu"
			m.err = ""
			return m, nil

		case "enter":
			switch m.view {
			case "menu":
				// Menu selection via number handled below
			case "create":
				return m, m.handleCreate
			case "history":
				m.view = "menu"
			case "result":
				m.view = "menu"
			}
		case "1":
			if m.view == "menu" {
				m.view = "create"
				m.err = ""
			}
		case "2":
			if m.view == "menu" {
				m.loadHistory()
				m.view = "history"
			}
		case "3":
			if m.view == "menu" {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
	}

	return m, cmd
}

func (m *model) View() string {
	switch m.view {
	case "menu":
		return m.menuView()
	case "create":
		return m.createView()
	case "history":
		return m.historyView()
	case "result":
		return m.resultView()
	case "error":
		return m.errorView()
	}
	return ""
}

func (m *model) menuView() string {
	return titleStyle.Render("🌟 VRF 透明抽奖系统 🌟") + "\n\n" +
		borderStyle.Render("1. 创建抽奖") + "\n" +
		borderStyle.Render("2. 查看历史") + "\n" +
		borderStyle.Render("3. 退出") + "\n\n" +
		"按数字选择，回车确认"
}

func (m *model) createView() string {
	s := titleStyle.Render("创建新抽奖") + "\n\n" +
		"参与者（每行一个）:\n" +
		m.participantsInput.View() + "\n\n" +
		"随机种子:\n" +
		m.seedInput.View() + "\n\n" +
		"获奖人数:\n" +
		m.countInput.View() + "\n\n" +
		borderStyle.Render("[回车] 创建抽奖") + " | " +
		borderStyle.Render("[ESC] 返回")

	if m.err != "" {
		s += "\n" + errorStyle.Render(m.err)
	}

	return s
}

func (m *model) historyView() string {
	s := titleStyle.Render("抽奖历史") + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += borderStyle.Render("[ESC] 返回")
	return s
}

func (m *model) resultView() string {
	if m.result == nil {
		return "无结果"
	}

	s := successStyle.Render("🎉 抽奖完成！") + "\n\n"
	s += fmt.Sprintf("📋 抽奖ID: %s\n", m.result.ID)
	s += fmt.Sprintf("🔢 区块高度: #%d\n\n", m.result.BlockHeight)
	s += "🎊 中奖者:\n"

	for i, w := range m.result.Winners {
		s += fmt.Sprintf("   %d. %s (%s)\n", i+1, w, m.result.WinnerAddrs[i])
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
	s += fmt.Sprintf("🔐 VRF Output: %s...\n", vrfOut)
	s += fmt.Sprintf("📜 VRF Proof: %s...\n", vrfProof)

	s += "\n" + borderStyle.Render("[ESC] 返回主菜单")

	return s
}

func (m *model) errorView() string {
	return errorStyle.Render("错误: "+m.err) + "\n\n" + borderStyle.Render("[ESC] 返回")
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

	return nil
}

func (m *model) loadHistory() {
	records := m.chain.GetLotteryRecords()
	if len(records) == 0 {
		m.viewport.SetContent("暂无抽奖记录")
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
	p := tea.NewProgram(NewLotteryApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
