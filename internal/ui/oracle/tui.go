package oracle

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/textinput"
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

	sourceInputName  textinput.Model
	sourceInputURL   textinput.Model
	sourceInputType  textinput.Model
	fetchInputSource textinput.Model
	queryInputSource textinput.Model
	queryInputLimit  textinput.Model
	inputFocus       int
	selectedSourceID string
	confirmAction    string
	errMsg           string
	successMsg       string
	fetchResult      *oracleapp.FetchDataResponse
	queryResult      *oracleapp.GetDataResponse
}

func NewOracleApp(repo domainoracle.Repository) *model {
	nameInput := textinput.New()
	nameInput.Placeholder = i18n.GetText("oracle.tui.enter_name")
	nameInput.Focus()

	urlInput := textinput.New()
	urlInput.Placeholder = i18n.GetText("oracle.tui.enter_url")

	typeInput := textinput.New()
	typeInput.Placeholder = i18n.GetText("oracle.tui.enter_type")

	fetchInput := textinput.New()
	fetchInput.Placeholder = i18n.GetText("oracle.tui.enter_source_id")

	queryInputSource := textinput.New()
	queryInputSource.Placeholder = i18n.GetText("oracle.tui.enter_source_id")

	queryInputLimit := textinput.New()
	queryInputLimit.Placeholder = i18n.GetText("oracle.tui.enter_limit")

	return &model{
		view:             "menu",
		repo:             repo,
		listUseCase:      oracleapp.NewListSourcesUseCase(repo),
		sourceInputName:  nameInput,
		sourceInputURL:   urlInput,
		sourceInputType:  typeInput,
		fetchInputSource: fetchInput,
		queryInputSource: queryInputSource,
		queryInputLimit:  queryInputLimit,
		inputFocus:       0,
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
			m.errMsg = ""
			m.successMsg = ""

		case "up", "k":
			switch m.view {
			case "menu":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "sources":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "addSource":
				if m.inputFocus > 0 {
					m.inputFocus--
					m.updateInputFocus()
				}
			case "fetch":
				if m.inputFocus > 0 {
					m.inputFocus--
					m.updateFetchInputFocus()
				}
			case "query":
				if m.inputFocus > 0 {
					m.inputFocus--
					m.updateQueryInputFocus()
				}
			case "confirmToggle", "confirmDelete":
				m.menuIndex = 0
			}

		case "down", "j":
			switch m.view {
			case "menu":
				if m.menuIndex < 3 {
					m.menuIndex++
				}
			case "sources":
				if m.menuIndex < len(m.sources)+1 {
					m.menuIndex++
				}
			case "addSource":
				if m.inputFocus < 2 {
					m.inputFocus++
					m.updateInputFocus()
				}
			case "fetch":
				m.inputFocus = 0
				m.updateFetchInputFocus()
			case "query":
				if m.inputFocus < 1 {
					m.inputFocus++
					m.updateQueryInputFocus()
				}
			case "confirmToggle", "confirmDelete":
				m.menuIndex = 1
			}

		case "tab":
			switch m.view {
			case "addSource":
				m.inputFocus = (m.inputFocus + 1) % 3
				m.updateInputFocus()
			case "fetch":
				m.inputFocus = (m.inputFocus + 1) % 2
				m.updateFetchInputFocus()
			case "query":
				m.inputFocus = (m.inputFocus + 1) % 2
				m.updateQueryInputFocus()
			}

		case "enter":
			switch m.view {
			case "menu":
				switch m.menuIndex {
				case 0:
					m.loadSources()
					m.menuIndex = 0
					m.view = "sources"
				case 1:
					m.initFetch()
					m.view = "fetch"
				case 2:
					m.initQuery()
					m.view = "query"
				case 3:
					return m, tea.Quit
				}
			case "sources":
				if m.menuIndex == 0 {
					m.initAddSource()
					m.view = "addSource"
				} else if m.menuIndex <= len(m.sources) {
					idx := m.menuIndex - 1
					if idx < len(m.sources) {
						m.selectedSourceID = m.sources[idx].ID
						m.menuIndex = 0
						m.view = "sourceDetail"
					}
				}
			case "addSource":
				m.handleAddSource()
			case "sourceDetail":
				m.view = "sources"
			case "confirmToggle":
				if m.menuIndex == 0 {
					m.handleToggleSource()
				} else {
					m.view = "sources"
				}
				m.menuIndex = 0
			case "confirmDelete":
				if m.menuIndex == 0 {
					m.handleDeleteSource()
				} else {
					m.view = "sources"
				}
				m.menuIndex = 0
			case "fetch":
				m.handleFetch()
			case "query":
				m.handleQuery()
			case "fetchResult", "queryResult":
				m.view = "menu"
			case "data":
				m.view = "menu"
			}

		case "d":
			if m.view == "sourceDetail" && m.selectedSourceID != "" {
				m.confirmAction = "delete"
				m.menuIndex = 1
				m.view = "confirmDelete"
			}
		case "t":
			if m.view == "sourceDetail" && m.selectedSourceID != "" {
				m.confirmAction = "toggle"
				m.menuIndex = 1
				m.view = "confirmToggle"
			}

		case "a":
			if m.view == "sources" {
				m.initAddSource()
				m.view = "addSource"
			}

		case "esc":
			switch m.view {
			case "addSource":
				m.view = "sources"
				m.errMsg = ""
			case "sourceDetail":
				m.view = "sources"
			case "confirmToggle", "confirmDelete":
				m.view = "sources"
			default:
				m.view = "menu"
			}
			m.errMsg = ""
			m.successMsg = ""
		}

	case tea.WindowSizeMsg:
		// Handle window resize if needed
	}

	// Update textinput models
	if m.view == "addSource" {
		var cmdName, cmdURL, cmdType tea.Cmd
		m.sourceInputName, cmdName = m.sourceInputName.Update(msg)
		m.sourceInputURL, cmdURL = m.sourceInputURL.Update(msg)
		m.sourceInputType, cmdType = m.sourceInputType.Update(msg)
		cmd = tea.Batch(cmdName, cmdURL, cmdType)
	}

	if m.view == "fetch" {
		var cmdFetch tea.Cmd
		m.fetchInputSource, cmdFetch = m.fetchInputSource.Update(msg)
		cmd = cmdFetch
	}

	if m.view == "query" {
		var cmdSource, cmdLimit tea.Cmd
		m.queryInputSource, cmdSource = m.queryInputSource.Update(msg)
		m.queryInputLimit, cmdLimit = m.queryInputLimit.Update(msg)
		cmd = tea.Batch(cmdSource, cmdLimit)
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
	case "addSource":
		v.SetContent(m.addSourceView())
	case "sourceDetail":
		v.SetContent(m.sourceDetailView())
	case "confirmToggle":
		v.SetContent(m.confirmToggleView())
	case "confirmDelete":
		v.SetContent(m.confirmDeleteView())
	case "fetch":
		v.SetContent(m.fetchView())
	case "fetchResult":
		v.SetContent(m.fetchResultView())
	case "query":
		v.SetContent(m.queryView())
	case "queryResult":
		v.SetContent(m.queryResultView())
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

	menuItems := []string{i18n.GetText("oracle.tui.add_source")}
	for _, ds := range m.sources {
		menuItems = append(menuItems, ds.Name)
	}

	for i, item := range menuItems {
		prefix := "  "
		if i == m.menuIndex {
			prefix = "▶ "
			s += components.MenuSelectedStyle().Render(prefix+item) + "\n"
		} else {
			s += components.MenuItemStyle().Render(prefix+item) + "\n"
		}
	}

	s += "\n"
	s += components.BorderStyle().Render("[A] " + i18n.GetText("oracle.tui.add_source"))
	s += " | [Enter] " + i18n.GetText("oracle.tui.edit_source")
	s += " | [ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}
	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

	return s
}

func (m *model) addSourceView() string {
	s := components.HeaderStyle().Render("➕ "+i18n.GetText("oracle.tui.add_source")) + "\n\n"

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_name") + ":\n")
	if m.inputFocus == 0 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputName.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputName.View() + "\n\n"
	}

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_url") + ":\n")
	if m.inputFocus == 1 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputURL.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputURL.View() + "\n\n"
	}

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_type") + ":\n")
	if m.inputFocus == 2 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputType.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputType.View() + "\n\n"
	}

	s += "\n" + components.BorderStyle().Render("[TAB] "+i18n.GetText("lottery.tui.next"))
	s += " | [Enter] " + i18n.GetText("lottery.tui.confirm")
	s += " | [ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}

	return s
}

func (m *model) sourceDetailView() string {
	var source *domainoracle.DataSource
	for _, s := range m.sources {
		if s.ID == m.selectedSourceID {
			source = s
			break
		}
	}

	s := components.HeaderStyle().Render("📝 "+i18n.GetText("oracle.tui.edit_source")) + "\n\n"

	if source != nil {
		s += components.KeyValue(i18n.GetText("oracle.tui.source_name"), source.Name) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_url"), source.URL) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_type"), source.Type) + "\n"

		status := i18n.GetText("oracle.tui.enabled")
		if !source.Enabled {
			status = i18n.GetText("oracle.tui.disabled")
		}
		s += components.KeyValue("Status", status) + "\n"
	}

	s += "\n"
	s += components.BorderStyle().Render("[T] Toggle On/Off")
	s += " | [D] " + i18n.GetText("oracle.tui.delete_source")
	s += " | [ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}
	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

	return s
}

func (m *model) confirmToggleView() string {
	var source *domainoracle.DataSource
	for _, s := range m.sources {
		if s.ID == m.selectedSourceID {
			source = s
			break
		}
	}

	s := components.HeaderStyle().Render("⚠️ "+i18n.GetText("oracle.tui.confirm_toggle")) + "\n\n"

	if source != nil {
		if source.Enabled {
			s += components.WarningStyle().Render(i18n.GetText("oracle.tui.sure_disable")) + "\n\n"
		} else {
			s += components.WarningStyle().Render(i18n.GetText("oracle.tui.sure_enable")) + "\n\n"
		}
		s += components.BodyStyle().Render(source.Name) + "\n"
	}

	s += "\n" + components.BorderStyle().Render(i18n.GetText("oracle.tui.yes_no")) + "\n"
	s += "\n[ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	return s
}

func (m *model) confirmDeleteView() string {
	var source *domainoracle.DataSource
	for _, s := range m.sources {
		if s.ID == m.selectedSourceID {
			source = s
			break
		}
	}

	s := components.HeaderStyle().Render("⚠️ "+i18n.GetText("oracle.tui.confirm_delete")) + "\n\n"

	if source != nil {
		s += components.WarningStyle().Render(i18n.GetText("oracle.tui.sure_delete")) + "\n\n"
		s += components.BodyStyle().Render(source.Name) + "\n"
	}

	s += "\n" + components.BorderStyle().Render(i18n.GetText("oracle.tui.yes_no")) + "\n"
	s += "\n[ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	return s
}

func (m *model) fetchView() string {
	s := components.HeaderStyle().Render("📥 "+i18n.GetText("oracle.tui.fetch_data")) + "\n\n"

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_id") + ":\n")
	if m.inputFocus == 0 {
		s += components.MenuSelectedStyle().Render("> "+m.fetchInputSource.View()) + "\n\n"
	} else {
		s += "  " + m.fetchInputSource.View() + "\n\n"
	}

	s += "\n" + components.BorderStyle().Render("[Enter] "+i18n.GetText("lottery.tui.confirm"))
	s += " | [ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}

	return s
}

func (m *model) queryView() string {
	s := components.HeaderStyle().Render("📊 "+i18n.GetText("oracle.tui.query_data")) + "\n\n"

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_id") + ":\n")
	if m.inputFocus == 0 {
		s += components.MenuSelectedStyle().Render("> "+m.queryInputSource.View()) + "\n\n"
	} else {
		s += "  " + m.queryInputSource.View() + "\n\n"
	}

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.limit") + ":\n")
	if m.inputFocus == 1 {
		s += components.MenuSelectedStyle().Render("> "+m.queryInputLimit.View()) + "\n\n"
	} else {
		s += "  " + m.queryInputLimit.View() + "\n\n"
	}

	s += "\n" + components.BorderStyle().Render("[TAB] "+i18n.GetText("lottery.tui.next"))
	s += " | [Enter] " + i18n.GetText("lottery.tui.confirm")
	s += " | [ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}

	return s
}

func (m *model) fetchResultView() string {
	s := components.HeaderStyle().Render("📥 "+i18n.GetText("oracle.tui.fetch_result")) + "\n\n"

	if m.fetchResult != nil {
		s += components.KeyValue("ID", m.fetchResult.ID) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_id"), m.fetchResult.SourceID) + "\n"
		s += components.KeyValue("Value", m.fetchResult.Value) + "\n"
		s += components.KeyValue("Timestamp", fmt.Sprintf("%d", m.fetchResult.Timestamp)) + "\n"
		s += components.KeyValue("BlockHeight", fmt.Sprintf("%d", m.fetchResult.BlockHeight)) + "\n"
	}

	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

	s += "\n" + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) queryResultView() string {
	s := components.HeaderStyle().Render("📊 "+i18n.GetText("oracle.tui.query_result")) + "\n\n"

	if m.queryResult != nil && len(m.queryResult.Data) > 0 {
		for i, d := range m.queryResult.Data {
			s += components.CaptionStyle().Render(fmt.Sprintf("--- #%d ---", i+1)) + "\n"
			s += components.KeyValue("ID", d.ID) + "\n"
			s += components.KeyValue(i18n.GetText("oracle.tui.source_id"), d.SourceID) + "\n"
			s += components.KeyValue("Value", d.Value) + "\n"
			s += components.KeyValue("Timestamp", fmt.Sprintf("%d", d.Timestamp)) + "\n"
			s += components.KeyValue("BlockHeight", fmt.Sprintf("%d", d.BlockHeight)) + "\n\n"
		}
	} else {
		s += components.InfoStyle().Render(i18n.GetText("oracle.tui.no_data")) + "\n"
	}

	s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}

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

func (m *model) initAddSource() {
	m.sourceInputName = textinput.New()
	m.sourceInputName.Placeholder = i18n.GetText("oracle.tui.enter_name")
	m.sourceInputName.Focus()

	m.sourceInputURL = textinput.New()
	m.sourceInputURL.Placeholder = i18n.GetText("oracle.tui.enter_url")

	m.sourceInputType = textinput.New()
	m.sourceInputType.Placeholder = i18n.GetText("oracle.tui.enter_type")

	m.inputFocus = 0
	m.errMsg = ""
	m.successMsg = ""
}

func (m *model) updateInputFocus() {
	m.sourceInputName.Blur()
	m.sourceInputURL.Blur()
	m.sourceInputType.Blur()

	switch m.inputFocus {
	case 0:
		m.sourceInputName.Focus()
	case 1:
		m.sourceInputURL.Focus()
	case 2:
		m.sourceInputType.Focus()
	}
}

func (m *model) handleAddSource() {
	name := m.sourceInputName.Value()
	url := m.sourceInputURL.Value()
	sourceType := m.sourceInputType.Value()

	if name == "" || url == "" {
		m.errMsg = i18n.GetText("error.invalid_input")
		return
	}

	if sourceType == "" {
		sourceType = "custom"
	}

	addUseCase := oracleapp.NewAddSourceUseCase(m.repo)
	_, err := addUseCase.Execute(&oracleapp.AddSourceRequest{
		Name: name,
		URL:  url,
		Type: sourceType,
	})

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.successMsg = i18n.GetText("oracle.tui.add_success")
		m.loadSources()
		m.view = "sources"
	}
}

func (m *model) handleToggleSource() {
	if m.selectedSourceID == "" {
		return
	}

	var source *domainoracle.DataSource
	for _, s := range m.sources {
		if s.ID == m.selectedSourceID {
			source = s
			break
		}
	}

	if source == nil {
		return
	}

	var err error
	if source.Enabled {
		disableUseCase := oracleapp.NewDisableSourceUseCase(m.repo)
		err = disableUseCase.Execute(m.selectedSourceID)
	} else {
		enableUseCase := oracleapp.NewEnableSourceUseCase(m.repo)
		err = enableUseCase.Execute(m.selectedSourceID)
	}

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.successMsg = i18n.GetText("oracle.tui.toggle_success")
		m.loadSources()
		m.view = "sources"
	}
}

func (m *model) handleDeleteSource() {
	if m.selectedSourceID == "" {
		return
	}

	deleteUseCase := oracleapp.NewDeleteSourceUseCase(m.repo)
	err := deleteUseCase.Execute(m.selectedSourceID)

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.successMsg = i18n.GetText("oracle.tui.delete_success")
		m.loadSources()
		m.selectedSourceID = ""
		m.view = "sources"
	}
}

func (m *model) initFetch() {
	m.fetchInputSource = textinput.New()
	m.fetchInputSource.Placeholder = i18n.GetText("oracle.tui.enter_source_id")
	m.fetchInputSource.Focus()
	m.inputFocus = 0
	m.errMsg = ""
	m.successMsg = ""
	m.fetchResult = nil
}

func (m *model) initQuery() {
	m.queryInputSource = textinput.New()
	m.queryInputSource.Placeholder = i18n.GetText("oracle.tui.enter_source_id")
	m.queryInputSource.Focus()

	m.queryInputLimit = textinput.New()
	m.queryInputLimit.Placeholder = i18n.GetText("oracle.tui.enter_limit")

	m.inputFocus = 0
	m.errMsg = ""
	m.successMsg = ""
	m.queryResult = nil
}

func (m *model) updateFetchInputFocus() {
	m.fetchInputSource.Blur()
	switch m.inputFocus {
	case 0:
		m.fetchInputSource.Focus()
	}
}

func (m *model) updateQueryInputFocus() {
	m.queryInputSource.Blur()
	m.queryInputLimit.Blur()

	switch m.inputFocus {
	case 0:
		m.queryInputSource.Focus()
	case 1:
		m.queryInputLimit.Focus()
	}
}

func (m *model) handleFetch() {
	sourceID := m.fetchInputSource.Value()
	if sourceID == "" {
		m.errMsg = i18n.GetText("error.invalid_input")
		return
	}

	fetchUseCase := oracleapp.NewFetchDataUseCase(m.repo)
	result, err := fetchUseCase.Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.fetchResult = result
		m.successMsg = i18n.GetText("oracle.tui.fetch_success")
		m.view = "fetchResult"
	}
}

func (m *model) handleQuery() {
	sourceID := m.queryInputSource.Value()
	if sourceID == "" {
		m.errMsg = i18n.GetText("error.invalid_input")
		return
	}

	limit := 10
	if m.queryInputLimit.Value() != "" {
		_, _ = fmt.Sscanf(m.queryInputLimit.Value(), "%d", &limit)
	}

	queryUseCase := oracleapp.NewGetDataUseCase(m.repo)
	result, err := queryUseCase.Execute(&oracleapp.GetDataRequest{SourceID: sourceID, Limit: limit})

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.queryResult = result
		m.view = "queryResult"
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
