package voting

import (
	"encoding/base64"
	"fmt"
	"os"

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
	storage           Storage
	menuIndex         int
	candidateParty    textinput.Model
	candidateProgram  textinput.Model
	err               string
	successMsg        string
	resultMsg         string
	candidatesInput   textinput.Model
	voterNameInput    textinput.Model
	voterPKInput      textinput.Model
	voterPrivInput    textinput.Model
	candidateIDInput  textinput.Model
	sessionTitleInput textinput.Model
	sessionDescInput  textinput.Model
	sessionCandsInput textinput.Model
	viewport          viewport.Model
	showHelp          bool
	currentVoterPK    string
	currentVoterPriv  string
}

func NewVotingApp(storage Storage) *model {
	cInput := textinput.New()
	cInput.Placeholder = "Candidate name"
	cInput.Focus()
	cInput.Prompt = "  "

	cpInput := textinput.New()
	cpInput.Placeholder = "Party"
	cpInput.Prompt = "  "

	cprInput := textinput.New()
	cprInput.Placeholder = "Program"
	cprInput.Prompt = "  "

	vnInput := textinput.New()
	vnInput.Placeholder = "Voter name"
	vnInput.Focus()
	vnInput.Prompt = "  "

	vpkInput := textinput.New()
	vpkInput.Placeholder = "Public key (base64)"
	vpkInput.Prompt = "  "

	vprivInput := textinput.New()
	vprivInput.Placeholder = "Private key (base64)"
	vprivInput.Prompt = "  "

	cidInput := textinput.New()
	cidInput.Placeholder = "Candidate ID"
	cidInput.Prompt = "  "

	stInput := textinput.New()
	stInput.Placeholder = "Session title"
	stInput.Focus()
	stInput.Prompt = "  "

	sdInput := textinput.New()
	sdInput.Placeholder = "Session description"
	sdInput.Prompt = "  "

	scInput := textinput.New()
	scInput.Placeholder = "Candidate IDs (comma separated)"
	scInput.Prompt = "  "

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))

	m := &model{
		view:              "menu",
		storage:           storage,
		menuIndex:         0,
		candidatesInput:   cInput,
		candidateParty:    cpInput,
		candidateProgram:  cprInput,
		voterNameInput:    vnInput,
		voterPKInput:      vpkInput,
		voterPrivInput:    vprivInput,
		candidateIDInput:  cidInput,
		sessionTitleInput: stInput,
		sessionDescInput:  sdInput,
		sessionCandsInput: scInput,
		viewport:          vp,
	}
	return m
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
			if m.view == "menu" && m.menuIndex < 4 {
				m.menuIndex++
			}

		case "enter":
			switch m.view {
			case "menu":
				switch m.menuIndex {
				case 0:
					m.view = "registerCandidate"
					m.err = ""
					m.successMsg = ""
					m.candidatesInput.SetValue("")
					m.candidateParty.SetValue("")
					m.candidateProgram.SetValue("")
				case 1:
					m.view = "registerVoter"
					m.err = ""
					m.successMsg = ""
					m.voterNameInput.SetValue("")
				case 2:
					m.view = "castVote"
					m.err = ""
					m.successMsg = ""
					m.voterPKInput.SetValue(m.currentVoterPK)
					m.voterPrivInput.SetValue(m.currentVoterPriv)
					m.candidateIDInput.SetValue("")
				case 3:
					m.view = "viewResults"
					m.err = ""
					m.loadResults()
				case 4:
					return m, tea.Quit
				}
			case "registerCandidate":
				return m, m.handleRegisterCandidate
			case "registerVoter":
				return m, m.handleRegisterVoter
			case "castVote":
				return m, m.handleCastVote
			case "history", "result", "listCandidates", "listVoters":
				m.view = "menu"
				m.successMsg = ""
			}
		case "1", "2", "3", "4", "5":
			if m.view == "menu" {
				m.menuIndex = int(msg.String()[0] - '1')
			}
		case "esc":
			if m.view != "menu" && m.view != "listCandidates" && m.view != "listVoters" {
				m.view = "menu"
				m.err = ""
				m.successMsg = ""
			} else if m.view == "listCandidates" || m.view == "listVoters" {
				m.view = "menu"
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
		case "registerCandidate":
			v.SetContent(m.registerCandidateView())
		case "registerVoter":
			v.SetContent(m.registerVoterView())
		case "castVote":
			v.SetContent(m.castVoteView())
		case "viewResults":
			v.SetContent(m.viewResultsView())
		case "listCandidates":
			v.SetContent(m.listCandidatesView())
		case "listVoters":
			v.SetContent(m.listVotersView())
		case "result":
			v.SetContent(m.resultView())
		}
	}
	v.AltScreen = true
	return v
}

func (m *model) menuView() string {
	menuItems := []string{"注册候选人", "注册投票人", "投票", "查看结果", "退出"}

	s := headerStyle.Render("🗳️ Ed25519 透明投票系统 🗳️") + "\n\n"

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

func (m *model) registerCandidateView() string {
	s := headerStyle.Render("📝 注册候选人") + "\n\n"
	s += infoStyle.Render("姓名:") + "\n"
	s += m.candidatesInput.View() + "\n\n"
	s += infoStyle.Render("党派:") + "\n"
	s += m.candidateParty.View() + "\n\n"
	s += infoStyle.Render("竞选纲领:") + "\n"
	s += m.candidateProgram.View() + "\n\n"

	if m.err != "" {
		s += errorStyle.Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += successStyle.Render("✓ "+m.successMsg) + "\n\n"
	}

	s += borderStyle.Render("[回车] 注册") + " | " + borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) registerVoterView() string {
	s := headerStyle.Render("👤 注册投票人") + "\n\n"
	s += infoStyle.Render("姓名:") + "\n"
	s += m.voterNameInput.View() + "\n\n"

	if m.err != "" {
		s += errorStyle.Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += successStyle.Render("✓ "+m.successMsg) + "\n\n"
		s += infoStyle.Render("公钥 (用于投票):") + "\n"
		s += "  " + m.currentVoterPK + "\n\n"
		s += successStyle.Render("私钥 (请妥善保存!):") + "\n"
		s += "  " + m.currentVoterPriv + "\n\n"
	}

	s += borderStyle.Render("[回车] 注册") + " | " + borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) castVoteView() string {
	s := headerStyle.Render("🗳️ 投票") + "\n\n"
	s += infoStyle.Render("公钥:") + "\n"
	s += m.voterPKInput.View() + "\n\n"
	s += infoStyle.Render("私钥:") + "\n"
	s += m.voterPrivInput.View() + "\n\n"
	s += infoStyle.Render("候选人ID:") + "\n"
	s += m.candidateIDInput.View() + "\n\n"

	if m.err != "" {
		s += errorStyle.Render("⚠ "+m.err) + "\n\n"
	}

	if m.successMsg != "" {
		s += successStyle.Render("✓ "+m.successMsg) + "\n\n"
	}

	s += borderStyle.Render("[回车] 投票") + " | " + borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) viewResultsView() string {
	s := headerStyle.Render("📊 投票结果") + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) listCandidatesView() string {
	s := headerStyle.Render("📋 候选人列表") + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) listVotersView() string {
	s := headerStyle.Render("👥 投票人列表") + "\n\n"
	s += m.viewport.View() + "\n\n"
	s += borderStyle.Render("[ESC] 返回")

	return s
}

func (m *model) resultView() string {
	s := successStyle.Render("✅ 操作成功！") + "\n\n"
	s += m.resultMsg + "\n\n"
	s += borderStyle.Render("[ESC] 返回主菜单")

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
	s += "  1    注册候选人\n"
	s += "  2    注册投票人\n"
	s += "  3    投票\n"
	s += "  4    查看结果\n"
	s += "  5    退出\n\n"
	s += helpStyle.Render("按 ESC 或 ? 返回")

	return s
}

func (m *model) handleRegisterCandidate() tea.Msg {
	name := m.candidatesInput.Value()
	party := m.candidateParty.Value()
	program := m.candidateProgram.Value()

	if name == "" {
		m.err = "姓名不能为空"
		return nil
	}
	if party == "" {
		m.err = "党派不能为空"
		return nil
	}

	SetCandidateStorage(m.storage)
	cand, err := RegisterCandidate(name, party, program)
	if err != nil {
		m.err = fmt.Sprintf("注册失败: %v", err)
		return nil
	}

	m.successMsg = fmt.Sprintf("候选人已注册: %s", cand.Name)
	m.resultMsg = fmt.Sprintf("候选人ID: %s\n党派: %s", cand.ID, cand.Party)
	m.view = "result"
	return nil
}

func (m *model) handleRegisterVoter() tea.Msg {
	name := m.voterNameInput.Value()

	if name == "" {
		m.err = "姓名不能为空"
		return nil
	}

	SetVoterStorage(m.storage)
	pub, priv, err := RegisterVoter(name)
	if err != nil {
		m.err = fmt.Sprintf("注册失败: %v", err)
		return nil
	}

	m.currentVoterPK = base64.StdEncoding.EncodeToString(pub)
	m.currentVoterPriv = base64.StdEncoding.EncodeToString(priv)
	m.successMsg = "投票人注册成功！"
	m.resultMsg = fmt.Sprintf("公钥: %s\n私钥: %s", m.currentVoterPK, m.currentVoterPriv)
	m.view = "result"
	return nil
}

func (m *model) handleCastVote() tea.Msg {
	voterPK := m.voterPKInput.Value()
	voterPriv := m.voterPrivInput.Value()
	candidateID := m.candidateIDInput.Value()

	if voterPK == "" {
		m.err = "公钥不能为空"
		return nil
	}
	if voterPriv == "" {
		m.err = "私钥不能为空"
		return nil
	}
	if candidateID == "" {
		m.err = "候选人ID不能为空"
		return nil
	}

	pubBytes, err := base64.StdEncoding.DecodeString(voterPK)
	if err != nil {
		m.err = "无效的公钥格式"
		return nil
	}

	privBytes, err := base64.StdEncoding.DecodeString(voterPriv)
	if err != nil {
		m.err = "无效的私钥格式"
		return nil
	}

	SetCandidateStorage(m.storage)
	SetVoterStorage(m.storage)
	SetVoteStorage(m.storage)

	chain := blockchain.InitBlockChain()
	record, err := CastVote(pubBytes, candidateID, privBytes, chain)
	if err != nil {
		m.err = fmt.Sprintf("投票失败: %v", err)
		return nil
	}

	m.successMsg = "投票成功！"
	m.resultMsg = fmt.Sprintf("投票ID: %s\n区块高度: %d", record.ID, record.BlockHeight)
	m.view = "result"
	return nil
}

func (m *model) loadResults() {
	SetCandidateStorage(m.storage)
	candidates, err := ListCandidates()
	if err != nil {
		m.viewport.SetContent(errorStyle.Render("加载失败: " + err.Error()))
		return
	}

	if len(candidates) == 0 {
		m.viewport.SetContent("暂无候选人\n\n" + helpStyle.Render("使用 '注册候选人' 创建候选人"))
		return
	}

	var content string
	for _, c := range candidates {
		content += fmt.Sprintf("📌 %s [%s]\n", c.Name, c.Party)
		content += fmt.Sprintf("   ID: %s\n", c.ID)
		content += fmt.Sprintf("   票数: %d\n\n", c.VoteCount)
	}
	m.viewport.SetContent(content)
}

func RunVotingTUI(storage Storage) error {
	p := tea.NewProgram(NewVotingApp(storage))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
