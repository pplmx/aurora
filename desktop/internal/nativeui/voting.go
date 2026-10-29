package nativeui

import (
	"errors"
	"fmt"
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

// votingView splits the page into the three stages an operator moves through:
// setup (register voters/candidates, create a session), ballot (start/end a
// session, cast a signed ballot), and results. The four areas used to be
// stacked at once, so a first-time operator had no visible order.
func (a *App) votingView(c *ui.Context) {
	a.syncVoteSubject()
	a.scrollPage(c, func() {
		// loadVoting keeps both lists when either read fails, so the warning
		// is drawn once for the page and both lists keep their last successful
		// rows. It used to be invisible entirely: nothing rendered loadErr.
		if a.voting.loadErr != nil {
			loadWarning(c, "Failed to load voting data", a.voting.loadErr,
				len(a.voting.voters) > 0 || len(a.voting.sessions) > 0 || len(a.voting.candidates) > 0)
		}
		guide(c, &a.guides.voting,
			"Register voters and candidates in Setup. A voter registration returns the only copy of that voter's signing key.",
			"Create a session with a roster and a start/end window.",
			"Open Ballot to start the session and cast signed ballots.",
			"Open Results and select or enter the session to read the tally.",
		)
		tabs(c, &a.voting.tab, "Voting sections", tr("Setup"), tr("Ballot"), tr("Results"))
		switch a.voting.tab {
		case 0:
			twoColumns(c, func() {
				a.votingVoterListCard(c)
				a.votingCandidateListCard(c)
				a.votingSessionListCard(c)
			}, func() {
				a.votingRegister(c)
				a.votingSessions(c)
			})
		case 1:
			a.votingVote(c)
		default:
			if id := strings.TrimSpace(a.voting.resultsSession); id != "" && a.voting.resultsLoadedID != id {
				// Opening Results with a session already chosen should show its
				// tally, not require another Show Results click. A failed or
				// empty load records the id, so it does not repeat every frame.
				a.clearTallyForAnotherSession(id)
				_ = a.loadSessionResults(id)
			}
			a.votingResults(c)
		}
	})
}

func (a *App) votingRegister(c *ui.Context) {
	formCard(c, "Register Voter", "Returns the key pair the voter signs ballots with; keep the private half.", func() {
		validatedField(c, "Voter Name", &a.voting.voterName, &a.voting.voterNameErr)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Register Voter").Disabled(!allSet(a.voting.voterName)).Clicked() {
				a.registerVoter()
			}
		})
		if a.voting.voterPriv != "" {
			ui.Text(c, tr("New voter key pair")).FontSize(textSize(c, 12)).Bold().TextColor(c.Theme().Text)
			copyRow(c, "Public key", a.voting.voterPub, func() {
				a.voting.status = okStatus("Public key copied.")
			})
			copyRow(c, "Private key", a.voting.voterPriv, func() {
				a.voting.status = okStatus("Private key copied. The store keeps only the public half; keep it safe.")
			})
			ui.Text(c, tr("The store keeps only the public half, so this is the only copy of the signing key.")).
				FontSize(textSize(c, 11)).TextColor(ui.Hex(auroraAmber))
		}
	})

	formCard(c, "Register Candidate", "Only the name is required; party and program are shown on the ballot when present.", func() {
		a.votingRegisterCandidateBody(c)
	})
}

func (a *App) votingRegisterCandidateBody(c *ui.Context) {
	validatedField(c, "Candidate Name", &a.voting.candName, &a.voting.candNameErr)
	fieldPair(c, "Party (optional)", &a.voting.candParty, "Program (optional)", &a.voting.candProgram)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Register Candidate").Disabled(!allSet(a.voting.candName)).Clicked() {
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
	party, program := strings.TrimSpace(a.voting.candParty), strings.TrimSpace(a.voting.candProgram)
	a.runWrite("Registering candidate…", func() func() {
		cand, err := a.services.RegisterCandidate(name, party, program)
		return func() {
			if err != nil {
				a.voting.candResult = errStatus(err)
				return
			}
			a.voting.candResult = okStatus("Registered %s", cand.Name)
			a.voting.candCopyID = cand.ID
			a.voting.candName, a.voting.candParty, a.voting.candProgram = "", "", ""
			a.voting.candChoice = -1
			a.loadCandidatePage(0)
			// The new candidate is the context: highlight it, clear a search that
			// would hide it, and aim the ballot at it. The operator can still change
			// either choice before casting.
			a.selectCandidateByID(cand.ID)
			a.voting.voteCandidate = cand.ID
		}
	})
}

// selectCandidateByID highlights a candidate after registration.
func (a *App) selectCandidateByID(id string) {
	a.voting.candSearch, a.voting.candApplied = "", ""
	for i, candidate := range a.filteredCandidates() {
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
	a.runWrite("Registering voter…", func() func() {
		v, err := a.services.RegisterVoter(name)
		return func() {
			if err != nil {
				a.voting.status = errStatus(err)
				return
			}
			a.applyVoterRegistration(v)
		}
	})
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
	a.rememberSuccessfulKey(v.PublicKey, v.PrivateKey)
	a.voting.voterPub, a.voting.voterPriv = v.PublicKey, v.PrivateKey
	a.voting.status = okStatus("Registered %s. The ballot fields are filled from the new key pair.", v.Name)
	a.applyVoteVoter(v.PublicKey)
	a.voting.votePriv = v.PrivateKey
	a.voting.voterName = ""
	a.voting.voterSearch, a.voting.voterApplied = "", ""
	a.voting.voterChoice = -1
	if a.loadVoterPage(0) {
		a.selectVoterByPublicKey(v.PublicKey)
	}
}

func (a *App) selectVoterByPublicKey(publicKey string) {
	a.voting.voterSearch, a.voting.voterApplied = "", ""
	for i, voter := range a.filteredVoters() {
		if voter.PublicKey == publicKey {
			a.voting.voterChoice = i
			return
		}
	}
}

// applyVoteVoter moves the ballot identity and clears signing state that
// belonged to the previous voter. The private key is set separately by the
// registration path because that is the one response allowed to supply it.
func (a *App) applyVoteVoter(publicKey string) {
	publicKey = strings.TrimSpace(publicKey)
	if a.voting.voteVoterPub != publicKey {
		a.voting.votePriv = ""
		a.voting.voteResult = statusLine{}
	}
	a.voting.voteVoterPub = publicKey
	a.voting.voteSubjectPub = publicKey
	a.voting.voteSubjectKnown = true
}

// syncVoteSubject reacts to the public-key field being edited by hand. The
// first frame adopts state restored before the page was opened; after that,
// changing the visible voter invalidates the old private key and verdict.
func (a *App) syncVoteSubject() {
	current := strings.TrimSpace(a.voting.voteVoterPub)
	if !a.voting.voteSubjectKnown {
		a.voting.voteSubjectPub = current
		a.voting.voteSubjectKnown = true
		return
	}
	if current == a.voting.voteSubjectPub {
		return
	}
	a.voting.votePriv = ""
	a.voting.voteResult = statusLine{}
	a.voting.voteSubjectPub = current
}

func (a *App) votingSessions(c *ui.Context) {
	formCard(c, "Create Session", "Choose the roster and the window during which ballots are accepted.", func() {
		a.votingCreateSessionBody(c)
	})
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
			emptyStateIcon(c, usersIcon, ui.Hex(votingAccent), "No voters registered yet.",
				"Register a voter on the right to issue the signing key the first ballot needs.")
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
		collectionRecords(voters, voterCollectionRecord),
		a.voting.voterChoice, func(i int) { a.chooseVoter(voters, i) })
	hint(c, "A prior ballot does not block a later session; each session accepts one ballot per voter.")
	if a.voting.voterChoice >= 0 && a.voting.voterChoice < len(voters) {
		copyRow(c, "Selected voter public key", voters[a.voting.voterChoice].PublicKey, nil)
	}
}

// voterCollectionRecord projects a registered voter onto a record. "Ever
// voted" is a fact with a verdict hue because it is the one state an operator
// scans the list for; the public key is the field the choice copies.
func voterCollectionRecord(voter VoterRow) collectionRecord {
	name := strings.TrimSpace(voter.Name)
	if name == "" {
		name = shortID(voter.PublicKey)
	}
	status, tone := tr("No"), ui.Color{}
	if voter.HasVoted {
		status, tone = tr("Yes"), ui.Hex(auroraGreen)
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

func (a *App) votingCreateSessionBody(c *ui.Context) {
	field(c, "Title", &a.voting.sessionTitle)
	field(c, "Description (optional)", &a.voting.sessionDesc)
	// The window is edited with the toolkit's own date and time controls:
	// asking an operator to type epoch seconds made the card unusable without
	// a clock and a converter outside the app. The two controls edit one
	// time.Time, shown in local time; the service still receives Unix seconds.
	var startDate, startTime ui.Element
	start := ui.Field(c, tr("Start"), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				startDate = ui.DateInput(c, &a.voting.sessionStart).Label(tr("Start date")).
					Description(tr("Local time. Ballots are rejected before it."))
				startTime = ui.TimeInput(c, &a.voting.sessionStart).Label(tr("Start time")).
					Description(tr("Local time. Ballots are rejected before it."))
			})
			hint(c, "Local time. Ballots are rejected before it.")
		})
	})
	if startDate.Changed() || startTime.Changed() {
		a.voting.sessionStartErr = ""
	}
	attachFieldError(c, start, "Start", a.voting.sessionStartErr)
	if a.voting.sessionStartErr != "" {
		startTime.Error(localizeServiceErrorMessage(a.voting.sessionStartErr))
	}
	var endDate, endTime ui.Element
	end := ui.Field(c, tr("End"), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				endDate = ui.DateInput(c, &a.voting.sessionEnd).Label(tr("End date")).
					Description(tr("Local time. Ballots are rejected after it."))
				endTime = ui.TimeInput(c, &a.voting.sessionEnd).Label(tr("End time")).
					Description(tr("Local time. Ballots are rejected after it."))
			})
			hint(c, "Local time. Ballots are rejected after it.")
		})
	})
	if endDate.Changed() || endTime.Changed() {
		a.voting.sessionEndErr = ""
	}
	attachFieldError(c, end, "End", a.voting.sessionEndErr)
	if a.voting.sessionEndErr != "" {
		endTime.Error(localizeServiceErrorMessage(a.voting.sessionEndErr))
	}
	fieldHint(c, "Candidate IDs (comma-separated)", "Leave empty to include every registered candidate.", &a.voting.sessionIDs)
	ui.Row(c).Children(func() {
		if primaryButton(c, "Create Session").Disabled(!allSet(a.voting.sessionTitle)).Clicked() {
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
			emptyStateIcon(c, calendarIcon, ui.Hex(votingAccent), "No sessions yet.",
				"Create one on the right with a roster and the window during which ballots count.")
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
			emptyStateIcon(c, userCheckIcon, ui.Hex(votingAccent), "No candidates registered yet.",
				"Register one on the right; a ballot names the candidate id it selects.")
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

// addCandidateToRoster appends the chosen candidate to the session form's
// roster field. The field takes comma-separated ids, so building a custom
// roster used to mean copying UUIDs by hand; this is the button that makes the
// form usable for anything but "every candidate".
func (a *App) addCandidateToRoster() {
	candidates := a.filteredCandidates()
	if a.voting.candChoice < 0 || a.voting.candChoice >= len(candidates) {
		a.voting.status = errorStatus("Choose a candidate first.")
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
	a.voting.sessionStartErr, a.voting.sessionEndErr = "", ""
	a.voting.sessionCopyID = ""
	start := a.voting.sessionStart.Unix()
	end := a.voting.sessionEnd.Unix()
	if a.voting.sessionStart.IsZero() {
		a.voting.sessionStartErr = "Choose a start time."
	}
	if a.voting.sessionEnd.IsZero() {
		a.voting.sessionEndErr = "Choose an end time."
	}
	if a.voting.sessionStartErr != "" || a.voting.sessionEndErr != "" {
		return
	}
	if end <= start {
		a.voting.sessionEndErr = "End time must be after the start time."
		return
	}
	title := strings.TrimSpace(a.voting.sessionTitle)
	if rejectIncomplete(&a.voting.sessionResult, "Session title is required.", title) {
		return
	}
	ids := splitParticipants(a.voting.sessionIDs)
	if len(ids) == 0 {
		// The visible list is one page, but an empty roster means every
		// registered candidate. Read the complete id set only for this write;
		// the collection itself remains paged.
		all, err := a.services.Candidates(0, 0)
		if err != nil {
			a.voting.sessionResult = errStatus(err)
			return
		}
		for _, c := range all {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		a.voting.sessionResult = errorStatus("Register at least one candidate before creating a session.")
		return
	}
	description := strings.TrimSpace(a.voting.sessionDesc)
	a.runWrite("Creating session…", func() func() {
		s, err := a.services.CreateSession(title, description, ids, start, end)
		return func() {
			if err != nil {
				a.voting.sessionResult = errStatus(err)
				return
			}
			a.voting.sessionResult = okStatus("Created session %s", s.Title)
			a.voting.sessionCopyID = s.ID
			a.voting.sessionTitle, a.voting.sessionDesc, a.voting.sessionIDs = "", "", ""
			a.voting.sessionStart, a.voting.sessionEnd = defaultSessionWindow(time.Now())
			a.voting.sessionChoice = -1
			a.loadSessionPage(0)
			// The new session becomes the context for controls, ballot and tally;
			// highlight its row so a successful create has a visible result even
			// before the copy button is used.
			a.selectSessionByID(s.ID)
			a.clearTallyForAnotherSession(s.ID)
			a.voting.controlID, a.voting.voteSession, a.voting.resultsSession = s.ID, s.ID, s.ID
		}
	})
}

// selectSessionByID highlights a session after creation. The search is
// cleared first so the index is in the list the table actually shows.
func (a *App) selectSessionByID(id string) {
	a.voting.sessionSearch, a.voting.sessionApplied = "", ""
	a.clearSessionRoster()
	for i, session := range a.filteredSessions() {
		if session.ID == id {
			a.voting.sessionChoice = i
			a.loadSelectedSessionRoster()
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
	// Cast Vote is the task and takes the leading column; the two session
	// lifecycle actions sit beside it rather than pushing the ballot form
	// below the fold on a default window.
	twoColumns(c, func() {
		a.votingCastVoteCard(c)
	}, func() {
		a.votingSessionControlsCard(c)
	})
}

func (a *App) votingCastVoteCard(c *ui.Context) {
	formCard(c, "Cast Vote", "Sign a ballot for one candidate in one session.", func() {
		fieldPair(c, "Session ID", &a.voting.voteSession, "Candidate ID", &a.voting.voteCandidate)
		field(c, "Voter Public Key", &a.voting.voteVoterPub)
		field(c, "Voter Private Key", &a.voting.votePriv)
		a.useSavedKeyButton(c, a.voting.voteVoterPub, &a.voting.votePriv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Cast Vote").Disabled(!allSet(a.voting.voteSession, a.voting.voteCandidate, a.voting.voteVoterPub, a.voting.votePriv)).Clicked() {
				a.castVote()
			}
		})
		inlineStatus(c, a.voting.voteResult)
	})
}

func (a *App) votingSessionControlsCard(c *ui.Context) {
	formCard(c, "Session Controls", "Start or end a session by id.", func() {
		fieldHint(c, "Session ID",
			"Choose a session above or enter its ID first.", &a.voting.controlID)
		sessionReady := strings.TrimSpace(a.voting.controlID) != ""
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if secondaryButton(c, "Start Session").Disabled(!sessionReady).Clicked() {
				a.setSessionControl("Starting session…", a.services.StartSession, "Session started.")
			}
			if secondaryButton(c, "End Session").Disabled(!sessionReady).Clicked() {
				a.setSessionControl("Ending session…", a.services.EndSession, "Session ended.")
			}
		})
		inlineStatus(c, a.voting.controlResult)
	})
}

func (a *App) setSessionControl(label string, fn func(string) error, success string) {
	id := strings.TrimSpace(a.voting.controlID)
	if id == "" {
		return
	}
	a.runWrite(label, func() func() {
		err := fn(id)
		return func() {
			if err != nil {
				a.voting.controlResult = errStatus(err)
				return
			}
			a.voting.controlResult = okStatus(success)
			a.loadVoting()
		}
	})
}

func (a *App) castVote() {
	a.voting.voteResult = statusLine{}
	if strings.TrimSpace(a.voting.voteSession) == "" || strings.TrimSpace(a.voting.voteCandidate) == "" {
		a.voting.voteResult = errorStatus("Choose a session and a candidate.")
		return
	}
	publicKey, privateKey := strings.TrimSpace(a.voting.voteVoterPub), strings.TrimSpace(a.voting.votePriv)
	sessionID, candidateID := strings.TrimSpace(a.voting.voteSession), strings.TrimSpace(a.voting.voteCandidate)
	if rejectIncomplete(&a.voting.voteResult, "Voter public key and private key are required.", publicKey, privateKey) {
		return
	}
	a.runWrite("Casting vote…", func() func() {
		err := a.services.CastVote(sessionID, candidateID, publicKey, privateKey)
		return func() {
			if err != nil {
				a.voting.voteResult = errStatus(err)
				return
			}
			a.rememberSuccessfulKey(publicKey, privateKey)
			a.voting.voteResult = okStatus("Ballot recorded.")
			a.voting.votePriv = ""
			a.loadVoting()
			// A tally already on screen for this session must include the ballot
			// that was just accepted; leaving the old numbers beside "Ballot
			// recorded" read as a vote that did not count. A failed refresh keeps
			// the old rows rather than replacing them with an error for a write
			// that succeeded.
			if sessionID != "" && sessionID == strings.TrimSpace(a.voting.resultsSession) &&
				(a.voting.resultsLoadedID == sessionID || len(a.voting.results) > 0 || a.voting.resultsErr != nil) {
				a.clearTallyForAnotherSession(sessionID)
				_ = a.loadSessionResults(sessionID)
			}
		}
	})
}

// loadSessionResults reads a session's tally into the Results card. A failed
// read records the attempted session just like a successful empty read, so the
// automatic tab load does not retry a failing query on every frame; callers
// that change the subject clear the old rows first. The explicit Show Results
// and Refresh actions remain the retry path.
func (a *App) loadSessionResults(sessionID string) error {
	rows, err := a.services.SessionResults(sessionID)
	if err != nil {
		a.voting.resultsErr = err
		a.voting.resultsLoadedID = sessionID
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
	startLine, endLine := sessionWindowLines(start, end)
	if endLine == "" {
		return startLine
	}
	return startLine + " " + endLine
}

// sessionWindowLines splits the exact local start and end times so a table
// column can stack them without truncating either timestamp.
func sessionWindowLines(start, end int64) (string, string) {
	if start == 0 && end == 0 {
		return "—", ""
	}
	return time.Unix(start, 0).Format(localTimeLayout),
		"→ " + time.Unix(end, 0).Format(localTimeLayout)
}

func (a *App) votingResults(c *ui.Context) {
	// The query and the tally are separate cards side by side: the session
	// field is one line of work, and the report it produces reads better at
	// half the window than as a full-width table under an empty field.
	twoColumns(c, func() {
		a.votingResultsQuery(c)
	}, func() {
		a.votingResultsTally(c)
	})
}

func (a *App) votingResultsQuery(c *ui.Context) {
	formCard(c, "Results", "Live tally per candidate for a session.", func() {
		field(c, "Session ID", &a.voting.resultsSession)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if secondaryButton(c, "Show Results").Disabled(!allSet(a.voting.resultsSession)).Clicked() {
				a.readSessionResults()
			}
		})
	})
}

func (a *App) votingResultsTally(c *ui.Context) {
	formCard(c, "Tally", "Votes per candidate in the loaded session.", func() {
		a.votingResultsBody(c)
	})
}

func (a *App) votingResultsBody(c *ui.Context) {
	id := strings.TrimSpace(a.voting.resultsSession)
	matches := id != "" && a.voting.resultsLoadedID == id
	if a.voting.resultsErr != nil && matches &&
		!loadWarning(c, "Failed to load results", a.voting.resultsErr, len(a.voting.results) > 0) {
		return
	}
	if !matches {
		emptyPanel(c, pageIcons[pageVoting], pageAccent(pageVoting), "No tally loaded yet.")
		return
	}
	if len(a.voting.results) == 0 {
		emptyPanel(c, pageIcons[pageVoting], pageAccent(pageVoting), "No votes recorded for this session.")
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
	if wideLayout(c) {
		a.votingResultBars(c, results)
	} else {
		// A bar cannot wrap a long name at enlarged text, so the tally falls
		// back to plain records: the same numbers, with the comparison made by
		// reading them instead of by the bar lengths.
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			for _, result := range results {
				result := result
				recordPanel(c, func() {
					ui.Text(c, valueOr(result.Name)).FontSize(textSize(c, 13)).Bold().
						TextColor(c.Theme().Text)
					recordMetric(c, "Votes", strconv.Itoa(result.Votes), c.Theme().Text)
				})
			}
		})
	}
}

// votingResultBars draws the tally as one row per candidate: the name and its
// exact count, over a bar scaled to the leader. The count stays exact, so the
// visual is a second reading of the same numbers rather than a replacement for
// them; the list role and each row's label keep the ordering and both facts
// available to a screen reader. At enlarged text, records take over because a
// bar cannot wrap a long name.
func (a *App) votingResultBars(c *ui.Context, results []ResultRow) {
	leader := 1
	for _, result := range results {
		leader = max(leader, result.Votes)
	}
	list := ui.Column(c).FillWidth().Gap(space3).Role(ui.RoleList).
		Label(tr("Voting results"))
	list.Children(func() {
		for _, result := range results {
			result := result
			share := float32(result.Votes) / float32(leader) * 100
			ui.Column(c).FillWidth().Gap(space1).Role(ui.RoleListItem).
				Label(fmt.Sprintf(tr("%s: %d votes"), valueOr(result.Name), result.Votes)).
				Children(func() {
					ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Children(func() {
						ui.Text(c, valueOr(result.Name)).FontSize(textSize(c, 12.5)).
							TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0).
							SingleLine().Ellipsis("…").Tooltip(result.Name)
						ui.Text(c, strconv.Itoa(result.Votes)).FontSize(textSize(c, 12.5)).
							FontWeight(700).FontFeatures("tnum").TextColor(ui.Hex(votingAccent))
					})
					track := ui.Row(c).FillWidth().Height(6).Radius(3).
						Background(c.Theme().Background)
					track.Children(func() {
						if share <= 0 {
							return
						}
						ui.Row(c).WidthPercent(share).Height(6).Radius(3).
							Gradient(ui.Hex(votingAccent), ui.Hex(aurora), 90)
					})
				})
		}
	})
}

func (a *App) readSessionResults() {
	id := strings.TrimSpace(a.voting.resultsSession)
	if rejectIncomplete(&a.voting.status, "Session ID is required for this read.", id) {
		return
	}
	a.clearTallyForAnotherSession(id)
	if err := a.loadSessionResults(id); err != nil {
		a.voting.status = errStatus(err)
	} else {
		a.voting.status = statusLine{}
	}
}
