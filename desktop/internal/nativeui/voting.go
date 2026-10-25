package nativeui

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// votingState is the Voting page's state.
type votingState struct {
	voterName   string
	voterResult string
	voterPriv   string

	candName    string
	candParty   string
	candProgram string
	candResult  string

	sessionTitle  string
	sessionDesc   string
	sessionStart  string
	sessionEnd    string
	sessionIDs    string
	sessionResult string

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

	sessionTable ui.ListState
	resultsTable ui.ListState
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
		renderStatus(c, a.voting.status)
		// The session list leads: an operator reads what exists before
		// registering more of it, and it was the one card pushed below the
		// fold by three registration forms.
		twoColumns(c, func() {
			a.votingSessionListCard(c)
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
			if ui.PrimaryButton(c, "Register Voter").Clicked() {
				a.registerVoter()
			}
		})
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
		if ui.Button(c, "Register Candidate").Clicked() {
			a.voting.candNameErr = ""
			a.voting.candResult = ""
			if strings.TrimSpace(a.voting.candName) == "" {
				a.voting.candNameErr = "Candidate name is required."
			} else if cand, err := a.services.RegisterCandidate(
				strings.TrimSpace(a.voting.candName),
				strings.TrimSpace(a.voting.candParty),
				strings.TrimSpace(a.voting.candProgram),
			); err != nil {
				a.voting.candResult = err.Error()
			} else {
				a.voting.candResult = "Registered " + cand.Name + " (" + cand.ID + ")"
				a.voting.candName, a.voting.candParty, a.voting.candProgram = "", "", ""
				a.loadVoting()
			}
		}
	})
	if a.voting.candResult != "" {
		ui.Text(c, a.voting.candResult).FontSize(12).TextColor(ui.Hex(inkMuted))
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
// operator. Filling the ballot fields means the voter can sign straight away
// instead of copying a key that is no longer readable anywhere.
func (a *App) applyVoterRegistration(v VoterRow) {
	a.voting.status = okStatus(
		"Registered %s (public key %s). Ballot signing key: %s",
		v.Name, v.PublicKey, v.PrivateKey)
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
	fieldHint(c, "Start (unix seconds)", "Ballots are rejected before this time.", &a.voting.sessionStart)
	fieldHint(c, "End (unix seconds)", "Ballots are rejected after this time.", &a.voting.sessionEnd)
	fieldHint(c, "Candidate IDs (comma-separated)", "Leave empty to include every registered candidate.", &a.voting.sessionIDs)
	ui.Row(c).Children(func() {
		if ui.PrimaryButton(c, "Create Session").Clicked() {
			a.createVotingSession()
		}
	})
	if a.voting.sessionResult != "" {
		ui.Text(c, a.voting.sessionResult).FontSize(12).TextColor(ui.Hex(inkMuted))
	}
}

func (a *App) votingSessionList(c *ui.Context) {
	if len(a.voting.sessions) == 0 {
		emptyNote(c, "No sessions yet.")
		return
	}
	sessions := a.voting.sessions
	ui.Table(c, &a.voting.sessionTable, []ui.TableColumn{
		{Title: "ID", Width: 140},
		{Title: "Title", MinWidth: 110},
		{Title: "Status", Width: 100},
		{Title: "Window", Width: 200},
	}, len(sessions), func(row, col int) {
		s := sessions[row]
		switch col {
		case 0:
			ui.Text(c, shortID(s.ID)).SingleLine()
		case 1:
			ui.Text(c, s.Title)
		case 2:
			ui.Text(c, s.Status)
		case 3:
			// The window is why a ballot is refused, so it belongs in the
			// list rather than only in the error a voter sees afterwards.
			ui.Text(c, sessionWindow(s.StartTime, s.EndTime)).FontSize(12)
		}
	})
}

func (a *App) createVotingSession() {
	a.voting.sessionResult = ""
	start, err := strconv.ParseInt(strings.TrimSpace(a.voting.sessionStart), 10, 64)
	if err != nil {
		a.voting.sessionResult = "Start time must be a unix timestamp in seconds."
		return
	}
	end, err := strconv.ParseInt(strings.TrimSpace(a.voting.sessionEnd), 10, 64)
	if err != nil {
		a.voting.sessionResult = "End time must be a unix timestamp in seconds."
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
	a.voting.sessionResult = "Created session " + s.ID
	a.voting.sessionTitle, a.voting.sessionDesc, a.voting.sessionStart, a.voting.sessionEnd, a.voting.sessionIDs = "", "", "", "", ""
	a.loadVoting()
}

func (a *App) votingVote(c *ui.Context) {
	formCard(c, "Session Controls", "Start or end a session by id.", func() {
		field(c, "Session ID", &a.voting.controlID)
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Start Session").Clicked() {
				if err := a.services.StartSession(strings.TrimSpace(a.voting.controlID)); err != nil {
					a.voting.controlResult = err.Error()
				} else {
					a.voting.controlResult = "Session started."
					a.loadVoting()
				}
			}
			if ui.Button(c, "End Session").Clicked() {
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
			if ui.PrimaryButton(c, "Cast Vote").Clicked() {
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
		if ui.Button(c, "Show Results").Clicked() {
			rows, err := a.services.SessionResults(strings.TrimSpace(a.voting.resultsSession))
			if err != nil {
				a.voting.results = nil
				a.voting.status = errStatus(err)
			} else {
				a.voting.results = rows
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
			ui.Text(c, r.Name)
		} else {
			ui.Text(c, strconv.Itoa(r.Votes))
		}
	})
}
