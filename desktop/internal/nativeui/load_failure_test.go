package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// loadedContent is one page with the text that proves its last successful
// load is still on screen.
type loadedContent struct {
	page   string
	marker string
}

// allPagesLoaded fills a fake with one row per list page and returns the page
// titles with a marker each rendered view shows for its data.
func allPagesLoaded(fake *fakeServices) []loadedContent {
	fake.draws = []DrawRow{{
		ID: "draw-row-1", BlockHeight: 1, Seed: "seed",
		Participants: []string{"Alice", "Bob"}, Winners: []string{"Alice"},
	}}
	fake.candidates = []CandidateRow{{ID: "cand-1", Name: "Alice"}}
	fake.voterRows = []VoterRow{{PublicKey: "pk-voter-1", Name: "Voter Alice"}}
	fake.sessions = []SessionRow{{ID: "sess-1", Title: "Budget 2026", Status: "active"}}
	fake.tokenRows = []TokenRow{{ID: "token-1", Name: "GoldToken", Symbol: "GLD", TotalSupply: "1000"}}
	fake.nftRows = []NFTRow{{ID: "nft-1", Name: "StarNFT"}}
	fake.sources = []SourceRow{{ID: "src-1", Name: "btc-price"}}
	fake.blocks = []BlockRow{{Height: 1, Module: "lottery", Summary: "draw payload"}}
	return []loadedContent{
		{"Dashboard", "Total Lotteries"},
		{"Lottery", "draw-row-1"},
		{"Voting", "Budget 2026"},
		{"Token", "GoldToken"},
		{"Oracle", "btc-price"},
		{"NFT", "StarNFT"},
		{"Blockchain", "draw payload"},
	}
}

// TestFailedRefreshKeepsTheLastLoadedContent pins the policy the loaders state:
// a transient read error must not blank a list the operator was reading. Each
// page keeps the rows from its last successful load, says why the refresh
// failed, and marks the rows as stale. Voting used to draw no error at all,
// and every page returned on the error line alone, so the retained rows were
// never seen.
func TestFailedRefreshKeepsTheLastLoadedContent(t *testing.T) {
	boom := errors.New("database is locked")
	fake := &fakeServices{}
	cases := allPagesLoaded(fake)
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 6000)

	for _, c := range cases {
		app.ShowPage(c.page)
		tester.Frame()
		require.True(t, tester.HasText(c.marker), "%s must render its loaded content", c.page)
	}

	fake.loadErrs = fakeLoadErrors{
		history: boom, tokens: boom, nfts: boom, sources: boom,
		voters: boom, candidates: boom, sessions: boom, chain: boom, stats: boom,
	}
	for _, c := range cases {
		app.ShowPage(c.page)
		app.refresh()
		tester.Frame()
		require.True(t, tester.HasText("database is locked"), "%s must report the failed read", c.page)
		require.True(t, tester.HasText("Showing the last successful load."), "%s must keep its previous content", c.page)
		require.True(t, tester.HasText(c.marker), "%s must still show %q", c.page, c.marker)
		require.False(t, tester.HasText("No draws recorded yet. Create one above."), "%s must not claim an empty store", c.page)
		require.False(t, tester.HasText("No sessions yet."), "%s must not claim an empty store", c.page)
	}

	// A later good read clears the warning and the stale note.
	fake.loadErrs = fakeLoadErrors{}
	app.refresh()
	tester.Frame()
	require.False(t, tester.HasText("database is locked"))
	require.False(t, tester.HasText("Showing the last successful load."))
}

// TestFailedFirstLoadIsNotAnEmptyState covers the other half of the policy: with
// nothing loaded yet there is no content to keep, so the page reports the
// failure and must not fall through to its "nothing here yet" note (nor, on the
// dashboard, to a spinner that can never finish).
func TestFailedFirstLoadIsNotAnEmptyState(t *testing.T) {
	boom := errors.New("database is locked")
	fake := &fakeServices{}
	fake.loadErrs.history = boom
	fake.loadErrs.voters = boom
	fake.loadErrs.candidates = boom
	fake.loadErrs.sessions = boom
	fake.loadErrs.stats = boom
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)

	tester.Frame()
	require.True(t, tester.HasText("Couldn't load stats"))
	require.False(t, tester.HasText("Loading counters…"), "a failed load must not leave the spinner running")
	require.False(t, tester.HasText("Showing the last successful load."))

	app.ShowPage("Lottery")
	tester.Frame()
	require.True(t, tester.HasText("Failed to load history"))
	require.False(t, tester.HasText("No draws recorded yet. Create one above."))
	require.False(t, tester.HasText("Showing the last successful load."))

	app.ShowPage("Voting")
	tester.Frame()
	require.True(t, tester.HasText("Failed to load voting data"))
	require.False(t, tester.HasText("No voters registered yet."))
	require.False(t, tester.HasText("No sessions yet."))
	require.False(t, tester.HasText("No candidates registered yet."))
	require.False(t, tester.HasText("Showing the last successful load."))
}

// TestFailedHealthReadKeepsFeedRows covers the one oracle list that swallowed
// its error: health was only assigned on success, so a transient read failure
// left yesterday's counters on screen with no sign they were old.
func TestFailedHealthReadKeepsFeedRows(t *testing.T) {
	fake := &fakeServices{
		sources:    []SourceRow{{ID: "s1", Name: "btc"}},
		healthRows: []HealthRow{{SourceID: "s1", Name: "btc-feed", Attempts: 3}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	app.oracle.tab = 1
	tester.Frame()
	require.True(t, tester.HasText("btc-feed"))

	fake.loadErrs.health = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.True(t, tester.HasText("Failed to load feed health"))
	require.True(t, tester.HasText("database is locked"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText("btc-feed"), "the last counters must stay visible")
}

// TestFeedHealthResolvesSourceNames pins the label the scheduler cannot send:
// its counters carry a source id, while the table the operator reads should
// say which feed is failing. The page resolves it from the source list it
// already loaded; the filter here hides that list's row so the name can only
// come from the health table.
func TestFeedHealthResolvesSourceNames(t *testing.T) {
	fake := &fakeServices{
		sources:    []SourceRow{{ID: "s1", Name: "btc-feed"}},
		healthRows: []HealthRow{{SourceID: "s1", Attempts: 3, Failures: 1, LastError: "timeout"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	// The health table is its own stage; the source list is not on screen, so
	// the name can only come from the loaded source rows.
	app.oracle.tab = 1
	tester.Frame()

	require.True(t, tester.HasText("btc-feed"), "the health row must resolve the source name")
	require.True(t, tester.HasText("timeout"))
}

// TestFailedPresetReadKeepsTheAddForm covers the oracle preset catalog: a failed
// read was dropped, leaving a stale or absent picker with no explanation. The
// warning sits above a form that stays usable.
func TestFailedPresetReadKeepsTheAddForm(t *testing.T) {
	fake := &fakeServices{templates: []TemplateRow{
		{ID: "btc-price", Name: "Bitcoin Price", URL: "https://example.com/btc"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Oracle")
	tester.Frame()
	require.True(t, tester.HasText("Preset (optional)"))

	fake.loadErrs.templates = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.True(t, tester.HasText("Failed to load presets"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText("Preset (optional)"), "the form must remain usable")
	require.True(t, tester.HasText("Add Source"), "the add action must survive the warning")
}

// TestFailedBlockReadKeepsTheLedgerRows covers the chain block list: only its
// counters reported failures, so a failed entry read could leave an empty table
// saying "Nothing recorded on the chain yet." or silent old rows.
func TestFailedBlockReadKeepsTheLedgerRows(t *testing.T) {
	fake := &fakeServices{blocks: []BlockRow{{Height: 7, Module: "lottery", Summary: `{"winner":"A"}`}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	require.True(t, tester.HasText("lottery"))

	fake.loadErrs.blocks = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.True(t, tester.HasText("Couldn't read chain blocks"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText(`{"winner":"A"}`), "the last rows must stay visible")
	require.False(t, tester.HasText("Nothing recorded on the chain yet."))
}

// TestFailedLedgerQueryDoesNotRetryEveryFrame keeps an automatic search reload
// from hammering the database after it fails. The rendered search still names
// the failed query, but the attempted-query key stops a repaint from issuing
// the same read again; Refresh remains the explicit retry.
func TestFailedLedgerQueryDoesNotRetryEveryFrame(t *testing.T) {
	fake := &fakeServices{blocks: []BlockRow{{Height: 7, Module: "lottery", Summary: "old-row"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()

	before := fake.blockListCalls
	fake.loadErrs.blocks = errors.New("database is locked")
	app.blockchain.search = "new-query"
	tester.Frame()
	afterAttempt := fake.blockListCalls
	require.Equal(t, before+1, afterAttempt, "the changed search must trigger one read")

	tester.Frame()
	tester.Frame()
	require.Equal(t, afterAttempt, fake.blockListCalls,
		"repainting the failed query must not issue it again")
	require.True(t, tester.HasText("Couldn't read chain blocks"))
}
