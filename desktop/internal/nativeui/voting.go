package nativeui

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// votingState is the Voting page's state.
type votingState struct {
	voterName string
	// voterPub/voterPriv are the key pair from the last registration. They are
	// kept so the operator can copy them: the store keeps only the public half,
	// and native text is not selectable, so a key shown once in a status line
	// could never leave the window.
	voterPub  string
	voterPriv string

	candName    string
	candParty   string
	candProgram string
	candResult  statusLine
	// candCopyID is the id of the candidate just registered, kept so the card
	// can offer a Copy button. The candidate table shortens ids and its
	// selection feeds the ballot, not the session roster field.
	candCopyID string

	sessionTitle  string
	sessionDesc   string
	sessionStart  time.Time
	sessionEnd    time.Time
	sessionIDs    string
	sessionResult statusLine
	// sessionCopyID is the id of the session just created, offered for copying
	// the same way.
	sessionCopyID string

	controlID     string
	controlResult statusLine

	voteSession   string
	voteCandidate string
	voteVoterPub  string
	votePriv      string
	voteResult    statusLine

	resultsSession string
	results        []ResultRow
	// resultsLoadedID is the session the rows describe. resultsSession is the
	// form field and can name another session between reads; keeping them
	// apart is what lets a failed read report "stale" for the session it
	// actually loaded, and never pin one session's tally to another's id.
	resultsLoadedID string
	// resultsErr is the last failure to read that tally. The rows from the
	// previous successful read stay when they still describe the same session
	// and are reported as stale; with no rows the card explains the failure
	// instead of claiming no tally exists.
	resultsErr error

	candidates []CandidateRow
	sessions   []SessionRow
	// candSearch and sessionSearch narrow those lists as they grow; both
	// filters are case-insensitive and leave the selection pointing at the
	// filtered row.
	candSearch    string
	sessionSearch string
	// candApplied/sessionApplied are the queries the current highlights
	// belong to.
	candApplied    string
	sessionApplied string

	voterNameErr string
	candNameErr  string
	loadErr      error

	// tab is the stage shown: 0 setup (register/create), 1 ballot (controls and
	// cast), 2 results. The page used to stack all four areas at once, so a
	// first-time operator had to find the order in the sidebar of cards.
	tab int

	status statusLine

	sessionTable   ui.ListState
	candidateTable ui.ListState
	resultsTable   ui.ListState

	// sessionChoice/candChoice are the rows chosen in those tables, -1 for
	// none. The tables shorten UUIDs to fit, so choosing a row is the only way
	// the full id reaches the forms.
	sessionChoice int
	candChoice    int
}

func (a *App) loadVoting() {
	if a.services == nil {
		return
	}
	cands, err := a.services.Candidates()
	if err != nil {
		a.voting.loadErr = err
		return
	}
	sessions, err := a.services.Sessions()
	if err != nil {
		a.voting.loadErr = err
		return
	}
	a.voting.candidates = cands
	a.voting.sessions = sessions
	a.voting.loadErr = nil
}

// votingView splits the page into the three stages an operator moves through:
// setup (register voters/candidates, create a session), ballot (start/end a
// session, cast a signed ballot), and results. The four areas used to be
// stacked at once, so a first-time operator had no visible order.
func (a *App) votingView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		// loadVoting keeps both lists when either read fails, so the warning
		// is drawn once for the page and both lists keep their last successful
		// rows. It used to be invisible entirely: nothing rendered loadErr.
		if a.voting.loadErr != nil {
			loadWarning(c, "Failed to load voting data", a.voting.loadErr,
				len(a.voting.sessions) > 0 || len(a.voting.candidates) > 0)
		}
		ui.Tabs(c, &a.voting.tab, "Setup", "Ballot", "Results")
		switch a.voting.tab {
		case 0:
			twoColumns(c, func() {
				a.votingSessionListCard(c)
				a.votingCandidateListCard(c)
			}, func() {
				a.votingRegister(c)
				a.votingSessions(c)
			})
		case 1:
			a.votingVote(c)
		default:
			a.votingResults(c)
		}
	})
}

func (a *App) votingRegister(c *ui.Context) {
	formCard(c, "Register Voter", "Returns the key pair the voter signs ballots with; keep the private half.", func() {
		field(c, "Voter Name", &a.voting.voterName)
		fieldError(c, a.voting.voterNameErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Register Voter").Clicked() {
				a.registerVoter()
			}
		})
		if a.voting.voterPriv != "" {
			ui.Text(c, "New voter key pair").FontSize(textSize(c, 12)).Bold().TextColor(ui.Hex(ink))
			copyRow(c, "Public key", a.voting.voterPub, func() {
				a.voting.status = okStatus("Public key copied.")
			})
			copyRow(c, "Private key", a.voting.voterPriv, func() {
				a.voting.status = okStatus("Private key copied. The store keeps only the public half; keep it safe.")
			})
			ui.Text(c, "The store keeps only the public half, so this is the only copy of the signing key.").
				FontSize(textSize(c, 11)).TextColor(ui.Hex(auroraAmber))
		}
	})

	formCard(c, "Register Candidate", "Only the name is required; party and program are shown on the ballot when present.", func() {
		a.votingRegisterCandidateBody(c)
	})
}

func (a *App) votingRegisterCandidateBody(c *ui.Context) {
	field(c, "Candidate Name", &a.voting.candName)
	fieldError(c, a.voting.candNameErr)
	field(c, "Party (optional)", &a.voting.candParty)
	field(c, "Program (optional)", &a.voting.candProgram)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Register Candidate").Clicked() {
			a.registerCandidate()
		}
	})
	inlineStatus(c, a.voting.candResult)
	if a.voting.candCopyID != "" {
		copyRow(c, "Candidate ID", a.voting.candCopyID, func() {
			a.voting.status = okStatus("Candidate ID copied.")
		})
	}
}

// registerCandidate registers the candidate described by the card and keeps
// its id for the Copy button: the row list shortens ids, and the session
// roster field needs the full one.
func (a *App) registerCandidate() {
	a.voting.candNameErr = ""
	a.voting.candResult = statusLine{}
	a.voting.candCopyID = ""
	name := strings.TrimSpace(a.voting.candName)
	if name == "" {
		a.voting.candNameErr = "Candidate name is required."
		return
	}
	cand, err := a.services.RegisterCandidate(
		name,
		strings.TrimSpace(a.voting.candParty),
		strings.TrimSpace(a.voting.candProgram),
	)
	if err != nil {
		a.voting.candResult = errStatus(err)
		return
	}
	a.voting.candResult = okStatus("Registered %s", cand.Name)
	a.voting.candCopyID = cand.ID
	a.voting.candName, a.voting.candParty, a.voting.candProgram = "", "", ""
	a.loadVoting()
	// The new candidate is the context: highlight it, clear a search that
	// would hide it, and aim the ballot at it. The operator can still change
	// either choice before casting.
	a.selectCandidateByID(cand.ID)
	a.voting.voteCandidate = cand.ID
}

// selectCandidateByID highlights a candidate after registration.
func (a *App) selectCandidateByID(id string) {
	a.voting.candSearch, a.voting.candApplied = "", ""
	for i, candidate := range a.voting.candidates {
		if candidate.ID == id {
			a.voting.candChoice = i
			return
		}
	}
}

// registerVoter registers the named voter and carries the generated key pair
// into the ballot form.
func (a *App) registerVoter() {
	a.voting.voterNameErr = ""
	a.voting.status = statusLine{}
	name := strings.TrimSpace(a.voting.voterName)
	if name == "" {
		a.voting.voterNameErr = "Voter name is required."
		return
	}
	v, err := a.services.RegisterVoter(name)
	if err != nil {
		a.voting.status = errStatus(err)
		return
	}
	a.applyVoterRegistration(v)
}

// applyVoterRegistration reports the new voter and fills the ballot form from
// it.
//
// The registration generates the ballot key pair and the store keeps only the
// public half, so this is the one chance to hand the private key to the
// operator. The pair is kept in the page state so the card can offer a Copy
// button for both halves, and the ballot fields are filled so the voter can
// sign straight away instead of copying a key that is no longer readable
// anywhere.
func (a *App) applyVoterRegistration(v VoterRow) {
	a.voting.voterPub, a.voting.voterPriv = v.PublicKey, v.PrivateKey
	a.voting.status = okStatus("Registered %s. The ballot fields are filled from the new key pair.", v.Name)
	a.voting.voteVoterPub = v.PublicKey
	a.voting.votePriv = v.PrivateKey
	a.voting.voterName = ""
}

func (a *App) votingSessions(c *ui.Context) {
	formCard(c, "Create Session", "Choose the roster and the window during which ballots are accepted.", func() {
		a.votingCreateSessionBody(c)
	})
}

func (a *App) votingSessionListCard(c *ui.Context) {
	formCard(c, "Sessions", "Every governance window, its status and when ballots are accepted.", func() {
		a.votingSessionList(c)
	})
}

func (a *App) votingCreateSessionBody(c *ui.Context) {
	field(c, "Title", &a.voting.sessionTitle)
	field(c, "Description (optional)", &a.voting.sessionDesc)
	// The window is edited with the toolkit's own date and time controls:
	// asking an operator to type epoch seconds made the card unusable without
	// a clock and a converter outside the app. The two controls edit one
	// time.Time, shown in local time; the service still receives Unix seconds.
	ui.Field(c, "Start", func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.DateInput(c, &a.voting.sessionStart)
				ui.TimeInput(c, &a.voting.sessionStart)
			})
			hint(c, "Local time. Ballots are rejected before it.")
		})
	})
	ui.Field(c, "End", func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.DateInput(c, &a.voting.sessionEnd)
				ui.TimeInput(c, &a.voting.sessionEnd)
			})
			hint(c, "Local time. Ballots are rejected after it.")
		})
	})
	fieldHint(c, "Candidate IDs (comma-separated)", "Leave empty to include every registered candidate.", &a.voting.sessionIDs)
	ui.Row(c).Children(func() {
		if primaryButton(c, "Create Session").Clicked() {
			a.createVotingSession()
		}
	})
	inlineStatus(c, a.voting.sessionResult)
	if a.voting.sessionCopyID != "" {
		copyRow(c, "Session ID", a.voting.sessionCopyID, func() {
			a.voting.status = okStatus("Session ID copied.")
		})
	}
}

func (a *App) votingSessionList(c *ui.Context) {
	if len(a.voting.sessions) == 0 {
		// A failed load is reported once, above the cards; with no rows there
		// is nothing to keep, and "No sessions yet" would claim the store is
		// empty when it could not be read.
		if a.voting.loadErr == nil {
			emptyNote(c, "No sessions yet.")
		}
		return
	}
	sessions := make([]SessionRow, 0, len(a.voting.sessions))
	ui.SearchField(c, &a.voting.sessionSearch).Label("Search sessions")
	if a.voting.sessionSearch != a.voting.sessionApplied {
		a.voting.sessionApplied = a.voting.sessionSearch
		a.voting.sessionChoice = -1
	}
	for _, s := range a.voting.sessions {
		if containsFold(a.voting.sessionSearch, s.ID, s.Title, s.Status) {
			sessions = append(sessions, s)
		}
	}
	if len(sessions) == 0 {
		emptyNote(c, "No sessions match “"+strings.TrimSpace(a.voting.sessionSearch)+"”.")
		return
	}
	a.voting.sessionTable.Selected = &a.voting.sessionChoice
	table := ui.Table(c, &a.voting.sessionTable, []ui.TableColumn{
		{Title: "ID", Width: 140},
		// SingleLine on every cell: a wrapped cell is clipped by the row's
		// fixed height, which cut "Budget 2026" in half.
		{Title: "Title", MinWidth: 130},
		{Title: "Status", Width: 100},
		{Title: "Window", Width: 210},
	}, len(sessions), func(row, col int) {
		s := sessions[row]
		switch col {
		case 0:
			ui.Text(c, shortID(s.ID)).SingleLine()
		case 1:
			ui.Text(c, s.Title).SingleLine()
		case 2:
			ui.Text(c, s.Status)
		case 3:
			// The window is why a ballot is refused, so it belongs in the
			// list rather than only in the error a voter sees afterwards.
			ui.Text(c, sessionWindow(s.StartTime, s.EndTime)).FontSize(textSize(c, 12)).SingleLine()
		}
	})
	if table.Changed() && a.voting.sessionChoice >= 0 && a.voting.sessionChoice < len(sessions) {
		// The row shows a shortened UUID and every form needs the real one:
		// choosing the row loads it into the controls, the ballot and the
		// tally reader at once, the same contract the other pages' tables
		// follow.
		id := sessions[a.voting.sessionChoice].ID
		a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = id, id, id
		if len(a.voting.results) > 0 || a.voting.resultsErr != nil {
			// The tally on screen belongs to the previously chosen session; a
			// selection must not leave it beside a new session id even when
			// the fresh read fails. Same-session refreshes keep their rows.
			a.clearTallyForAnotherSession(id)
			_ = a.loadSessionResults(id)
		}
	}
	if a.voting.sessionChoice >= 0 && a.voting.sessionChoice < len(sessions) {
		// The row cannot show the roster (the window column already fills the
		// width), but an operator about to start or end a session needs to see
		// who is on the ballot and what the session is for.
		chosen := sessions[a.voting.sessionChoice]
		readout(c, []readoutRow{
			{"About", valueOr(chosen.Description)},
			{"Roster", a.sessionRosterNames(chosen.Candidates)},
		})
		copyRow(c, "Selected session ID", chosen.ID, nil)
	}
}

// sessionRosterNames renders a session's candidate ids as names, falling back
// to the shortened id when a candidate has been deleted.
func (a *App) sessionRosterNames(ids []string) string {
	if len(ids) == 0 {
		return "—"
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := ""
		for _, candidate := range a.voting.candidates {
			if candidate.ID == id {
				name = candidate.Name
				break
			}
		}
		if name == "" {
			name = shortID(id)
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// votingCandidateListCard lists the registered candidates. Before it existed
// the only place a candidate id appeared was the one-line result of
// Register Candidate: after the next action or a restart, nothing in the
// window could produce the id a session roster or a ballot requires.
func (a *App) votingCandidateListCard(c *ui.Context) {
	formCard(c, "Candidates", "Every registered candidate. Choose one to load its id into the ballot, or add it to the new session's roster.", func() {
		a.votingCandidateList(c)
	})
}

func (a *App) votingCandidateList(c *ui.Context) {
	if len(a.voting.candidates) == 0 {
		if a.voting.loadErr == nil {
			emptyNote(c, "No candidates registered yet.")
		}
		return
	}
	candidates := a.filteredCandidates()
	ui.SearchField(c, &a.voting.candSearch).Label("Search candidates")
	if a.voting.candSearch != a.voting.candApplied {
		a.voting.candApplied = a.voting.candSearch
		a.voting.candChoice = -1
	}
	if len(candidates) == 0 {
		emptyNote(c, "No candidates match “"+strings.TrimSpace(a.voting.candSearch)+"”.")
		return
	}
	a.voting.candidateTable.Selected = &a.voting.candChoice
	table := ui.Table(c, &a.voting.candidateTable, []ui.TableColumn{
		{Title: "ID", Width: 150},
		{Title: "Name", MinWidth: 120},
		{Title: "Party", MinWidth: 100},
	}, len(candidates), func(row, col int) {
		cand := candidates[row]
		switch col {
		case 0:
			ui.Text(c, shortID(cand.ID)).SingleLine()
		case 1:
			ui.Text(c, cand.Name).SingleLine()
		case 2:
			ui.Text(c, cand.Party).SingleLine()
		}
	})
	if table.Changed() && a.voting.candChoice >= 0 && a.voting.candChoice < len(candidates) {
		a.voting.voteCandidate = candidates[a.voting.candChoice].ID
	}
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Add to session roster").Clicked() {
			a.addCandidateToRoster()
		}
	})
	if a.voting.candChoice >= 0 && a.voting.candChoice < len(candidates) {
		// The table has no room for the program, which is the part of a
		// candidate's registration the ballot shows; the selected row carries
		// it below the list.
		chosen := candidates[a.voting.candChoice]
		readout(c, []readoutRow{
			{"Program", valueOr(chosen.Program)},
			{"Votes", strconv.Itoa(chosen.VoteCount)},
		})
		copyRow(c, "Selected candidate ID", chosen.ID, nil)
	}
}

// filteredCandidates is the candidate list the table shows: the stored rows
// narrowed by the search field. Selection indexes this slice, so every reader
// of candChoice must resolve through it rather than through the full list.
func (a *App) filteredCandidates() []CandidateRow {
	rows := make([]CandidateRow, 0, len(a.voting.candidates))
	for _, candidate := range a.voting.candidates {
		if containsFold(a.voting.candSearch, candidate.ID, candidate.Name, candidate.Party, candidate.Program) {
			rows = append(rows, candidate)
		}
	}
	return rows
}

// addCandidateToRoster appends the chosen candidate to the session form's
// roster field. The field takes comma-separated ids, so building a custom
// roster used to mean copying UUIDs by hand; this is the button that makes the
// form usable for anything but "every candidate".
func (a *App) addCandidateToRoster() {
	candidates := a.filteredCandidates()
	if a.voting.candChoice < 0 || a.voting.candChoice >= len(candidates) {
		a.voting.status = statusLine{text: "Choose a candidate first.", isErr: true}
		return
	}
	candidate := candidates[a.voting.candChoice]
	ids := splitParticipants(a.voting.sessionIDs)
	for _, id := range ids {
		if id == candidate.ID {
			a.voting.status = okStatus("%s is already in the session roster.", candidate.Name)
			return
		}
	}
	ids = append(ids, candidate.ID)
	a.voting.sessionIDs = strings.Join(ids, ",")
	a.voting.status = okStatus("Added %s to the session roster.", candidate.Name)
}

func (a *App) createVotingSession() {
	a.voting.sessionResult = statusLine{}
	a.voting.sessionCopyID = ""
	start := a.voting.sessionStart.Unix()
	end := a.voting.sessionEnd.Unix()
	if a.voting.sessionStart.IsZero() || a.voting.sessionEnd.IsZero() {
		a.voting.sessionResult = statusLine{text: "Choose both a start and an end time.", isErr: true}
		return
	}
	if end <= start {
		a.voting.sessionResult = statusLine{text: "End time must be after the start time.", isErr: true}
		return
	}
	ids := splitParticipants(a.voting.sessionIDs)
	if len(ids) == 0 {
		for _, c := range a.voting.candidates {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		a.voting.sessionResult = statusLine{text: "Register at least one candidate before creating a session.", isErr: true}
		return
	}
	s, err := a.services.CreateSession(strings.TrimSpace(a.voting.sessionTitle), strings.TrimSpace(a.voting.sessionDesc), ids, start, end)
	if err != nil {
		a.voting.sessionResult = errStatus(err)
		return
	}
	a.voting.sessionResult = okStatus("Created session %s", s.Title)
	a.voting.sessionCopyID = s.ID
	a.voting.sessionTitle, a.voting.sessionDesc, a.voting.sessionIDs = "", "", ""
	a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
	a.loadVoting()
	// The new session becomes the context for controls, ballot and tally;
	// highlight its row so a successful create has a visible result even
	// before the copy button is used.
	a.selectSessionByID(s.ID)
	a.clearTallyForAnotherSession(s.ID)
	a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = s.ID, s.ID, s.ID
}

// selectSessionByID highlights a session after creation. The search is
// cleared first so the index is in the list the table actually shows.
func (a *App) selectSessionByID(id string) {
	a.voting.sessionSearch, a.voting.sessionApplied = "", ""
	for i, session := range a.voting.sessions {
		if session.ID == id {
			a.voting.sessionChoice = i
			return
		}
	}
}

// defaultSessionWindow is the window the Create Session card starts with: now
// (to the minute) until a day later, so a session is creatable without
// touching the date and time controls.
func defaultSessionWindow(now time.Time) (start, end time.Time) {
	start = now.Truncate(time.Minute)
	return start, start.Add(24 * time.Hour)
}

func (a *App) votingVote(c *ui.Context) {
	formCard(c, "Session Controls", "Start or end a session by id.", func() {
		field(c, "Session ID", &a.voting.controlID)
		ui.Row(c).Gap(8).Children(func() {
			if secondaryButton(c, "Start Session").Clicked() {
				if err := a.services.StartSession(strings.TrimSpace(a.voting.controlID)); err != nil {
					a.voting.controlResult = errStatus(err)
				} else {
					a.voting.controlResult = okStatus("Session started.")
					a.loadVoting()
				}
			}
			if secondaryButton(c, "End Session").Clicked() {
				if err := a.services.EndSession(strings.TrimSpace(a.voting.controlID)); err != nil {
					a.voting.controlResult = errStatus(err)
				} else {
					a.voting.controlResult = okStatus("Session ended.")
					a.loadVoting()
				}
			}
		})
		inlineStatus(c, a.voting.controlResult)
	})

	formCard(c, "Cast Vote", "Sign a ballot for one candidate in one session.", func() {
		field(c, "Session ID", &a.voting.voteSession)
		field(c, "Candidate ID", &a.voting.voteCandidate)
		field(c, "Voter Public Key", &a.voting.voteVoterPub)
		field(c, "Voter Private Key", &a.voting.votePriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Cast Vote").Clicked() {
				a.castVote()
			}
		})
		inlineStatus(c, a.voting.voteResult)
	})
}

func (a *App) castVote() {
	a.voting.voteResult = statusLine{}
	if strings.TrimSpace(a.voting.voteSession) == "" || strings.TrimSpace(a.voting.voteCandidate) == "" {
		a.voting.voteResult = statusLine{text: "Choose a session and a candidate.", isErr: true}
		return
	}
	err := a.services.CastVote(
		strings.TrimSpace(a.voting.voteSession),
		strings.TrimSpace(a.voting.voteCandidate),
		strings.TrimSpace(a.voting.voteVoterPub),
		strings.TrimSpace(a.voting.votePriv),
	)
	if err != nil {
		a.voting.voteResult = errStatus(err)
		return
	}
	a.voting.voteResult = okStatus("Ballot recorded.")
	a.voting.votePriv = ""
	sessionID := strings.TrimSpace(a.voting.voteSession)
	a.loadVoting()
	// A tally already on screen for this session must include the ballot that
	// was just accepted; leaving the old numbers beside "Ballot recorded" read
	// as a vote that did not count. A failed refresh keeps the old rows rather
	// than replacing them with an error for a write that succeeded.
	if sessionID != "" && sessionID == strings.TrimSpace(a.voting.resultsSession) && len(a.voting.results) > 0 {
		_ = a.loadSessionResults(sessionID)
	}
}

// loadSessionResults reads a session's tally into the Results card. A failed
// read keeps the rows already on screen only when they describe the same
// session; callers that change the subject clear them first.
func (a *App) loadSessionResults(sessionID string) error {
	rows, err := a.services.SessionResults(sessionID)
	if err != nil {
		a.voting.resultsErr = err
		return err
	}
	a.voting.results = rows
	a.voting.resultsLoadedID = sessionID
	a.voting.resultsErr = nil
	return nil
}

// clearTallyForAnotherSession drops the rows when they describe a different
// session than the one about to be read, so a failed read cannot leave the
// previous session's numbers under the new id. Refreshing the same session
// keeps its rows and reports them as the last successful load.
func (a *App) clearTallyForAnotherSession(sessionID string) {
	if a.voting.resultsLoadedID == sessionID {
		return
	}
	a.voting.results = nil
	a.voting.resultsLoadedID = ""
	a.voting.resultsErr = nil
}

// sessionWindow renders a session's acceptance window as a compact pair of
// local times.
func sessionWindow(start, end int64) string {
	if start == 0 && end == 0 {
		return "—"
	}
	const layout = "2006-01-02 15:04"
	return time.Unix(start, 0).Format(layout) + " → " + time.Unix(end, 0).Format(layout)
}

func (a *App) votingResults(c *ui.Context) {
	formCard(c, "Results", "Live tally per candidate for a session.", func() {
		a.votingResultsBody(c)
	})
}

func (a *App) votingResultsBody(c *ui.Context) {
	field(c, "Session ID", &a.voting.resultsSession)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Show Results").Clicked() {
			id := strings.TrimSpace(a.voting.resultsSession)
			a.clearTallyForAnotherSession(id)
			if err := a.loadSessionResults(id); err != nil {
				a.voting.status = errStatus(err)
			} else {
				a.voting.status = statusLine{}
			}
		}
	})
	if a.voting.resultsErr != nil &&
		!loadWarning(c, "Failed to load results", a.voting.resultsErr, len(a.voting.results) > 0) {
		return
	}
	if len(a.voting.results) == 0 {
		emptyNote(c, "No tally loaded yet.")
		return
	}
	results := a.voting.results
	total := 0
	for _, r := range results {
		total += r.Votes
	}
	// The total is what a reader checks first ("did every ballot count?"); the
	// rows alone make them add the column up.
	readout(c, []readoutRow{{"Total votes", strconv.Itoa(total)}})
	ui.Table(c, &a.voting.resultsTable, []ui.TableColumn{
		{Title: "Candidate"},
		{Title: "Votes", Width: 80, Align: ui.End},
	}, len(results), func(row, col int) {
		r := results[row]
		if col == 0 {
			ui.Text(c, r.Name).SingleLine()
		} else {
			ui.Text(c, strconv.Itoa(r.Votes))
		}
	})
}
