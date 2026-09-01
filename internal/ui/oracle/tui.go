package oracle

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	domainoracle "github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ui/components"
)

// addSourceFieldCount is the number of fields in the add-source form; the
// up/down/tab handlers and updateInputFocus must agree on it so focus cycles
// the whole form (name, url, type, method, path, interval).
const addSourceFieldCount = 6

type model struct {
	view        string
	repo        domainoracle.Repository
	menuIndex   int
	sources     []*domainoracle.DataSource
	listUseCase *oracleapp.ListSourcesUseCase

	sourceInputName     textinput.Model
	sourceInputURL      textinput.Model
	sourceInputType     textinput.Model
	sourceInputMethod   textinput.Model
	sourceInputPath     textinput.Model
	sourceInputInterval textinput.Model
	fetchInputSource    textinput.Model
	queryInputSource    textinput.Model
	queryInputLimit     textinput.Model
	inputFocus          int
	showHelp            bool
	selectedSourceID    string
	confirmAction       string
	errMsg              string
	successMsg          string
	fetchResult         *oracleapp.FetchDataResponse
	queryResult         *oracleapp.GetDataResponse

	// chain records TUI-fetched data on the ledger. Every other fetch surface
	// (REST handler, scheduler, CLI) calls FetchDataUseCase.SetChain; without
	// it handleFetch saves observations at block_height=0 with no ledger block
	// (TASK-180, mirrors the TASK-097 scheduler regression). Nil (tests only)
	// means "no on-chain recording", matching the CLI's default.
	chain oracleapp.ChainInterface

	// viewport bounds the read-only result views (fetchResult/queryResult)
	// so multi-row results scroll instead of rendering past the terminal
	// edge (TASK-176, ISS-174). sources is a navigable menu, so it gets its
	// own bounded window (see sourcesView) that follows the cursor.
	viewport viewport.Model
	// menuRows is the maximum number of rows sourcesView renders for the
	// source menu, sized from the terminal height in WindowSizeMsg so a long
	// source list stays reachable. Default 15 until the first resize lands.
	menuRows int
}

func NewOracleApp(repo domainoracle.Repository) *model {
	nameInput := textinput.New()
	nameInput.Placeholder = i18n.GetText("oracle.tui.enter_name")
	nameInput.Focus()

	urlInput := textinput.New()
	urlInput.Placeholder = i18n.GetText("oracle.tui.enter_url")

	typeInput := textinput.New()
	typeInput.Placeholder = i18n.GetText("oracle.tui.enter_type")

	methodInput := textinput.New()
	methodInput.Placeholder = i18n.GetText("oracle.tui.enter_method")

	pathInput := textinput.New()
	pathInput.Placeholder = i18n.GetText("oracle.tui.enter_path")

	intervalInput := textinput.New()
	intervalInput.Placeholder = i18n.GetText("oracle.tui.enter_interval")

	fetchInput := textinput.New()
	fetchInput.Placeholder = i18n.GetText("oracle.tui.enter_source_id")

	queryInputSource := textinput.New()
	queryInputSource.Placeholder = i18n.GetText("oracle.tui.enter_source_id")

	queryInputLimit := textinput.New()
	queryInputLimit.Placeholder = i18n.GetText("oracle.tui.enter_limit")

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))

	return &model{
		view:                "menu",
		repo:                repo,
		listUseCase:         oracleapp.NewListSourcesUseCase(repo),
		sourceInputName:     nameInput,
		sourceInputURL:      urlInput,
		sourceInputType:     typeInput,
		sourceInputMethod:   methodInput,
		sourceInputPath:     pathInput,
		sourceInputInterval: intervalInput,
		fetchInputSource:    fetchInput,
		queryInputSource:    queryInputSource,
		queryInputLimit:     queryInputLimit,
		inputFocus:          0,

		viewport: vp,
		menuRows: 15,
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
			// source name/url/type/fetch/query inputs (the help screen scopes
			// q to the menu); read-only views keep back-to-menu.
			switch m.view {
			case "sources", "sourceDetail", "confirmToggle", "confirmDelete",
				"fetchResult", "queryResult":
				m.view = "menu"
				m.errMsg = ""
				m.successMsg = ""
				return m, nil
			}

		case "?":
			// scoped to non-form views: in the form views "?" is a typable
			// character — the add-source/fetch/query URL fields routinely carry
			// query strings ("https://api.example.com/data?format=json") —
			// mirroring the q fall-through convention (TASK-161, ISS-154;
			// ISS-164). Forms can still reach help via esc to the menu.
			switch m.view {
			case "addSource", "fetch", "query":
				// fall through to the focused textinput below.
			default:
				m.showHelp = true
				return m, nil
			}

		case "1", "2", "3", "4":
			// Menu quick-select, mirroring the lottery/nft/token TUIs (TASK-…,
			// the "↑↓ select + Enter" affordance is the same, so the digit
			// shortcuts should be too). Outside the menu digits fall through
			// so they stay typable in the fetch/query input fields.
			if m.view == "menu" {
				m.menuIndex = int(msg.String()[0] - '1')
			}

		case "up":
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
			// Arrows are always navigation; never let them also reach the
			// textinput below. In the read-only result views the arrow falls
			// through to the viewport scroll forward at the bottom instead.
			if m.view != "fetchResult" && m.view != "queryResult" {
				return m, nil
			}

		case "k":
			// Typable letter in form views: falls through to the focused
			// textinput below so a source URL/name can contain 'k'. Only the
			// menu/source-list and confirm dialogs (no free-text field)
			// navigate on the bare letter. Previously 'k' moved focus AND
			// typed the letter into the newly-focused field (ISS-164).
			switch m.view {
			case "menu":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "sources":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "confirmToggle", "confirmDelete":
				m.menuIndex = 0
			}

		case "down":
			switch m.view {
			case "menu":
				if m.menuIndex < 3 {
					m.menuIndex++
				}
			case "sources":
				// sourcesView renders "[Add source]" + one row per source, so
				// valid indices are 0..len(m.sources). Allowing len+1 let the
				// cursor advance past the last row (invisible, Enter dead) —
				// e.g. with zero sources one ↓ set menuIndex=1 on a 1-row list
				// and only a subsequent ↑ recovered it (TASK-164, ISS-157).
				if m.menuIndex < len(m.sources) {
					m.menuIndex++
				}
			case "addSource":
				if m.inputFocus < addSourceFieldCount-1 {
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
			// Arrows are always navigation; never let them also reach the
			// textinput below. In the read-only result views the arrow falls
			// through to the viewport scroll forward at the bottom instead.
			if m.view != "fetchResult" && m.view != "queryResult" {
				return m, nil
			}

		case "j":
			// Typable letter in form views (see "k" above); navigation only in
			// views with no free-text field.
			switch m.view {
			case "menu":
				if m.menuIndex < 3 {
					m.menuIndex++
				}
			case "sources":
				if m.menuIndex < len(m.sources) {
					m.menuIndex++
				}
			case "confirmToggle", "confirmDelete":
				m.menuIndex = 1
			}

		case "tab":
			switch m.view {
			case "addSource":
				m.inputFocus = (m.inputFocus + 1) % addSourceFieldCount
				m.updateInputFocus()
			case "fetch":
				// The fetch view has a single input; cycling (mod 2) could
				// land focus on a non-existent second field, blur the only
				// input and leave typing dead. Keep focus on it (mirrors
				// the 'down' handling for this view).
				m.inputFocus = 0
				m.updateFetchInputFocus()
			case "query":
				m.inputFocus = (m.inputFocus + 1) % 2
				m.updateQueryInputFocus()
			}
			// Tab is always navigation; never let it also reach the focused
			// textinput below (consistent with lottery/nft/token TUIs).
			return m, nil

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
			}

		case "y", "Y":
			// Confirm-dialog hotkeys: the highlighted ▶ row is what Enter
			// executes, and y/n select those same choices directly.
			switch m.view {
			case "confirmToggle":
				m.handleToggleSource()
				m.menuIndex = 0
			case "confirmDelete":
				m.handleDeleteSource()
				m.menuIndex = 0
			}

		case "n", "N":
			if m.view == "confirmToggle" || m.view == "confirmDelete" {
				m.view = "sources"
				m.menuIndex = 0
				m.errMsg = ""
			}

		case "d", "D":
			if m.view == "sourceDetail" && m.selectedSourceID != "" {
				m.confirmAction = "delete"
				m.menuIndex = 1
				m.view = "confirmDelete"
			}
		case "t", "T":
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
		// Size the result viewport and the source-menu window off the
		// terminal so long content scrolls instead of clipping (TASK-176,
		// ISS-174). The -12 budget mirrors the lottery/token/nft viewports.
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 12)
		if rows := msg.Height - 12; rows > 3 {
			m.menuRows = rows
		}
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

	// The result views are bounded viewports; let the viewport handle the
	// scroll keys (up/down/j/k/pgup/pgdn/space/b/f/u/d) so long query/fetch
	// results are reachable instead of clipped at the terminal edge
	// (TASK-176, ISS-174).
	if m.view == "fetchResult" || m.view == "queryResult" {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	if m.showHelp {
		v.SetContent(components.HelpView())
		return v
	}
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

	// Bounded window that follows the cursor: render at most menuRows rows
	// centred on the selection so a long source list stays reachable instead
	// of spilling past the terminal edge (TASK-176, ISS-174).
	start, end := m.menuWindow(len(menuItems))
	for i := start; i < end; i++ {
		item := menuItems[i]
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

// menuWindow returns the [start, end) range of a menu with total items that
// fits in the terminal, kept centred on the cursor so the highlighted row is
// always visible (TASK-176, ISS-174).
func (m *model) menuWindow(total int) (int, int) {
	if total <= m.menuRows {
		return 0, total
	}
	if m.menuRows < 1 {
		m.menuRows = 1
	}
	start := m.menuIndex - m.menuRows/2
	if start < 0 {
		start = 0
	}
	if start+m.menuRows > total {
		start = total - m.menuRows
	}
	return start, start + m.menuRows
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

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_method") + ":\n")
	if m.inputFocus == 3 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputMethod.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputMethod.View() + "\n\n"
	}

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_path") + ":\n")
	if m.inputFocus == 4 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputPath.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputPath.View() + "\n\n"
	}

	s += components.CaptionStyle().Render(i18n.GetText("oracle.tui.source_interval") + ":\n")
	if m.inputFocus == 5 {
		s += components.MenuSelectedStyle().Render("> "+m.sourceInputInterval.View()) + "\n\n"
	} else {
		s += "  " + m.sourceInputInterval.View() + "\n\n"
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

	s := components.HeaderStyle().Render("🔎 "+i18n.GetText("oracle.tui.edit_source")) + "\n\n"

	if source != nil {
		s += components.KeyValue(i18n.GetText("oracle.tui.source_name"), source.Name) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_url"), source.URL) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_type"), source.Type) + "\n"

		// Method/path/interval round out the feed-tailoring fields the CLI
		// `source list` prints (TASK-225); the detail view is where an operator
		// reads a source's full configuration, so keep it in parity with the
		// CLI instead of showing only the subset (TASK-229, ISS-227).
		method := source.Method
		if method == "" {
			method = "GET"
		}
		s += components.KeyValue(i18n.GetText("oracle.tui.source_method"), method) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_path"), source.Path) + "\n"

		interval := source.Interval
		if interval == 0 {
			// 0 means "unset -> default 60" (AddSourceUseCase maps it); show the
			// effective value instead of a bare 0 that looks like "no interval".
			interval = 60
		}
		s += components.KeyValue(i18n.GetText("oracle.tui.source_interval"), fmt.Sprintf("%ds", interval)) + "\n"

		status := i18n.GetText("oracle.tui.enabled")
		if !source.Enabled {
			status = i18n.GetText("oracle.tui.disabled")
		}
		s += components.KeyValue(i18n.GetText("oracle.tui.status"), status) + "\n"
	}

	s += "\n"
	s += components.BorderStyle().Render(i18n.GetText("oracle.tui.toggle"))
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

	s += "\n" + m.confirmChoiceView()
	s += components.HelpTextStyle().Render(i18n.GetText("oracle.tui.yes_no")) + "\n"
	s += "[ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	// A failed toggle/delete keeps the operator on this dialog; surface the
	// reason here (the sources view it returns to would clear it on ESC).
	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}
	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

	return s
}

// confirmChoiceView renders the Yes/No picker for the confirm dialogs. The
// highlighted row (▶) follows menuIndex (0=Yes/confirm, 1=No/cancel), giving
// the user a visible answer to "what will Enter do" — previously the dialogs
// showed a static "[Y]es/[N]o" banner with no selection marker, and the y/n
// hotkeys were dead (TASK-129, ISS-121).
func (m *model) confirmChoiceView() string {
	yesItem := i18n.GetText("oracle.tui.yes")
	noItem := i18n.GetText("oracle.tui.no")

	if m.menuIndex == 0 {
		return components.MenuActiveStyle().Render("▶ "+yesItem+"\n") +
			components.MenuInactiveStyle().Render("  "+noItem+"\n") + "\n"
	}
	return components.MenuInactiveStyle().Render("  "+yesItem+"\n") +
		components.MenuActiveStyle().Render("▶ "+noItem+"\n") + "\n"
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

	s += "\n" + m.confirmChoiceView()
	s += components.HelpTextStyle().Render(i18n.GetText("oracle.tui.yes_no")) + "\n"
	s += "[ESC] " + i18n.GetText("lottery.tui.back") + "\n"

	// A failed toggle/delete keeps the operator on this dialog; surface the
	// reason here (the sources view it returns to would clear it on ESC).
	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}
	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

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

	s += m.viewport.View() + "\n"

	if m.successMsg != "" {
		s += "\n" + components.SuccessStyle().Render(m.successMsg)
	}

	s += "\n" + components.BorderStyle().Render("[ESC] "+i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) queryResultView() string {
	s := components.HeaderStyle().Render("📊 "+i18n.GetText("oracle.tui.query_result")) + "\n\n"

	s += m.viewport.View() + "\n"

	if m.errMsg != "" {
		s += "\n" + components.ErrorStyle().Render(m.errMsg)
	}

	s += components.BorderStyle().Render("[ESC] " + i18n.GetText("lottery.tui.back"))

	return s
}

func (m *model) loadSources() {
	list, err := m.listUseCase.Execute(&oracleapp.ListSourcesRequest{})
	if err != nil {
		m.errMsg = i18n.GetText("error.load_failed")
		m.sources = nil
		return
	}
	m.sources = make([]*domainoracle.DataSource, len(list.Sources))
	for i, s := range list.Sources {
		m.sources[i] = &domainoracle.DataSource{
			ID:       s.ID,
			Name:     s.Name,
			URL:      s.URL,
			Type:     s.Type,
			Method:   s.Method,
			Path:     s.Path,
			Interval: s.Interval,
			Enabled:  s.Enabled,
		}
	}
	m.errMsg = ""
}

func (m *model) initAddSource() {
	// Re-create every field, not just the first three: method/path/interval are
	// textinput models too, and an init that skipped them left their previous
	// values on screen the next time the form opened (TASK-228, ISS-226; the
	// token/nft/lottery TUIs all clear every field on form entry).
	m.sourceInputName = textinput.New()
	m.sourceInputName.Placeholder = i18n.GetText("oracle.tui.enter_name")
	m.sourceInputName.Focus()

	m.sourceInputURL = textinput.New()
	m.sourceInputURL.Placeholder = i18n.GetText("oracle.tui.enter_url")

	m.sourceInputType = textinput.New()
	m.sourceInputType.Placeholder = i18n.GetText("oracle.tui.enter_type")

	m.sourceInputMethod = textinput.New()
	m.sourceInputMethod.Placeholder = i18n.GetText("oracle.tui.enter_method")

	m.sourceInputPath = textinput.New()
	m.sourceInputPath.Placeholder = i18n.GetText("oracle.tui.enter_path")

	m.sourceInputInterval = textinput.New()
	m.sourceInputInterval.Placeholder = i18n.GetText("oracle.tui.enter_interval")

	m.inputFocus = 0
	m.errMsg = ""
	m.successMsg = ""
}

func (m *model) updateInputFocus() {
	m.sourceInputName.Blur()
	m.sourceInputURL.Blur()
	m.sourceInputType.Blur()
	m.sourceInputMethod.Blur()
	m.sourceInputPath.Blur()
	m.sourceInputInterval.Blur()

	switch m.inputFocus {
	case 0:
		m.sourceInputName.Focus()
	case 1:
		m.sourceInputURL.Focus()
	case 2:
		m.sourceInputType.Focus()
	case 3:
		m.sourceInputMethod.Focus()
	case 4:
		m.sourceInputPath.Focus()
	case 5:
		m.sourceInputInterval.Focus()
	}
}

func (m *model) handleAddSource() {
	name := m.sourceInputName.Value()
	url := m.sourceInputURL.Value()
	sourceType := m.sourceInputType.Value()
	method := strings.TrimSpace(m.sourceInputMethod.Value())
	path := strings.TrimSpace(m.sourceInputPath.Value())
	intervalStr := strings.TrimSpace(m.sourceInputInterval.Value())

	if strings.TrimSpace(name) == "" || strings.TrimSpace(url) == "" {
		m.errMsg = i18n.GetText("error.invalid_input")
		return
	}

	if sourceType == "" {
		sourceType = "custom"
	}

	// Interval is optional; an empty field means "default" (0 -> the use case
	// applies 60). A non-numeric or negative value is a clear client error the
	// domain would only surface as a wrapped ErrInvalidSource.
	var interval int
	if intervalStr != "" {
		n, err := strconv.Atoi(intervalStr)
		if err != nil || n < 0 {
			m.errMsg = i18n.GetText("error.invalid_input")
			return
		}
		interval = n
	}

	addUseCase := oracleapp.NewAddSourceUseCase(m.repo)
	_, err := addUseCase.Execute(&oracleapp.AddSourceRequest{
		Name:     strings.TrimSpace(name),
		URL:      strings.TrimSpace(url),
		Type:     sourceType,
		Method:   method,
		Path:     path,
		Interval: interval,
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

	result, err := m.newFetchUseCase().Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.fetchResult = result
		m.successMsg = i18n.GetText("oracle.tui.fetch_success")
		m.loadFetchResult()
		m.view = "fetchResult"
	}
}

// loadFetchResult renders the single fetch record into the bounded result
// viewport so a long value scrolls instead of clipping (TASK-176, ISS-174).
func (m *model) loadFetchResult() {
	if m.fetchResult == nil {
		m.viewport.SetContent("")
		return
	}
	s := components.KeyValue("ID", m.fetchResult.ID) + "\n"
	s += components.KeyValue(i18n.GetText("oracle.tui.source_id"), m.fetchResult.SourceID) + "\n"
	s += components.KeyValue(i18n.GetText("oracle.tui.value"), m.fetchResult.Value) + "\n"
	s += components.KeyValue(i18n.GetText("oracle.tui.timestamp"), fmt.Sprintf("%d", m.fetchResult.Timestamp)) + "\n"
	s += components.KeyValue(i18n.GetText("oracle.tui.block_height"), fmt.Sprintf("%d", m.fetchResult.BlockHeight)) + "\n"
	m.viewport.SetContent(s)
	m.viewport.GotoTop()
}

func (m *model) handleQuery() {
	sourceID := m.queryInputSource.Value()
	if sourceID == "" {
		m.errMsg = i18n.GetText("error.invalid_input")
		return
	}

	// The CLI (clampQueryLimit) and the REST API (maxQueryLimit) both bound
	// the query limit to [1, maxTUIQueryLimit]; the TUI reached the use case
	// directly, so a stray/inflated number (e.g. 999999999) would have forced
	// an unbounded DB scan that the other two surfaces reject (TASK-178).
	limit := clampQueryLimitValue(m.queryInputLimit.Value())

	queryUseCase := oracleapp.NewGetDataUseCase(m.repo)
	result, err := queryUseCase.Execute(&oracleapp.GetDataRequest{SourceID: sourceID, Limit: limit})

	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.queryResult = result
		m.loadQueryResult()
		m.view = "queryResult"
	}
}

// loadQueryResult renders the possibly multi-row query result into the bounded
// result viewport so rows scroll instead of clipping (TASK-176, ISS-174).
func (m *model) loadQueryResult() {
	if m.queryResult == nil || len(m.queryResult.Data) == 0 {
		m.viewport.SetContent(i18n.GetText("oracle.tui.no_data"))
		return
	}
	var s string
	for i, d := range m.queryResult.Data {
		s += components.CaptionStyle().Render(fmt.Sprintf("--- #%d ---", i+1)) + "\n"
		s += components.KeyValue("ID", d.ID) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.source_id"), d.SourceID) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.value"), d.Value) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.timestamp"), fmt.Sprintf("%d", d.Timestamp)) + "\n"
		s += components.KeyValue(i18n.GetText("oracle.tui.block_height"), fmt.Sprintf("%d", d.BlockHeight)) + "\n\n"
	}
	m.viewport.SetContent(s)
	m.viewport.GotoTop()
}

// maxTUIQueryLimit mirrors the REST API handler cap (maxQueryLimit=100) and the
// CLI's maxCLIQueryLimit so the TUI cannot drive an unbounded DB scan either
// (TASK-178).
const maxTUIQueryLimit = 100

// clampQueryLimitValue parses the query limit input and clamps it into
// [1, maxTUIQueryLimit], defaulting to 10 when empty/invalid/<=0 — the same
// contract as the CLI's clampQueryLimit and the API handler.
func clampQueryLimitValue(raw string) int {
	if raw == "" {
		return 10
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return 10
	}
	if v > maxTUIQueryLimit {
		return maxTUIQueryLimit
	}
	return v
}

// newFetchUseCase builds the FetchDataUseCase used by handleFetch, wiring the
// on-chain recorder exactly like the REST handler, scheduler and CLI paths do.
// Skipping SetChain saves observations at block_height=0 and loses the ledger
// block (TASK-180, the TASK-097 class). The FetchDataUseCase.Chain() seam makes
// the wiring observable to a regression test without a network fetch.
func (m *model) newFetchUseCase() *oracleapp.FetchDataUseCase {
	uc := oracleapp.NewFetchDataUseCase(m.repo)
	uc.SetChain(m.chain)
	return uc
}

// SetChain wires the on-chain recorder for TUI fetches, mirroring the REST
// handler / scheduler / CLI paths. Without a chain, handleFetch would save
// observations at block_height=0 (TASK-180).
func (m *model) SetChain(chain oracleapp.ChainInterface) {
	m.chain = chain
}

func RunOracleTUI(repo domainoracle.Repository) error {
	app := NewOracleApp(repo)
	// Same singleton the CLI fetch command wires (cmd/aurora/cmd/oracle.go);
	// initConfig has already resolved db.path by the time the TUI runs.
	app.SetChain(blockchain.GetBlockChain())
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
