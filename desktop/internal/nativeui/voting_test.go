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

	require.NoError(t, tester.Click(shortID(sessionID)), "click the session's id cell")
	tester.Frame()
	require.Equal(t, sessionID, app.voting.controlID, "session controls must receive the session")
	require.Equal(t, sessionID, app.voting.voteSession, "the ballot must receive the session")
	require.Equal(t, sessionID, app.voting.resultsSession, "the tally reader must receive the session")

	require.NoError(t, tester.Click(shortID(candID)), "click the candidate's id cell")
	tester.Frame()
	require.Equal(t, candID, app.voting.voteCandidate, "the ballot must receive the candidate")
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
	require.Contains(t, app.voting.sessionResult.text, "after the start")
	require.True(t, app.voting.sessionResult.isErr)
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
	require.NoError(t, tester.Click("s1"))
	tester.Frame()
	require.True(t, tester.HasText("Alice, Bob"), "the chosen session's roster must be shown")
	require.True(t, tester.HasText("Approve the 2027 budget"), "the chosen session's description must be shown")
}

// TestSelectedCandidateShowsItsProgram pins the candidate detail below the
// list: the table has no room for the program the ballot shows.
func TestSelectedCandidateShowsItsProgram(t *testing.T) {
	fake := &fakeServices{candidates: []CandidateRow{{
		ID: "c1", Name: "Alice", Party: "Greens", Program: "Free transit", VoteCount: 37,
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Voting")
	tester.Frame()

	require.False(t, tester.HasText("Free transit"), "no program before a candidate is chosen")
	require.NoError(t, tester.Click("c1"))
	tester.Frame()
	require.True(t, tester.HasText("Free transit"), "the chosen candidate's program must be shown")
	require.True(t, tester.HasText("37"), "the chosen candidate's votes must be shown")
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
	require.False(t, app.voting.voteResult.isErr)
	require.Equal(t, "Ballot recorded.", app.voting.voteResult.text)

	app.voting.sessionStart = time.Unix(2000, 0)
	app.voting.sessionEnd = time.Unix(1000, 0)
	app.createVotingSession()
	require.True(t, app.voting.sessionResult.isErr)
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
	require.True(t, tester.HasText("Voted"))
	require.NoError(t, tester.Click(shortID(publicKey)))
	tester.Frame()

	require.Equal(t, publicKey, app.voting.voteVoterPub,
		"choosing a voter must fill the ballot public-key field")
	require.Empty(t, app.voting.votePriv, "another voter's private key must not carry over")
	require.True(t, tester.HasText("Selected voter public key"))
}
