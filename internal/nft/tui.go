package nft

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
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
)

type model struct {
	view      string
	storage   *NFTStorage
	menuIndex int
}

func NewNFTApp(storage *NFTStorage) *model {
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
	case tea.KeyMsg:
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
			if m.view == "menu" && m.menuIndex < 2 {
				m.menuIndex++
			}

		case "enter":
			if m.view == "menu" {
				switch m.menuIndex {
				case 0, 1, 2:
					m.view = "info"
				case 3:
					return m, tea.Quit
				}
			} else {
				m.view = "menu"
			}
		}
	}

	return m, cmd
}

func (m *model) View() string {
	switch m.view {
	case "menu":
		return m.menuView()
	case "info":
		return m.infoView()
	}
	return ""
}

func (m *model) menuView() string {
	s := headerStyle.Render("🖼️ NFT 系统 🖼️") + "\n\n"
	items := []string{"铸造 NFT", "转让 NFT", "查询 NFT", "退出"}
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

func (m *model) infoView() string {
	s := headerStyle.Render("ℹ️ 信息 ℹ️") + "\n\n"
	s += "请使用 CLI 命令操作 NFT:\n\n"
	s += borderStyle.Render("  aurora nft mint -n <名称> -c <公钥>\n")
	s += "  aurora nft transfer --nft <id> --from <from> --to <to> -k <私钥>\n"
	s += "  aurora nft get --id <id>\n"
	s += "  aurora nft list --owner <公钥>\n\n"
	s += borderStyle.Render("[ESC] 返回")
	return s
}

func RunNFTUI(storage *NFTStorage) error {
	p := tea.NewProgram(NewNFTApp(storage), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
