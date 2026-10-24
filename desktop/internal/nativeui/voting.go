package nativeui

import (
	"strconv"
	"strings"

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

func (a *App) votingView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			a.votingRegister(c)
			a.votingSessions(c)
			a.votingVote(c)
			a.votingResults(c)
		})
	})
}

func (a *App) votingRegister(c *ui.Context) {
	sectionTitle(c, "Register Voter", "Returns the key pair the voter signs ballots with; keep the private half.")
	field(c, "Voter Name", &a.voting.voterName)
	if a.voting.voterNameErr != "" {
		ui.Text(c, a.voting.voterNameErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	if ui.Button(c, "Register Voter").Clicked() {
		a.voting.voterNameErr = ""
		a.voting.status = statusLine{}
		if strings.TrimSpace(a.voting.voterName) == "" {
			a.voting.voterNameErr = "Voter name is required."
		} else if v, err := a.services.RegisterVoter(strings.TrimSpace(a.voting.voterName)); err != nil {
			a.voting.status = errStatus(err)
		} else {
			a.voting.status = okStatus("Registered %s (public key %s)", v.Name, v.PublicKey)
			a.voting.voteVoterPub = v.PublicKey
			a.voting.voterName = ""
		}
	}
	renderStatus(c, a.voting.status)

	ui.Divider(c)
	sectionTitle(c, "Register Candidate", "Only the name is required; party and program are shown on the ballot when present.")
	field(c, "Candidate Name", &a.voting.candName)
	if a.voting.candNameErr != "" {
		ui.Text(c, a.voting.candNameErr).FontSize(11).TextColor(ui.Hex("#c62828"))
	}
	field(c, "Party (optional)", &a.voting.candParty)
	field(c, "Program (optional)", &a.voting.candProgram)
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
	if a.voting.candResult != "" {
		ui.Text(c, a.voting.candResult).FontSize(12)
	}
}

func (a *App) votingSessions(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Create Session", "Choose the roster and the window during which ballots are accepted.")
	field(c, "Title", &a.voting.sessionTitle)
	field(c, "Description (optional)", &a.voting.sessionDesc)
	fieldHint(c, "Start (unix seconds)", "Ballots are rejected before this time.", &a.voting.sessionStart)
	fieldHint(c, "End (unix seconds)", "Ballots are rejected after this time.", &a.voting.sessionEnd)
	fieldHint(c, "Candidate IDs (comma-separated)", "Leave empty to include every registered candidate.", &a.voting.sessionIDs)
	if ui.PrimaryButton(c, "Create Session").Clicked() {
		a.createVotingSession()
	}
	if a.voting.sessionResult != "" {
		ui.Text(c, a.voting.sessionResult).FontSize(12)
	}

	ui.Divider(c)
	sectionTitle(c, "Sessions", "Every governance window with its status.")
	if len(a.voting.sessions) == 0 {
		emptyNote(c, "No sessions yet.")
		return
	}
	sessions := a.voting.sessions
	ui.Table(c, &a.voting.sessionTable, []ui.TableColumn{
		{Title: "ID", Width: 140},
		{Title: "Title"},
		{Title: "Status", Width: 100},
	}, len(sessions), func(row, col int) {
		s := sessions[row]
		switch col {
		case 0:
			ui.Text(c, s.ID).SingleLine()
		case 1:
			ui.Text(c, s.Title)
		case 2:
			ui.Text(c, s.Status)
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
	ui.Divider(c)
	sectionTitle(c, "Session Controls", "Start or end a session by id.")
	field(c, "Session ID", &a.voting.controlID)
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
	if a.voting.controlResult != "" {
		ui.Text(c, a.voting.controlResult).FontSize(12)
	}

	ui.Divider(c)
	sectionTitle(c, "Cast Vote", "Sign a ballot for one candidate in one session.")
	field(c, "Session ID", &a.voting.voteSession)
	field(c, "Candidate ID", &a.voting.voteCandidate)
	field(c, "Voter Public Key", &a.voting.voteVoterPub)
	field(c, "Voter Private Key", &a.voting.votePriv)
	if ui.PrimaryButton(c, "Cast Vote").Clicked() {
		a.castVote()
	}
	if a.voting.voteResult != "" {
		ui.Text(c, a.voting.voteResult).FontSize(12)
	}
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

func (a *App) votingResults(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Results", "Live tally per candidate for a session.")
	field(c, "Session ID", &a.voting.resultsSession)
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
	renderStatus(c, a.voting.status)
	if len(a.voting.results) == 0 {
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
