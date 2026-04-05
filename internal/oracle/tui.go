package oracle

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"

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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))
)

type model struct {
	view      string
	storage   Storage
	menuIndex int
	sources   []*DataSource
}

func NewOracleApp(storage Storage) *model {
	return &model{
		view:    "menu",
		storage: storage,
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
					m.loadSources()
					m.view = "sources"
				case 1:
					m.view = "fetch"
				case 2:
					m.view = "data"
				case 3:
					return m, tea.Quit
				}
			case "sources", "fetch", "data":
				m.view = "menu"
			}

		case "esc":
			m.view = "menu"
		}
	}

	return m, cmd
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	switch m.view {
	case "menu":
		v.SetContent(m.menuView())
	case "sources":
		v.SetContent(m.sourcesView())
	case "fetch":
		v.SetContent(m.fetchView())
	case "data":
		v.SetContent(m.dataView())
	}
	v.AltScreen = true
	return v
}

func (m *model) menuView() string {
	s := headerStyle.Render("🔮 Oracle 预言机系统 🔮") + "\n\n"
	items := []string{"数据源管理", "获取数据", "数据查询", "退出"}
	for i, item := range items {
		if i == m.menuIndex {
			s += menuSelectedStyle.Render("▶ " + item + "\n")
		} else {
			s += menuItemStyle.Render("  " + item + "\n")
		}
	}
	s += "\n" + helpStyle.Render("按 ↑↓ 选择, 回车确认, q 退出")
	return s
}

func (m *model) sourcesView() string {
	s := headerStyle.Render("📡 数据源管理") + "\n\n"

	if len(m.sources) == 0 {
		s += infoStyle.Render("暂无数据源\n\n")
	} else {
		for i, ds := range m.sources {
			status := "✓"
			if !ds.Enabled {
				status = "✗"
			}
			s += fmt.Sprintf("%d. %s [%s] %s\n", i+1, ds.Name, ds.Type, status)
		}
	}

	s += "\n" + borderStyle.Render("[ESC] 返回")
	return s
}

func (m *model) fetchView() string {
	s := headerStyle.Render("📥 获取数据") + "\n\n"
	s += infoStyle.Render("使用 CLI 命令获取数据:\n")
	s += "  aurora oracle fetch --source <id>\n\n"
	s += borderStyle.Render("[ESC] 返回")
	return s
}

func (m *model) dataView() string {
	s := headerStyle.Render("📊 数据查询") + "\n\n"
	s += infoStyle.Render("使用 CLI 命令查询数据:\n")
	s += "  aurora oracle data --source <id>\n"
	s += "  aurora oracle latest --source <id>\n\n"
	s += borderStyle.Render("[ESC] 返回")
	return s
}

func (m *model) loadSources() {
	if list, err := ListDataSources(); err == nil {
		m.sources = list
	}
}

func RunOracleTUI(storage Storage) error {
	p := tea.NewProgram(NewOracleApp(storage))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
