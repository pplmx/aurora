package nativeui

import (
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
	require.Contains(t, app.voting.sessionResult, "after the start")
}
