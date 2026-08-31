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

// recorder persists lottery draws so the TUI and the CLI surfaces
// (lottery history/stats/export/verify) read the same lottery_records store.
// The CLI wiring injects the SQLite repository; a nil recorder leaves the TUI
// chain-only — used by tests and viewers that only exercise the block store.
type recorder interface {
	Save(record *lottery.LotteryRecord) error
	GetAll() ([]*lottery.LotteryRecord, error)
}

type model struct {
	view              string
	chain             *blockchain.BlockChain
	repo              recorder
	count             string
	result            *lottery.LotteryRecord
	err               string
	persistErr        string
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
		case "ctrl+c":
			// ctrl+c is the hard quit in every view.
			return m, tea.Quit
		case "q":
			if m.view == "menu" {
				return m, tea.Quit
			}
			// In a non-menu view, q must fall through when it would be a
			// typable character: the create form's participants/seed/count
			// inputs accept free text, and the help screen scopes q to the
			// menu. Without the fall-through every "q" keystroke reset the
			// view back to the menu, making names like "quack"/"Aq"
			// untypeable (TASK-161, ISS-154). In read-only views (history,
			// result) q keeps the classic back-to-menu behaviour.
			if m.view != "create" {
				m.view = "menu"
				m.err = ""
				m.successMsg = ""
				return m, nil
			}

		case "?":
			// scoped to non-form views: in the create form "?" is a typable
			// character (seeds like "a?b", names with "?"), mirroring the
			// q/esc fall-through convention (TASK-161, ISS-154; ISS-164).
			if m.view != "create" {
				m.showHelp = true
				return m, nil
			}
			// create form: fall through to the focused textinput below.

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
		case "up":
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
		case "k":
			// In the create form "k" is a typable letter (participants like
			// "Jack", seeds with 'k') and must fall through to the focused
			// textinput below — arrow keys and Tab are the form-navigation
			// keys. In read-only views the viewport (history) handles "k" as
			// scroll-up; only the menu treats the bare letter as navigation
			// (TASK-161 convention extended to j/k, ISS-164).
			if m.view == "menu" && m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down":
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
		case "j":
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

	// The history view is a viewport; let the viewport handle scrolling keys
	// (up/down/j/k/pgup/pgdn/space/b/f/u/d) so long histories are reachable
	// instead of clipped at the 15-row viewport (TASK-127, ISS-119).
	if m.view == "history" {
		m.viewport, cmd = m.viewport.Update(msg)
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
	s := ""
	// A draw's created-on-chain feedback (and any persist warning appended by
	// handleCreate) is otherwise set and cleared on every exit of this view
	// without ever being rendered; surface it here.
	if m.successMsg != "" {
		s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
	}

	if m.result == nil {
		return i18n.GetText("error.not_found")
	}

	s += components.SuccessStyle().Render("🎉 "+i18n.GetText("lottery.tui.completed")) + "\n\n"
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
	// The help screen is shared and fully localized (components.HelpView);
	// the inline layout above was hardcoded CJK that leaked into en-locale
	// sessions (TASK-131, ISS-120) and duplicated the other TUIs' copy.
	return components.HelpView()
}

func (m *model) handleCreate() tea.Msg {
	participants := parseTextArea(m.participantsInput.Value())
	seed := m.seedInput.Value()
	// A non-numeric / cleared winners field must be a visible error, not a
	// silent 3-winner draw (the sibling count checks have their own messages);
	// before this the fallback count masked operator mistakes (TASK-165,
	// ISS-158).
	count, err := strconv.Atoi(m.countInput.Value())
	if err != nil {
		m.err = i18n.GetText("lottery.tui.winners_invalid")
		return nil
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
	if m.persistErr != "" {
		m.successMsg += " · " + m.persistErr
	}

	return nil
}

func (m *model) loadHistory() {
	if m.repo != nil {
		// Read the persistent lottery_records store so the TUI shows exactly
		// what `lottery history` shows — TUI draws (dual-written in
		// runLottery) and imported draws alike, and nothing else. Reading the
		// chain's raw blocks here also used to list unrelated on-chain records
		// (votes, transfers) as "lotteries" (TASK-203, ISS-199).
		records, err := m.repo.GetAll()
		if err != nil {
			m.viewport.SetContent(i18n.GetText("error.load_failed") + ": " + err.Error())
			return
		}
		if len(records) == 0 {
			m.viewport.SetContent(i18n.GetText("lottery.tui.no_records") + "\n\n" +
				components.HelpTextStyle().Render(i18n.GetText("lottery.tui.create_hint")))
			return
		}
		var content string
		for i, r := range records {
			jsonData, jerr := r.ToJSON()
			if jerr != nil {
				continue
			}
			content += fmt.Sprintf(i18n.GetText("lottery.tui.history_item"), i+1, jsonData)
		}
		m.viewport.SetContent(content)
		return
	}

	// No recorder wired (tests / chain-only view): fall back to the chain.
	records := m.chain.GetLotteryRecords()
	if len(records) == 0 {
		m.viewport.SetContent(i18n.GetText("lottery.tui.no_records") + "\n\n" +
			components.HelpTextStyle().Render(i18n.GetText("lottery.tui.create_hint")))
	} else {
		var content string
		for i, data := range records {
			content += fmt.Sprintf(i18n.GetText("lottery.tui.history_item"), i+1, data)
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

	// Persist to the same lottery_records store the CLI history/stats/export/
	// verify read, mirroring CreateLotteryUseCase which dual-writes blocks +
	// lottery_records. The draw is already on-chain; if the record write fails
	// (rare, same DB), flag it so the result view doesn't claim full history
	// persistence (TASK-203, ISS-199).
	if m.repo != nil {
		if err := m.repo.Save(record); err != nil {
			m.persistErr = fmt.Sprintf("%s: %v", i18n.GetText("lottery.tui.persist_failed"), err)
		} else {
			// A previous draw's persistence failure must not keep tainting
			// later draw's "created on-chain" text once saving works again.
			m.persistErr = ""
		}
	}

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

func RunLotteryTUI(repo recorder) error {
	app := NewLotteryApp()
	app.repo = repo
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
