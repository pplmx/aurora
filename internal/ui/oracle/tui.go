package oracle

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	domainoracle "github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ui/components"
)

type model struct {
	view        string
	repo        domainoracle.Repository
	menuIndex   int
	sources     []*domainoracle.DataSource
	listUseCase *oracleapp.ListSourcesUseCase
}

func NewOracleApp(repo domainoracle.Repository) *model {
	return &model{
		view:        "menu",
		repo:        repo,
		listUseCase: oracleapp.NewListSourcesUseCase(repo),
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
	s := components.HeaderStyle().Render("🔮 "+i18n.GetText("oracle.tui.title")+" 🔮") + "\n\n"
	items := []string{
		i18n.GetText("oracle.tui.source_mgmt"),
		i18n.GetText("oracle.tui.fetch_data"),
		i18n.GetText("oracle.tui.query_data"),
		i18n.GetText("lottery.tui.exit"),
	}
	for i, item := range items {
		if i == m.menuIndex {
			s += components.MenuSelectedStyle().Render("▶ " + item + "\n")
		} else {
			s += components.MenuItemStyle().Render("  " + item + "\n")
		}
	}
	s += "\n" + components.HelpTextStyle().Render(i18n.GetText("help.nav"))
	return s
}

func (m *model) sourcesView() string {
	s := components.HeaderStyle().Render("📡 "+i18n.GetText("oracle.tui.source_mgmt")) + "\n\n"

	if len(m.sources) == 0 {
		s += components.InfoStyle().Render(i18n.GetText("oracle.tui.no_sources") + "\n\n")
	} else {
		for i, ds := range m.sources {
			status := i18n.GetText("oracle.tui.enabled")
			if !ds.Enabled {
				status = i18n.GetText("oracle.tui.disabled")
			}
			s += fmt.Sprintf("%d. %s [%s] %s\n", i+1, ds.Name, ds.Type, status)
		}
	}

	s += "\n" + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))
	return s
}

func (m *model) fetchView() string {
	s := components.HeaderStyle().Render("📥 "+i18n.GetText("oracle.tui.fetch_data")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("oracle.tui.cli_tip") + "\n")
	s += "  aurora oracle fetch --source <id>\n\n"
	s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))
	return s
}

func (m *model) dataView() string {
	s := components.HeaderStyle().Render("📊 "+i18n.GetText("oracle.tui.query_data")) + "\n\n"
	s += components.InfoStyle().Render(i18n.GetText("oracle.tui.cli_tip") + "\n")
	s += "  aurora oracle data --source <id>\n"
	s += "  aurora oracle latest --source <id>\n\n"
	s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))
	return s
}

func (m *model) loadSources() {
	if list, err := m.listUseCase.Execute(&oracleapp.ListSourcesRequest{}); err == nil {
		m.sources = make([]*domainoracle.DataSource, len(list.Sources))
		for i, s := range list.Sources {
			m.sources[i] = &domainoracle.DataSource{
				ID:      s.ID,
				Name:    s.Name,
				URL:     s.URL,
				Type:    s.Type,
				Enabled: s.Enabled,
			}
		}
	}
}

func RunOracleTUI(repo domainoracle.Repository) error {
	p := tea.NewProgram(NewOracleApp(repo))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
