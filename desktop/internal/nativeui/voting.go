package nativeui

import (
	"errors"
	"time"
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

	sessionTitle    string
	sessionDesc     string
	sessionStart    time.Time
	sessionEnd      time.Time
	sessionIDs      string
	sessionStartErr string
	sessionEndErr   string
	sessionResult   statusLine
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
	// voteSubjectPub/voteSubjectKnown track the voter whose signing context the
	// ballot form holds. A public key typed directly into the field bypasses
	// the voter row and registration paths, so the previous voter's private
	// key and verdict must be dropped when the visible identity changes.
	voteSubjectPub   string
	voteSubjectKnown bool

	resultsSession string
	results        []ResultRow
	// resultsLoadedID is the session the readout describes, including a failed
	// empty read. resultsSession is the form field and can name another session
	// between reads; keeping them apart is what lets a failed read report
	// "stale" for the session it actually loaded, and never pin one session's
	// tally to another's id.
	resultsLoadedID string
	// resultsErr is the last failure to read that tally. The rows from the
	// previous successful read stay when they still describe the same session
	// and are reported as stale; with no rows the card explains the failure
	// instead of claiming no tally exists.
	resultsErr error

	candidates []CandidateRow
	sessions   []SessionRow
	voters     []VoterRow
	// offsets page the three collections. Each loader requests one extra row so
	// Next is disabled on the exact final page instead of opening an empty one.
	voterOffset    int
	voterHasMore   bool
	candOffset     int
	candHasMore    bool
	sessionOffset  int
	sessionHasMore bool
	voterLoadErr   error
	candLoadErr    error
	sessionLoadErr error
	// voterSearch, candSearch and sessionSearch narrow those lists as they grow;
	// all filters are case-insensitive and operate on the current page.
	voterSearch   string
	candSearch    string
	sessionSearch string
	// voterApplied/candApplied/sessionApplied are the queries the highlights
	// belong to.
	voterApplied   string
	candApplied    string
	sessionApplied string
	// rosterSessionID identifies the successful name lookup behind
	// rosterNames. A session stores only candidate ids, and its roster can
	// reach beyond the visible candidate page.
	rosterSessionID string
	rosterNames     map[string]string
	rosterErr       error

	voterNameErr string
	candNameErr  string
	loadErr      error

	// tab is the stage shown: 0 setup (register/create), 1 ballot (controls and
	// cast), 2 results. The page used to stack all four areas at once, so a
	// first-time operator had to find the order in the sidebar of cards.
	tab int

	status statusLine

	// sessionChoice/candChoice/voterChoice are the rows chosen in those tables,
	// -1 for none. The tables shorten ids to fit, so choosing a row is the only
	// way the full value reaches the forms.
	sessionChoice int
	candChoice    int
	voterChoice   int
}

const votingPageSize = 100

func (a *App) loadVoting() {
	if a.services == nil {
		return
	}
	candID := a.selectedCandidateID()
	sessionID := a.selectedSessionID()
	voterKey := a.selectedVoterPublicKey()
	a.loadVoterPage(a.voting.voterOffset)
	a.loadCandidatePage(a.voting.candOffset)
	a.loadSessionPage(a.voting.sessionOffset)
	a.restoreVoterChoice(voterKey)
	a.restoreCandidateChoice(candID)
	a.restoreSessionChoice(sessionID)
	a.loadSelectedSessionRoster()
}

func (a *App) loadVoterPage(offset int) bool {
	if a.services == nil {
		return false
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.services.Voters(votingPageSize+1, offset)
	a.voting.voterLoadErr = err
	a.syncVotingLoadErr()
	if err != nil {
		return false
	}
	if len(rows) == 0 && offset > 0 {
		return a.loadVoterPage(0)
	}
	a.voting.voterOffset = offset
	a.voting.voterHasMore = len(rows) > votingPageSize
	if a.voting.voterHasMore {
		rows = rows[:votingPageSize]
	}
	a.voting.voters = rows
	return true
}

// loadCandidatePage commits a page offset only after its read succeeds. A
// failed pager move therefore leaves the rows, offset and highlight on the page
// they actually describe.
func (a *App) loadCandidatePage(offset int) bool {
	if a.services == nil {
		return false
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.services.Candidates(votingPageSize+1, offset)
	a.voting.candLoadErr = err
	a.syncVotingLoadErr()
	if err != nil {
		return false
	}
	if len(rows) == 0 && offset > 0 {
		return a.loadCandidatePage(0)
	}
	a.voting.candOffset = offset
	a.voting.candHasMore = len(rows) > votingPageSize
	if a.voting.candHasMore {
		rows = rows[:votingPageSize]
	}
	a.voting.candidates = rows
	return true
}

func (a *App) loadSessionPage(offset int) bool {
	if a.services == nil {
		return false
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.services.Sessions(votingPageSize+1, offset)
	a.voting.sessionLoadErr = err
	a.syncVotingLoadErr()
	if err != nil {
		return false
	}
	if len(rows) == 0 && offset > 0 {
		return a.loadSessionPage(0)
	}
	a.voting.sessionOffset = offset
	a.voting.sessionHasMore = len(rows) > votingPageSize
	if a.voting.sessionHasMore {
		rows = rows[:votingPageSize]
	}
	a.voting.sessions = rows
	return true
}

func (a *App) syncVotingLoadErr() {
	a.voting.loadErr = errors.Join(a.voting.voterLoadErr, a.voting.candLoadErr, a.voting.sessionLoadErr)
}

func (a *App) selectedVoterPublicKey() string {
	rows := a.filteredVoters()
	if a.voting.voterChoice < 0 || a.voting.voterChoice >= len(rows) {
		return ""
	}
	return rows[a.voting.voterChoice].PublicKey
}

func (a *App) restoreVoterChoice(publicKey string) {
	a.voting.voterChoice = -1
	if publicKey == "" {
		return
	}
	for i, voter := range a.filteredVoters() {
		if voter.PublicKey == publicKey {
			a.voting.voterChoice = i
			return
		}
	}
}

// selectedCandidateID resolves the current highlight through the filtered
// slice the table actually uses. The loaders restore the id rather than the
// index so a refresh cannot make the old row number select a different person.
func (a *App) selectedCandidateID() string {
	rows := a.filteredCandidates()
	if a.voting.candChoice < 0 || a.voting.candChoice >= len(rows) {
		return ""
	}
	return rows[a.voting.candChoice].ID
}

func (a *App) restoreCandidateChoice(id string) {
	a.voting.candChoice = -1
	if id == "" {
		return
	}
	for i, candidate := range a.filteredCandidates() {
		if candidate.ID == id {
			a.voting.candChoice = i
			return
		}
	}
}

func (a *App) selectedSessionID() string {
	rows := a.filteredSessions()
	if a.voting.sessionChoice < 0 || a.voting.sessionChoice >= len(rows) {
		return ""
	}
	return rows[a.voting.sessionChoice].ID
}

func (a *App) restoreSessionChoice(id string) {
	a.voting.sessionChoice = -1
	if id == "" {
		return
	}
	for i, session := range a.filteredSessions() {
		if session.ID == id {
			a.voting.sessionChoice = i
			return
		}
	}
}

func (a *App) selectedSession() (SessionRow, bool) {
	rows := a.filteredSessions()
	if a.voting.sessionChoice < 0 || a.voting.sessionChoice >= len(rows) {
		return SessionRow{}, false
	}
	return rows[a.voting.sessionChoice], true
}
