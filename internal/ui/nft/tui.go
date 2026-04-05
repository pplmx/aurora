package nft

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/pplmx/aurora/internal/i18n"
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
	menuIndex int
}

func NewNFTApp() *model {
	return &model{
		view:      "menu",
		menuIndex: 0,
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

func (m *model) View() tea.View {
	v := tea.NewView("")
	switch m.view {
	case "menu":
		v.SetContent(m.menuView())
	case "info":
		v.SetContent(m.infoView())
	}
	v.AltScreen = true
	return v
}

func (m *model) menuView() string {
	s := headerStyle.Render("🖼️ "+i18n.GetText("nft.tui.title")+" 🖼️") + "\n\n"
	items := []string{
		i18n.GetText("nft.tui.mint"),
		i18n.GetText("nft.tui.transfer"),
		i18n.GetText("nft.tui.query"),
		i18n.GetText("lottery.tui.exit"),
	}
	for i, item := range items {
		if i == m.menuIndex {
			s += menuSelectedStyle.Render("▶ " + item + "\n")
		} else {
			s += menuItemStyle.Render("  " + item + "\n")
		}
	}
	s += "\n" + helpStyle.Render(i18n.GetText("help.nav"))
	return s
}

func (m *model) infoView() string {
	s := headerStyle.Render("ℹ️ "+i18n.GetText("nft.tui.info")+" ℹ️") + "\n\n"
	s += i18n.GetText("nft.tui.cli_tip") + "\n\n"
	s += borderStyle.Render("  aurora nft mint -n <名称> -c <公钥>\n")
	s += "  aurora nft transfer --nft <id> --from <from> --to <to> -k <私钥>\n"
	s += "  aurora nft get --id <id>\n"
	s += "  aurora nft list --owner <公钥>\n\n"
	s += borderStyle.Render("[ESC] " + i18n.GetText("lottery.tui.back"))
	return s
}

func RunNFTUI() error {
	p := tea.NewProgram(NewNFTApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
