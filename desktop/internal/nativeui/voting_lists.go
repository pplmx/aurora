package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// loadSelectedSessionRoster resolves the names that a selected session's
// stored candidate ids refer to. Candidate lists are paged, so looking only at
// the visible rows turned older roster entries into opaque shortened ids.
func (a *App) loadSelectedSessionRoster() {
	session, ok := a.selectedSession()
	if !ok {
		a.clearSessionRoster()
		return
	}
	a.loadSessionRoster(session.ID, session.Candidates)
}

func (a *App) loadSessionRoster(sessionID string, ids []string) {
	if a.services == nil {
		return
	}
	if a.voting.rosterSessionID != sessionID {
		// Never leave the previous session's names under a new roster when the
		// fresh read fails.
		a.voting.rosterSessionID = ""
		a.voting.rosterNames = nil
		a.voting.rosterErr = nil
	}
	names, err := a.services.CandidateNames(ids)
	if err != nil {
		a.voting.rosterErr = err
		return
	}
	a.voting.rosterSessionID = sessionID
	a.voting.rosterNames = names
	a.voting.rosterErr = nil
}

func (a *App) clearSessionRoster() {
	a.voting.rosterSessionID = ""
	a.voting.rosterNames = nil
	a.voting.rosterErr = nil
}

// votingVoterListCard restores the registered-voter view the desktop lacked:
// after registration the only visible public key was the one-time key card,
// so a restart hid every voter that could still cast a ballot.
func (a *App) votingVoterListCard(c *ui.Context) {
	formCard(c, "Voters", "Registered signing keys and whether each voter has ever cast a ballot.", func() {
		a.votingVoterList(c)
	})
}

func (a *App) votingVoterList(c *ui.Context) {
	if len(a.voting.voters) == 0 {
		if a.voting.loadErr == nil {
			emptyStateIcon(c, usersIcon, pageAccent(c, pageVoting), "No voters registered yet.",
				"Register a voter in Register Voter; that issues the signing key the first ballot needs.")
		}
		return
	}
	queryChanged := a.voting.voterSearch != a.voting.voterApplied
	if queryChanged {
		a.voting.voterApplied = a.voting.voterSearch
		a.voting.voterChoice = -1
	}
	previous, next := collectionToolbar(c, &a.voting.voterSearch, "Search voters",
		"Search narrows the current page; use Previous/Next for older voters.",
		"Voters", a.voting.voterOffset, len(a.voting.voters), a.voting.voterHasMore)
	if previous || next {
		target := a.voting.voterOffset
		if previous {
			target = max(0, target-votingPageSize)
		} else {
			target += votingPageSize
		}
		if a.loadVoterPage(target) {
			a.voting.voterChoice = -1
		}
		return
	}
	voters := a.filteredVoters()
	announceSearchResults(c, queryChanged, len(voters))
	if len(voters) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No voters match “%s”."), strings.TrimSpace(a.voting.voterSearch)))
		return
	}
	collectionList(c, "Voters",
		collectionRecords(voters, func(voter VoterRow) collectionRecord {
			return voterCollectionRecord(c, voter)
		}),
		a.voting.voterChoice, func(i int) { a.chooseVoter(voters, i) })
	hint(c, "A prior ballot does not block a later session; each session accepts one ballot per voter.")
	if a.voting.voterChoice >= 0 && a.voting.voterChoice < len(voters) {
		copyRow(c, "Selected voter public key", voters[a.voting.voterChoice].PublicKey, nil)
	}
}

// voterCollectionRecord projects a registered voter onto a record. "Ever
// voted" is a fact with a verdict hue because it is the one state an operator
// scans the list for; the public key is the field the choice copies.
func voterCollectionRecord(c *ui.Context, voter VoterRow) collectionRecord {
	name := strings.TrimSpace(voter.Name)
	if name == "" {
		name = shortID(voter.PublicKey)
	}
	status, tone := tr("No"), ui.Color{}
	if voter.HasVoted {
		status, tone = tr("Yes"), c.Theme().Success
	}
	return collectionRecord{
		title:  name,
		facts:  []collectionFact{{label: "Ever voted", value: status, tone: tone}},
		fields: []collectionField{{label: "Public key", value: voter.PublicKey}},
	}
}

// chooseVoter is shared by the table and large-text records so either selection
// loads the same complete public key into the ballot.
func (a *App) chooseVoter(voters []VoterRow, index int) {
	if index < 0 || index >= len(voters) {
		return
	}
	a.voting.voterChoice = index
	a.applyVoteVoter(voters[index].PublicKey)
}

func (a *App) filteredVoters() []VoterRow {
	rows := make([]VoterRow, 0, len(a.voting.voters))
	for _, voter := range a.voting.voters {
		if containsFold(a.voting.voterSearch, voter.Name, voter.PublicKey) {
			rows = append(rows, voter)
		}
	}
	return rows
}

func (a *App) votingSessionListCard(c *ui.Context) {
	formCard(c, "Sessions", "Every governance window, its status and when ballots are accepted.", func() {
		a.votingSessionList(c)
	})
}

func (a *App) votingSessionList(c *ui.Context) {
	if len(a.voting.sessions) == 0 {
		// A failed load is reported once, above the cards; with no rows there
		// is nothing to keep, and "No sessions yet" would claim the store is
		// empty when it could not be read.
		if a.voting.loadErr == nil {
			emptyStateIcon(c, calendarIcon, pageAccent(c, pageVoting), "No sessions yet.",
				"Create one in Create Session with a roster and the window during which ballots count.")
		}
		return
	}
	queryChanged := a.voting.sessionSearch != a.voting.sessionApplied
	if queryChanged {
		a.voting.sessionApplied = a.voting.sessionSearch
		a.voting.sessionChoice = -1
		a.clearSessionRoster()
	}
	previous, next := collectionToolbar(c, &a.voting.sessionSearch, "Search sessions",
		"Search narrows the current page; use Previous/Next for older sessions.",
		"Sessions", a.voting.sessionOffset, len(a.voting.sessions), a.voting.sessionHasMore)
	if previous || next {
		target := a.voting.sessionOffset
		if previous {
			target = max(0, target-votingPageSize)
		} else {
			target += votingPageSize
		}
		if a.loadSessionPage(target) {
			a.voting.sessionChoice = -1
			a.clearSessionRoster()
		}
		return
	}
	sessions := a.filteredSessions()
	announceSearchResults(c, queryChanged, len(sessions))
	if len(sessions) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No sessions match “%s”."), strings.TrimSpace(a.voting.sessionSearch)))
		return
	}
	collectionList(c, "Sessions",
		collectionRecords(sessions, sessionCollectionRecord),
		a.voting.sessionChoice, func(i int) { a.chooseSession(sessions, i) })
	if a.voting.sessionChoice >= 0 && a.voting.sessionChoice < len(sessions) {
		// The row cannot show the roster (the window column already fills the
		// width), but an operator about to start or end a session needs to see
		// who is on the ballot and what the session is for.
		chosen := sessions[a.voting.sessionChoice]
		if a.voting.rosterErr != nil {
			_ = loadWarning(c, "Couldn't load session roster", a.voting.rosterErr,
				a.voting.rosterSessionID == chosen.ID)
		}
		readout(c, []readoutRow{
			{"Title", chosen.Title},
			{"About", valueOr(chosen.Description)},
			{"Window", sessionWindow(chosen.StartTime, chosen.EndTime)},
			{"Roster", a.sessionRosterNames(chosen.ID, chosen.Candidates)},
		})
		copyRow(c, "Selected session ID", chosen.ID, nil)
	}
}

// sessionCollectionRecord projects a governance session onto a record. The
// window is a field because it is the reason a ballot is refused, and it
// belongs in the list rather than only in the error a voter sees afterwards.
func sessionCollectionRecord(session SessionRow) collectionRecord {
	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = shortID(session.ID)
	}
	return collectionRecord{
		title: title,
		facts: []collectionFact{
			{label: "Status", value: valueOr(tr(session.Status))},
		},
		fields: []collectionField{
			{label: "Window", value: sessionWindow(session.StartTime, session.EndTime)},
			{label: "ID", value: session.ID},
		},
	}
}

// chooseSession is shared by the table and record presentations. A shortened
// row id becomes the full id for controls, ballot and tally at once, and any
// already-open result is invalidated before it can describe another session.
func (a *App) chooseSession(sessions []SessionRow, index int) {
	if index < 0 || index >= len(sessions) {
		return
	}
	a.voting.sessionChoice = index
	id := sessions[index].ID
	a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = id, id, id
	if len(a.voting.results) > 0 || a.voting.resultsLoadedID != "" || a.voting.resultsErr != nil {
		// The tally on screen belongs to the previously chosen session; a
		// selection must not leave it beside a new session id even when the
		// fresh read fails. Same-session refreshes keep their rows.
		a.clearTallyForAnotherSession(id)
		_ = a.loadSessionResults(id)
	}
	a.loadSelectedSessionRoster()
}

func (a *App) filteredSessions() []SessionRow {
	rows := make([]SessionRow, 0, len(a.voting.sessions))
	for _, session := range a.voting.sessions {
		if containsFold(a.voting.sessionSearch, session.ID, session.Title, session.Status) {
			rows = append(rows, session)
		}
	}
	return rows
}

// sessionRosterNames renders a session's candidate ids as names, falling back
// to the shortened id when a candidate has been deleted.
func (a *App) sessionRosterNames(sessionID string, ids []string) string {
	if len(ids) == 0 {
		return "—"
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := ""
		if a.voting.rosterSessionID == sessionID {
			name = a.voting.rosterNames[id]
		}
		if name == "" {
			for _, candidate := range a.voting.candidates {
				if candidate.ID == id {
					name = candidate.Name
					break
				}
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
			emptyStateIcon(c, userCheckIcon, pageAccent(c, pageVoting), "No candidates registered yet.",
				"Register one in Register Candidate; a ballot names the id it selects.")
		}
		return
	}
	queryChanged := a.voting.candSearch != a.voting.candApplied
	if queryChanged {
		a.voting.candApplied = a.voting.candSearch
		a.voting.candChoice = -1
	}
	previous, next := collectionToolbar(c, &a.voting.candSearch, "Search candidates",
		"Search narrows the current page; use Previous/Next for older candidates.",
		"Candidates", a.voting.candOffset, len(a.voting.candidates), a.voting.candHasMore)
	if previous || next {
		target := a.voting.candOffset
		if previous {
			target = max(0, target-votingPageSize)
		} else {
			target += votingPageSize
		}
		if a.loadCandidatePage(target) {
			a.voting.candChoice = -1
		}
		return
	}
	candidates := a.filteredCandidates()
	announceSearchResults(c, queryChanged, len(candidates))
	if len(candidates) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No candidates match “%s”."), strings.TrimSpace(a.voting.candSearch)))
		return
	}
	collectionList(c, "Candidates",
		collectionRecords(candidates, candidateCollectionRecord),
		a.voting.candChoice, func(i int) { a.chooseCandidate(candidates, i) })
	rosterReady := a.voting.candChoice >= 0 && a.voting.candChoice < len(candidates)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Add to session roster").Disabled(!rosterReady).Clicked() {
			a.addCandidateToRoster()
		}
	})
	if rosterReady {
		copyRow(c, "Selected candidate ID", candidates[a.voting.candChoice].ID, nil)
	}
}

// candidateCollectionRecord projects a registered candidate onto a record. The
// program is a field because it is the part of a registration the ballot shows,
// and votes are a fact because they are the number the list is read for.
func candidateCollectionRecord(candidate CandidateRow) collectionRecord {
	name := strings.TrimSpace(candidate.Name)
	if name == "" {
		name = shortID(candidate.ID)
	}
	record := collectionRecord{
		title: name,
		facts: []collectionFact{
			{label: "Party", value: valueOr(candidate.Party)},
		},
		fields: []collectionField{{label: "ID", value: candidate.ID}},
	}
	if candidate.VoteCount > 0 {
		record.facts = append(record.facts,
			collectionFact{label: "Votes", value: strconv.Itoa(candidate.VoteCount)})
	}
	if program := strings.TrimSpace(candidate.Program); program != "" {
		record.fields = append([]collectionField{{label: "Program", value: program}}, record.fields...)
	}
	return record
}

// chooseCandidate is the one selection path for filling the ballot with a
// candidate id, whether the choice came from a table or a record.
func (a *App) chooseCandidate(candidates []CandidateRow, index int) {
	if index < 0 || index >= len(candidates) {
		return
	}
	a.voting.candChoice = index
	a.voting.voteCandidate = candidates[index].ID
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
