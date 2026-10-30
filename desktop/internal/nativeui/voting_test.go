package nativeui

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestVotingTablesLoadFullIDsIntoTheForms pins how a UUID reaches the forms on
// the voting page. The tables shorten ids to fit, and every session form (start,
// end, ballot, results) plus the ballot's candidate field needs the full one.
// Before this, the session table had no selection at all and no candidate list
// was rendered: the only place a candidate id appeared was the one-line result
// of Register Candidate, so after a restart neither a ballot nor a session
// roster could be filled from the window.
func TestVotingTablesLoadFullIDsIntoTheForms(t *testing.T) {
	const (
		sessionID = "11111111-2222-3333-4444-555555555555"
		candID    = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	)
	fake := &fakeServices{
		candidates: []CandidateRow{{ID: candID, Name: "Alice", Party: "Greens"}},
		sessions: []SessionRow{{
			ID: sessionID, Title: "Budget 2026", Status: "active",
			StartTime: 1_800_000_000, EndTime: 1_800_086_400,
		}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.True(t, tester.HasText("Budget 2026"), "the session list must render")
	require.True(t, tester.HasText("Alice"), "the candidate list must render")

	// A record is chosen by its title; the full id stays on the record as a
	// field, so the choice still hands the forms the complete UUID.
	require.True(t, tester.HasText(sessionID), "the session record must show its full id")
	require.NoError(t, tester.Click("Budget 2026"), "choose the session record")
	tester.Frame()
	require.Equal(t, sessionID, app.voting.controlID, "session controls must receive the session")
	require.Equal(t, sessionID, app.voting.voteSession, "the ballot must receive the session")
	require.Equal(t, sessionID, app.voting.resultsSession, "the tally reader must receive the session")

	require.True(t, tester.HasText(candID), "the candidate record must show its full id")
	require.NoError(t, tester.Click("Alice"), "choose the candidate record")
	tester.Frame()
	require.Equal(t, candID, app.voting.voteCandidate, "the ballot must receive the candidate")
}

// TestSessionControlsUseTheBackgroundWritePath keeps Start/End consistent with
// every other write: the window shows which operation is running and remains
// responsive while the service call is in flight.
func TestSessionControlsUseTheBackgroundWritePath(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	fake := &gatedSessionControls{fakeServices: &fakeServices{}, release: release}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.SetBackgroundWrites(true)
	app.ShowPage("Voting")
	app.voting.tab = 1
	app.voting.controlID = "s1"
	tester.Frame()

	require.NoError(t, tester.Click("Start Session"))
	require.Equal(t, "Starting session…", app.busy,
		"the running control operation must be visible before it finishes")

	close(release)
	require.Eventually(t, func() bool {
		tester.Frame()
		return app.busy == "" && app.voting.controlResult.text == "Session started."
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, 1, fake.startSessionCalls)
}

// TestSessionControlsRequireASession keeps Start/End from writing with an
// empty id. Both buttons become usable as soon as the control field names a
// session, and the hint explains the disabled state before that.
func TestSessionControlsRequireASession(t *testing.T) {
	// One stored session keeps the Ballot stage on the page: with none at all
	// the stage names the missing session instead of showing its controls.
	fake := &fakeServices{sessions: []SessionRow{{ID: "s1", Title: "Budget", Status: "active"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 1
	tester.Frame()

	require.True(t, tester.HasText("Choose a session in Setup or enter its ID first."))
	require.NoError(t, tester.Click("Start Session"))
	require.NoError(t, tester.Click("End Session"))
	tester.Frame()
	require.Zero(t, fake.startSessionCalls)
	require.Zero(t, fake.endSessionCalls)

	app.voting.controlID = "s1"
	tester.Frame()
	require.NoError(t, tester.Click("Start Session"))
	tester.Frame()
	require.NoError(t, tester.Click("End Session"))
	tester.Frame()
	require.Equal(t, 1, fake.startSessionCalls)
	require.Equal(t, 1, fake.endSessionCalls)
}

// TestRegisterCandidateRequiresAName keeps the write button disabled until
// the field that names the new candidate is present.
func TestRegisterCandidateRequiresAName(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.NoError(t, tester.Click("Candidate Name"))
	tester.Frame()
	for range 3 {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	require.False(t, tester.Focused("Register Candidate"),
		"an empty candidate name must keep the action out of the tab order")
	app.registerCandidate()
	tester.Frame()
	require.Zero(t, fake.candidateRegisterCalls)
	require.True(t, tester.HasText("Candidate name is required."),
		"the handler must keep rejecting an empty name defensively")

	require.NoError(t, tester.Click("Candidate Name"))
	tester.Type("Alice")
	tester.Frame()
	for range 3 {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	require.True(t, tester.Focused("Register Candidate"))
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, 1, fake.candidateRegisterCalls)
}

// gatedSessionControls holds the write open until the test has observed the
// busy state. A short sleep raced the 20-frame click settle under -race and
// could complete before the assertion even though the production path worked.
type gatedSessionControls struct {
	*fakeServices
	release <-chan struct{}
}

func (s *gatedSessionControls) StartSession(string) error {
	s.startSessionCalls++
	<-s.release
	return nil
}

// TestVoterKeyPairLeavesTheWindow proves the one-time key handover is usable.
// Native text cannot be selected, and the store never returns the private half,
// so without a Copy button the operator could neither keep nor transfer the
// signing key the registration generated.
func TestVoterKeyPairLeavesTheWindow(t *testing.T) {
	app := New(&voterKeyRecorder{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	app.voting.voterName = "Alice"
	app.registerVoter()
	tester.Frame()

	require.True(t, tester.HasText("New voter key pair"))
	require.True(t, tester.HasText("sk-Alice"))

	require.NoError(t, tester.Click("Copy Private key"))
	tester.Frame()
	require.Equal(t, "sk-Alice", tester.Clipboard(),
		"the private key must reach the clipboard, not just the screen")
	require.Contains(t, app.voting.status.text, "Private key copied")

	require.NoError(t, tester.Click("Copy Public key"))
	tester.Frame()
	require.Equal(t, "pk-Alice", tester.Clipboard())
}

// TestNewCandidateAndSessionIDsCanBeCopied covers the two identifiers the
// voting forms cannot do without and cannot show in full: the candidate id a
// session roster needs, and the session id the controls, ballot and tally
// reader need. The tables shorten both, so the just-created value is offered
// with a Copy button.
func TestNewCandidateAndSessionIDsCanBeCopied(t *testing.T) {
	fake := &fakeServices{candidates: []CandidateRow{{ID: "c1", Name: "Alice"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	app.voting.candName = "Alice"
	app.registerCandidate()
	tester.Frame()
	require.Equal(t, "c1", app.voting.candCopyID)
	require.NoError(t, tester.Click("Copy Candidate ID"))
	tester.Frame()
	require.Equal(t, "c1", tester.Clipboard())

	app.voting.sessionTitle = "Budget"
	app.voting.sessionStart = time.Unix(1000, 0)
	app.voting.sessionEnd = time.Unix(2000, 0)
	app.voting.sessionIDs = "c1"
	app.createVotingSession()
	tester.Frame()
	require.Equal(t, "s1", app.voting.sessionCopyID)
	require.NoError(t, tester.Click("Copy Session ID"))
	tester.Frame()
	require.Equal(t, "s1", tester.Clipboard())
}

// sessionRecorder remembers the window and roster the form sent.
type sessionRecorder struct {
	*fakeServices
	start, end int64
	ids        []string
}

func (r *sessionRecorder) CreateSession(title, description string, ids []string, start, end int64) (SessionRow, error) {
	r.start, r.end, r.ids = start, end, ids
	return SessionRow{ID: "s1", Title: title, Status: "pending"}, nil
}

// TestSessionWindowIsEditedAsLocalTime pins the Create Session card's inputs.
// It used to ask for two Unix timestamps typed by hand, so creating a session
// from the window needed a clock and a converter outside the app. The card now
// opens with now → +24h and edits one time.Time with the toolkit's date and
// time controls; the service still receives Unix seconds.
func TestSessionWindowIsEditedAsLocalTime(t *testing.T) {
	rec := &sessionRecorder{fakeServices: &fakeServices{
		candidates: []CandidateRow{{ID: "c1", Name: "Alice"}},
	}}
	app := New(rec)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()
	require.True(t, tester.HasText("Start"))
	require.True(t, tester.HasText("End"))

	// The form opens with a usable window instead of empty controls.
	require.False(t, app.voting.sessionStart.IsZero())
	require.True(t, app.voting.sessionEnd.After(app.voting.sessionStart))

	start := time.Date(2026, 3, 1, 9, 30, 0, 0, time.Local)
	end := start.Add(48 * time.Hour)
	app.voting.sessionTitle = "Budget"
	app.voting.sessionStart, app.voting.sessionEnd = start, end
	app.createVotingSession()
	require.Equal(t, start.Unix(), rec.start)
	require.Equal(t, end.Unix(), rec.end)
	require.Equal(t, []string{"c1"}, rec.ids, "an empty roster includes every candidate")

	// The form is ready for the next session again, and the id is copyable.
	require.False(t, app.voting.sessionStart.IsZero())
	require.True(t, app.voting.sessionEnd.After(app.voting.sessionStart))
	require.Equal(t, "s1", app.voting.sessionCopyID)

	// An end before the start is refused before the service is called.
	sentStart := rec.start
	app.voting.sessionTitle = "Broken"
	app.voting.sessionStart, app.voting.sessionEnd = end, start
	app.createVotingSession()
	require.Equal(t, sentStart, rec.start, "an invalid window must not reach the service")
	require.Contains(t, app.voting.sessionEndErr, "after the start")
	require.True(t, app.voting.sessionResult.empty())
	tester.Frame()
	require.True(t, tester.HasText("End time must be after the start time."),
		"the error must stay attached to the End field")

	app.voting.sessionStart = time.Time{}
	app.voting.sessionEnd = time.Unix(1000, 0)
	app.createVotingSession()
	require.NotEmpty(t, app.voting.sessionStartErr)
	require.Empty(t, app.voting.sessionEndErr)
	tester.Frame()
	require.True(t, tester.HasText("Choose a start time."),
		"a missing start must be explained at the Start field")

	app.voting.sessionStart = time.Unix(1000, 0)
	app.voting.sessionEnd = time.Time{}
	app.createVotingSession()
	require.Empty(t, app.voting.sessionStartErr)
	require.NotEmpty(t, app.voting.sessionEndErr)
	tester.Frame()
	require.True(t, tester.HasText("Choose an end time."),
		"a missing end must be explained at the End field")
}

// TestEmptySessionRosterIncludesCandidatesBeyondTheVisiblePage keeps the
// collection pager from changing the form's contract: an empty roster still
// means every registered candidate, not just the newest hundred on screen.
func TestEmptySessionRosterIncludesCandidatesBeyondTheVisiblePage(t *testing.T) {
	all := make([]CandidateRow, votingPageSize+25)
	for i := range all {
		all[i] = CandidateRow{ID: "c" + strconv.Itoa(i), Name: "Candidate"}
	}
	rec := &sessionRecorder{fakeServices: &fakeServices{candidates: all}}
	app := New(rec)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()
	require.Len(t, app.voting.candidates, votingPageSize)

	app.voting.sessionTitle = "All candidates"
	app.voting.sessionStart = time.Unix(1000, 0)
	app.voting.sessionEnd = time.Unix(2000, 0)
	app.createVotingSession()

	require.Len(t, rec.ids, len(all), "the session must include candidates outside the visible page")
	require.Equal(t, all[0].ID, rec.ids[0])
	require.Equal(t, all[len(all)-1].ID, rec.ids[len(rec.ids)-1])
}

// TestResultsShowTheirTotal pins the line above the tally. Counting the rows'
// votes by eye is work the view can do, and the total is what tells an
// operator whether every ballot was counted.
func TestResultsShowTheirTotal(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 2
	app.voting.resultsSession = "s1"
	app.voting.resultsLoadedID = "s1"
	app.voting.results = []ResultRow{
		{CandidateID: "c1", Name: "Alice", Votes: 7},
		{CandidateID: "c2", Name: "Bob", Votes: 6},
	}
	tester.Frame()

	require.True(t, tester.HasText("Total votes"))
	require.True(t, tester.HasText("13"), "the total must be the sum of the rows")
}

// TestCandidateCardBuildsACustomRoster pins the roster button. The Create
// Session field takes comma-separated ids, so a roster other than "every
// candidate" used to require copying UUIDs out of a shortened table by hand.
func TestCandidateCardBuildsACustomRoster(t *testing.T) {
	rec := &sessionRecorder{fakeServices: &fakeServices{candidates: []CandidateRow{
		{ID: "c1", Name: "Alice"},
		{ID: "c2", Name: "Bob"},
	}}}
	app := New(rec)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.NoError(t, tester.Click("c1"))
	tester.Frame()
	require.Equal(t, "c1", app.voting.voteCandidate, "choosing a candidate fills the ballot")

	require.NoError(t, tester.Click("Add to session roster"))
	tester.Frame()
	require.Equal(t, "c1", app.voting.sessionIDs)
	require.Contains(t, app.voting.status.text, "Added Alice")

	require.NoError(t, tester.Click("c2"))
	tester.Frame()
	require.NoError(t, tester.Click("Add to session roster"))
	tester.Frame()
	require.Equal(t, "c1,c2", app.voting.sessionIDs)

	// Adding the same candidate twice must not duplicate the roster entry.
	require.NoError(t, tester.Click("c1"))
	tester.Frame()
	require.NoError(t, tester.Click("Add to session roster"))
	tester.Frame()
	require.Equal(t, "c1,c2", app.voting.sessionIDs)
	require.Contains(t, app.voting.status.text, "already in the session roster")

	app.voting.sessionTitle = "Budget"
	app.voting.sessionStart = time.Unix(1000, 0)
	app.voting.sessionEnd = time.Unix(2000, 0)
	app.createVotingSession()
	require.Equal(t, []string{"c1", "c2"}, rec.ids, "the session must use the built roster")
}

// TestSelectedSessionShowsItsRoster pins the preview under the session table:
// the row cannot carry the roster, and an operator about to start or end a
// session needs to see who is on the ballot without opening another page. The
// description that was typed into the Create Session form is shown with it.
func TestSelectedSessionShowsItsRoster(t *testing.T) {
	fake := &fakeServices{
		candidates: []CandidateRow{
			{ID: "c1", Name: "Alice"},
			{ID: "c2", Name: "Bob"},
		},
		sessions: []SessionRow{{
			ID: "s1", Title: "Budget", Description: "Approve the 2027 budget", Status: "active",
			Candidates: []string{"c1", "c2"},
		}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.False(t, tester.HasText("Alice, Bob"), "no roster before a row is chosen")
	require.False(t, tester.HasText("Approve the 2027 budget"), "no description before a row is chosen")
	require.NoError(t, tester.Click("Budget"), "choose the session record")
	tester.Frame()
	require.True(t, tester.HasText("Alice, Bob"), "the chosen session's roster must be shown")
	require.True(t, tester.HasText("Approve the 2027 budget"), "the chosen session's description must be shown")
}

func TestSelectedSessionRosterResolvesCandidatesBeyondTheVisiblePage(t *testing.T) {
	page := make([]CandidateRow, votingPageSize)
	for i := range page {
		page[i] = CandidateRow{ID: "page-" + strconv.Itoa(i), Name: "Page candidate " + strconv.Itoa(i)}
	}
	fake := &fakeServices{
		candidates: page,
		candidateNames: map[string]string{
			"cand-101": "Alice",
			"cand-102": "Bob",
		},
		sessions: []SessionRow{{
			ID: "s1", Title: "Budget", Status: "active",
			Candidates: []string{"cand-101", "cand-102"},
		}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Voting")
	tester.Frame()

	// The session list sits below a full page of candidates, so scroll it into
	// view before choosing it.
	session, ok := tester.Find("Budget")
	require.True(t, ok)
	app.scroll.Y = max(0, session.Y-300)
	tester.Frame()
	require.NoError(t, tester.Click("Budget"), "choose the session record")
	tester.Frame()
	require.True(t, tester.HasText("Alice, Bob"),
		"a session roster must resolve names even when both candidates are on an older page")
	require.Equal(t, []string{"cand-101", "cand-102"}, fake.lastRosterIDs)
	reads := fake.candidateNameCalls
	tester.Frame()
	require.Equal(t, reads, fake.candidateNameCalls,
		"rendering another frame must not repeat the roster lookup")
}

func TestFailedSessionRosterReadKeepsOnlyThatSessionsNames(t *testing.T) {
	fake := &fakeServices{
		candidateNames: map[string]string{"c1": "Alice", "c2": "Bob"},
		sessions: []SessionRow{
			{ID: "s1", Title: "Budget", Status: "active", Candidates: []string{"c1"}},
			{ID: "s2", Title: "Standup", Status: "draft", Candidates: []string{"c2"}},
		},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.True(t, tester.HasText("Alice"))

	boom := errors.New("database is locked")
	fake.loadErrs.sessionRoster = boom
	app.refresh()
	tester.Frame()
	require.True(t, tester.HasText("Couldn't load session roster: database is locked"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText("Alice"), "the same session keeps its last good roster names")

	require.NoError(t, tester.Click("s2"))
	tester.Frame()
	require.True(t, tester.HasText("Couldn't load session roster: database is locked"))
	require.False(t, tester.HasText("Showing the last successful load."),
		"a different session must not inherit the previous session's stale label")
	require.False(t, tester.HasText("Alice"), "one session's names must not follow another session")
	require.True(t, tester.HasText("c2"), "a missing name falls back to the stored id")
}

// TestCandidateRecordShowsItsProgram keeps the program and the tally on the
// candidate's own record. The old table had no room for either, so they only
// appeared in a readout after the row was chosen — a fact about the candidate
// that the operator had to select the candidate to discover.
func TestCandidateRecordShowsItsProgram(t *testing.T) {
	fake := &fakeServices{candidates: []CandidateRow{{
		ID: "c1", Name: "Alice", Party: "Greens", Program: "Free transit", VoteCount: 37,
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.True(t, tester.HasText("Free transit"), "the record must show the program")
	require.True(t, tester.HasText("37"), "the record must show the tally")
	require.NoError(t, tester.Click("Alice"))
	tester.Frame()
	require.Equal(t, "c1", app.voting.voteCandidate, "choosing the record must aim the ballot")
}

// TestFailedTallyRefreshKeepsTheRowsAndSaysWhy covers a refresh of the tally
// already on screen: a transient read failure must keep the last successful
// rows (they still describe the same session) and mark them stale, while a
// later successful read clears the warning.
func TestFailedTallyRefreshKeepsTheRowsAndSaysWhy(t *testing.T) {
	fake := &fakeServices{resultsByID: map[string][]ResultRow{
		"s1": {{CandidateID: "c1", Name: "Alice", Votes: 3}},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 2
	tester.Frame()

	app.voting.resultsSession = "s1"
	require.NoError(t, app.loadSessionResults("s1"))
	tester.Frame()
	require.True(t, tester.HasText("Alice"))

	fake.sessionResultsErr = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.Len(t, app.voting.results, 1, "the last successful tally must stay")
	require.True(t, tester.HasText("Failed to load results"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText("Alice"))

	fake.sessionResultsErr = nil
	app.refresh()
	tester.Frame()
	require.NoError(t, app.voting.resultsErr)
	require.False(t, tester.HasText("Failed to load results"))
	require.True(t, tester.HasText("Alice"))
}

// TestFailedTallyReadDropsAnotherSessionsRows pins the subject rule: selecting
// a different session clears the rows before reading, so a failure cannot
// leave session A's numbers under session B's id with no warning.
func TestFailedTallyReadDropsAnotherSessionsRows(t *testing.T) {
	fake := &fakeServices{
		sessions: []SessionRow{
			{ID: "s1", Title: "Budget", Status: "ended"},
			{ID: "s2", Title: "Charter", Status: "active"},
		},
		resultsByID: map[string][]ResultRow{
			"s1": {{CandidateID: "c1", Name: "Alice", Votes: 3}},
		},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 2
	tester.Frame()

	app.voting.resultsSession = "s1"
	require.NoError(t, app.loadSessionResults("s1"))
	tester.Frame()
	require.True(t, tester.HasText("Alice"))

	fake.sessionResultsErr = errors.New("database is locked")
	// The session table is on the Setup stage; choosing a row there still moves
	// the tally reader, which is what the Results stage shows.
	app.voting.tab = 0
	tester.Frame()
	require.NoError(t, tester.Click("s2"), "click the second session's id cell")
	tester.Frame()
	app.voting.tab = 2
	tester.Frame()
	require.Equal(t, "s2", app.voting.resultsSession)
	require.Empty(t, app.voting.results, "session A's tally must not sit under session B")
	require.True(t, tester.HasText("Failed to load results"))
	require.False(t, tester.HasText("Alice"))
}

// TestVotingCardOutcomesCarryTheirPolarity pins the type behind the four
// card-local results. They were plain strings drawn in the same muted colour,
// so a refused ballot looked like "Ballot recorded."; the handler must now
// mark the failure and the renderer uses that to choose the error colour.
func TestVotingCardOutcomesCarryTheirPolarity(t *testing.T) {
	app := New(&fakeServices{})

	app.castVote()
	require.True(t, app.voting.voteResult.isErr)
	require.Contains(t, app.voting.voteResult.text, "Choose a session and a candidate")

	app.voting.voteSession = "s1"
	app.voting.voteCandidate = "c1"
	app.castVote()
	require.True(t, app.voting.voteResult.isErr)
	require.Contains(t, app.voting.voteResult.text, "Voter public key and private key are required")

	app.voting.voteVoterPub, app.voting.votePriv = "pk", "sk"
	app.castVote()
	require.False(t, app.voting.voteResult.isErr)
	require.Equal(t, "Ballot recorded.", app.voting.voteResult.text)

	app.voting.sessionStart = time.Unix(2000, 0)
	app.voting.sessionEnd = time.Unix(1000, 0)
	app.createVotingSession()
	require.NotEmpty(t, app.voting.sessionEndErr)
	require.True(t, app.voting.sessionResult.empty())
}

// TestResultsStageLoadsTheChosenSession pins the tab's first move: a session
// selected in Setup is already the Results subject, so opening Results should
// show its tally instead of asking for another Show Results click.
func TestResultsStageLoadsTheChosenSession(t *testing.T) {
	fake := &fakeServices{
		sessions: []SessionRow{{ID: "s1", Title: "Budget", Status: "ended"}},
		resultsByID: map[string][]ResultRow{
			"s1": {{CandidateID: "c1", Name: "Alice", Votes: 4}},
		},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.Equal(t, "s1", app.voting.resultsSession)
	require.NoError(t, tester.Click("Results"))
	tester.Frame()

	require.True(t, tester.HasText("Alice"), "the selected session's tally must load with the tab")
	require.Equal(t, 4, app.voting.results[0].Votes)
}

// TestFailedAutomaticTallyReadDoesNotHammerTheDatabase keeps the results tab's
// auto-load from becoming a render loop. A failed read has no rows to key on,
// so the attempted session must still be remembered; otherwise every frame
// retries the same locked database and the failure consumes the UI thread.
func TestFailedAutomaticTallyReadDoesNotHammerTheDatabase(t *testing.T) {
	fake := &fakeServices{sessionResultsErr: errors.New("database is locked")}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	app.voting.tab = 2
	app.voting.resultsSession = "s1"

	tester.Frame()
	require.ErrorIs(t, app.voting.resultsErr, fake.sessionResultsErr)
	require.Equal(t, 1, fake.sessionResultsCalls, "opening Results should read once")

	tester.Frame()
	tester.Frame()
	require.Equal(t, 1, fake.sessionResultsCalls,
		"a failed automatic read must not repeat on every rendered frame")
	require.True(t, tester.HasText("Failed to load results"))

	fake.sessionResultsErr = nil
	require.NoError(t, tester.Click("Show Results"))
	tester.Frame()
	require.Equal(t, 2, fake.sessionResultsCalls, "the explicit retry must still reach the service")
	require.NoError(t, app.voting.resultsErr)
}

// TestVoterListLoadsThePublicKeyIntoTheBallot restores the voter view after a
// restart. The registration key card disappears with the next action, so a
// registered voter otherwise had no visible public key to put in the ballot
// form even though the store still had one.
func TestVoterListLoadsThePublicKeyIntoTheBallot(t *testing.T) {
	const publicKey = "pk-voter-1111111111111111111111111111111111111111"
	app := New(&fakeServices{voterRows: []VoterRow{{
		PublicKey: publicKey, Name: "Alice", HasVoted: true,
	}}})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()

	require.True(t, tester.HasText("Alice"))
	// A voter record is chosen by name; its full key stays on the record.
	require.True(t, tester.HasText(publicKey), "the record must show the full key")
	require.NoError(t, tester.Click("Alice"))
	tester.Frame()

	require.Equal(t, publicKey, app.voting.voteVoterPub,
		"choosing a voter must fill the ballot public-key field")
	require.Empty(t, app.voting.votePriv, "another voter's private key must not carry over")
	require.True(t, tester.HasText("Selected voter public key"))
}

// TestVoterListUsesCompactHistoryLabels keeps both history states readable in
// the table's fixed-height rows. The previous multi-word labels wrapped and
// the next row was drawn over the clipped second line.
func TestVoterListUsesCompactHistoryLabels(t *testing.T) {
	app := New(&fakeServices{voterRows: []VoterRow{
		{PublicKey: "pk-voter-1", Name: "Alice", HasVoted: true},
		{PublicKey: "pk-voter-2", Name: "Ben"},
	}})
	tester := ui.NewTester(app.View, 900, 2200)
	app.ShowPage("Voting")
	tester.Frame()

	texts := tester.Texts()
	require.Contains(t, texts, "Ever voted")
	require.Contains(t, texts, "Yes")
	require.Contains(t, texts, "No")
	require.NotContains(t, texts, "Voted before")
	require.NotContains(t, texts, "No ballot yet")
}

// TestManualVoterKeyChangeClearsThePrivateKey covers the remaining ballot entry
// path: a public key typed directly into the field must not leave the previous
// voter's private key and "Ballot recorded" sentence attached to it.
func TestManualVoterKeyChangeClearsThePrivateKey(t *testing.T) {
	// One stored session keeps the ballot form on the stage; the manual field
	// this test types into only exists there.
	app := New(&fakeServices{sessions: []SessionRow{{ID: "s1", Title: "Budget", Status: "active"}}})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	app.voting.tab = 1
	tester.Frame()

	app.voting.voteVoterPub = "voter-a"
	tester.Frame()
	app.voting.votePriv = "key-a"
	app.voting.voteResult = okStatus("Ballot recorded.")
	tester.Frame()

	app.voting.voteVoterPub = "voter-b"
	tester.Frame()

	require.Empty(t, app.voting.votePriv, "another voter's private key must not survive")
	require.Empty(t, app.voting.voteResult.text, "the previous voter's verdict must not survive")
	require.False(t, tester.HasText("Ballot recorded."))
}

// TestBallotAndResultsNameTheMissingSession pins the first step of the two
// session-keyed stages. With no session stored, their forms asked for an id
// that could not exist yet; the stages now name the missing session and offer
// the one that creates it. A typed id still opens the forms, because the
// session may live outside the page the list loaded.
func TestBallotAndResultsNameTheMissingSession(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Voting")

	app.voting.tab = 1
	tester.Frame()
	require.True(t, tester.HasText("No sessions yet."))
	require.True(t, tester.HasText("A ballot names a session; create one in Setup and its id fills this form."))
	require.False(t, tester.HasText("Cast Vote"), "the ballot form has no session to name")

	require.NoError(t, tester.Click("Open Setup"))
	tester.Frame()
	require.Equal(t, 0, app.voting.tab, "Open Setup returns to the registration stage")

	app.voting.tab = 2
	tester.Frame()
	require.True(t, tester.HasText("A tally belongs to a session; create one in Setup, then read its results here."))

	app.voting.resultsSession = "s1"
	tester.Frame()
	require.True(t, tester.HasText("Show Results"), "a typed session id opens the tally query")

	app.voting.tab = 1
	app.voting.voteSession = "s1"
	tester.Frame()
	require.True(t, tester.HasText("Cast Vote"), "a typed session id opens the ballot form")
}
