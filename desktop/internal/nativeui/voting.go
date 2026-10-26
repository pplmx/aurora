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
	candResult  string
	// candCopyID is the id of the candidate just registered, kept so the card
	// can offer a Copy button. The candidate table shortens ids and its
	// selection feeds the ballot, not the session roster field.
	candCopyID string

	sessionTitle  string
	sessionDesc   string
	sessionStart  time.Time
	sessionEnd    time.Time
	sessionIDs    string
	sessionResult string
	// sessionCopyID is the id of the session just created, offered for copying
	// the same way.
	sessionCopyID string

	controlID     string
	controlResult string

	voteSession   string
	voteCandidate string
	voteVoterPub  string
	votePriv      string
	voteResult    string

	resultsSession string
	results        []ResultRow

	candidates []CandidateRow
	sessions   []SessionRow

	voterNameErr string
	candNameErr  string
	loadErr      error

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

// votingView pairs the setup work (who votes, who runs, in which window) with
// the act of voting and reading the tally: two columns, because a session
// operator moves between them constantly and scrolling past four stacked forms
// to reach the ballot was the wrong shape for that.
func (a *App) votingView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		// loadVoting keeps both lists when either read fails, so the warning
		// is drawn once for the page and both lists keep their last successful
		// rows. It used to be invisible entirely: nothing rendered loadErr.
		if a.voting.loadErr != nil {
			loadWarning(c, "Failed to load voting data", a.voting.loadErr,
				len(a.voting.sessions) > 0 || len(a.voting.candidates) > 0)
		}
		// The session list leads: an operator reads what exists before
		// registering more of it, and it was the one card pushed below the
		// fold by three registration forms.
		twoColumns(c, func() {
			a.votingSessionListCard(c)
			a.votingCandidateListCard(c)
			a.votingRegister(c)
		}, func() {
			a.votingVote(c)
			a.votingSessions(c)
			a.votingResults(c)
		})
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
			ui.Text(c, "New voter key pair").FontSize(12).Bold().TextColor(ui.Hex(ink))
			copyRow(c, "Public key", a.voting.voterPub, func() {
				a.voting.status = okStatus("Public key copied.")
			})
			copyRow(c, "Private key", a.voting.voterPriv, func() {
				a.voting.status = okStatus("Private key copied. The store keeps only the public half; keep it safe.")
			})
			ui.Text(c, "The store keeps only the public half, so this is the only copy of the signing key.").
				FontSize(11).TextColor(ui.Hex(auroraAmber))
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
	if a.voting.candResult != "" {
		ui.Text(c, a.voting.candResult).FontSize(12).TextColor(ui.Hex(inkMuted))
	}
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
	a.voting.candResult = ""
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
		a.voting.candResult = err.Error()
		return
	}
	a.voting.candResult = "Registered " + cand.Name
	a.voting.candCopyID = cand.ID
	a.voting.candName, a.voting.candParty, a.voting.candProgram = "", "", ""
	a.loadVoting()
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
	if a.voting.sessionResult != "" {
		ui.Text(c, a.voting.sessionResult).FontSize(12).TextColor(ui.Hex(inkMuted))
	}
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
	sessions := a.voting.sessions
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
			ui.Text(c, sessionWindow(s.StartTime, s.EndTime)).FontSize(12).SingleLine()
		}
	})
	if table.Changed() && a.voting.sessionChoice >= 0 && a.voting.sessionChoice < len(sessions) {
		// The row shows a shortened UUID and every form needs the real one:
		// choosing the row loads it into the controls, the ballot and the
		// tally reader at once, the same contract the other pages' tables
		// follow.
		id := sessions[a.voting.sessionChoice].ID
		a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = id, id, id
	}
}

// votingCandidateListCard lists the registered candidates. Before it existed
// the only place a candidate id appeared was the one-line result of
// Register Candidate: after the next action or a restart, nothing in the
// window could produce the id a session roster or a ballot requires.
func (a *App) votingCandidateListCard(c *ui.Context) {
	formCard(c, "Candidates", "Every registered candidate. Choose one to load its id into the ballot.", func() {
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
	candidates := a.voting.candidates
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
}

func (a *App) createVotingSession() {
	a.voting.sessionResult = ""
	a.voting.sessionCopyID = ""
	start := a.voting.sessionStart.Unix()
	end := a.voting.sessionEnd.Unix()
	if a.voting.sessionStart.IsZero() || a.voting.sessionEnd.IsZero() {
		a.voting.sessionResult = "Choose both a start and an end time."
		return
	}
	if end <= start {
		a.voting.sessionResult = "End time must be after the start time."
		return
	}
	ids := splitParticipants(a.voting.sessionIDs)
	if len(ids) == 0 {
		for _, c := range a.voting.candidates {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		a.voting.sessionResult = "Register at least one candidate before creating a session."
		return
	}
	s, err := a.services.CreateSession(strings.TrimSpace(a.voting.sessionTitle), strings.TrimSpace(a.voting.sessionDesc), ids, start, end)
	if err != nil {
		a.voting.sessionResult = err.Error()
		return
	}
	a.voting.sessionResult = "Created session " + s.Title
	a.voting.sessionCopyID = s.ID
	a.voting.sessionTitle, a.voting.sessionDesc, a.voting.sessionIDs = "", "", ""
	a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
	a.loadVoting()
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
					a.voting.controlResult = err.Error()
				} else {
					a.voting.controlResult = "Session started."
					a.loadVoting()
				}
			}
			if secondaryButton(c, "End Session").Clicked() {
				if err := a.services.EndSession(strings.TrimSpace(a.voting.controlID)); err != nil {
					a.voting.controlResult = err.Error()
				} else {
					a.voting.controlResult = "Session ended."
					a.loadVoting()
				}
			}
		})
		if a.voting.controlResult != "" {
			ui.Text(c, a.voting.controlResult).FontSize(12).TextColor(ui.Hex(inkMuted))
		}
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
		if a.voting.voteResult != "" {
			ui.Text(c, a.voting.voteResult).FontSize(12).TextColor(ui.Hex(inkMuted))
		}
	})
}

func (a *App) castVote() {
	a.voting.voteResult = ""
	if strings.TrimSpace(a.voting.voteSession) == "" || strings.TrimSpace(a.voting.voteCandidate) == "" {
		a.voting.voteResult = "Choose a session and a candidate."
		return
	}
	err := a.services.CastVote(
		strings.TrimSpace(a.voting.voteSession),
		strings.TrimSpace(a.voting.voteCandidate),
		strings.TrimSpace(a.voting.voteVoterPub),
		strings.TrimSpace(a.voting.votePriv),
	)
	if err != nil {
		a.voting.voteResult = err.Error()
		return
	}
	a.voting.voteResult = "Ballot recorded."
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

// loadSessionResults reads a session's tally into the Results card.
func (a *App) loadSessionResults(sessionID string) error {
	rows, err := a.services.SessionResults(sessionID)
	if err != nil {
		return err
	}
	a.voting.results = rows
	return nil
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
			if err := a.loadSessionResults(strings.TrimSpace(a.voting.resultsSession)); err != nil {
				a.voting.results = nil
				a.voting.status = errStatus(err)
			} else {
				a.voting.status = statusLine{}
			}
		}
	})
	if len(a.voting.results) == 0 {
		emptyNote(c, "No tally loaded yet.")
		return
	}
	results := a.voting.results
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
