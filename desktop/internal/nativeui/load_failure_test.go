package nativeui

import (
	"errors"
	"strings"
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

// TestFailedBlockReadKeepsTheLedgerRows covers the chain record list: only its
// counters reported failures, so a failed entry read could leave an empty page
// saying no module records exist or silent old rows.
func TestFailedBlockReadKeepsTheLedgerRows(t *testing.T) {
	fake := &fakeServices{blocks: []BlockRow{{
		Height: 7, Module: "lottery", Action: "Draw", Subject: "Alice",
		Summary: `{"winner":"A"}`, Payload: `{"winner":"A"}`,
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	require.True(t, tester.HasText("Alice"), "the record must be listed before the failure")

	fake.loadErrs.blocks = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.True(t, tester.HasText("Couldn't read chain blocks"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.True(t, tester.HasText("Alice"), "the last records must stay visible")
	require.True(t, tester.HasText("Payload"), "the surviving record keeps its payload control")
	require.False(t, tester.HasText("No module records yet; the first write adds one."))
}

// TestLoadWarningReadsAsAStrip pins the presentation of a failed list read: the
// reason sits beside a mark and the word "Failed" instead of in danger-red body
// copy, and the note about the retained rows is muted rather than red. Colouring
// the paragraph stated the failure twice and made the retained-data line read as
// part of the error; the strip's tint is what separates the two lines now.
func TestLoadWarningReadsAsAStrip(t *testing.T) {
	fake := &fakeServices{
		tokenRows: []TokenRow{{ID: "token-1", Name: "Gold", Symbol: "GLD", TotalSupply: "1000"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1400)
	app.ShowPage("Token")
	tester.Frame()
	fake.loadErrs.tokens = errors.New("database is locked")
	app.refresh()
	tester.Frame()

	mark, ok := tester.Find("Failed")
	require.True(t, ok, "the strip must state the outcome in words, not only in colour")
	reason, ok := tester.Find("Failed to load tokens: database is locked")
	require.True(t, ok)
	require.GreaterOrEqual(t, reason.X, mark.X, "the reason follows the outcome mark")
	stale, ok := tester.Find("Showing the last successful load.")
	require.True(t, ok)
	require.Greater(t, stale.Y, reason.Y, "the retained-data note sits under the reason")

	// The tint is the strip's own surface: it starts lit at the left edge and
	// fades before the card's right edge, so the warning reads as a band rather
	// than as a paragraph of coloured text.
	img := tester.Image()
	luma := func(x, y int) int {
		r, g, b, _ := img.At(x, y).RGBA()
		return int(r>>8) + int(g>>8) + int(b>>8)
	}
	y := int(reason.Y) + int(reason.H)/2
	require.Greater(t, luma(int(mark.X), y), luma(int(mark.X)+420, y),
		"the danger wash must weaken toward the far edge")
}

// TestLoadWarningStaysInsideTheCardAtLargeText keeps a long failure reason from
// pushing the warning past its card at the largest text scale, where the reason
// and the retained-data note wrap instead of being clipped.
func TestLoadWarningStaysInsideTheCardAtLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		ui.Column(c).Width(560).Padding(space4).Children(func() {
			loadWarning(c, "Failed to load tokens", errors.New("database is locked"), true)
		})
	}, 640, 400)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	reason, ok := tester.Find("Failed to load tokens: database is locked")
	require.True(t, ok)
	require.LessOrEqual(t, reason.X+reason.W, float32(640),
		"the reason must wrap inside the card")
	require.True(t, tester.HasText("Showing the last successful load."))
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

// TestFailedVotingReadsReportEachCauseOnce covers the page warning's text. All
// three list reads fail with the same cause when the database is locked, and
// errors.Join renders each of them, so the strip painted the identical sentence
// three times under a heading that already named the module. The dedup lives in
// loadWarning, which the strip reaches through a Label rather than a text run,
// so the assertion is on the message it builds.
func TestFailedVotingReadsReportEachCauseOnce(t *testing.T) {
	boom := errors.New("database is locked")
	app := New(&fakeServices{loadErrs: fakeLoadErrors{
		voters: boom, candidates: boom, sessions: boom,
	}})
	tester := ui.NewTester(app.View, 1280, 1400)
	app.ShowPage("Voting")
	tester.Frame()

	// The model keeps every cause, so a caller reading loadErr still learns all
	// three reads failed.
	require.Equal(t, 3, strings.Count(app.voting.loadErr.Error(), "database is locked"))
	// What the strip draws states it once.
	require.Equal(t, "Failed to load voting data: database is locked",
		dedupeLines("Failed to load voting data: "+localizeServiceError(app.voting.loadErr)))
	require.True(t, tester.HasText("Failed to load voting data"))

	// Different causes are different facts: every one has to survive, because
	// the warning is the only place a partial read failure is reported.
	app = New(&fakeServices{loadErrs: fakeLoadErrors{
		voters:     errors.New("voters table missing"),
		candidates: errors.New("candidates table missing"),
		sessions:   errors.New("sessions table missing"),
	}})
	tester = ui.NewTester(app.View, 1280, 1400)
	app.ShowPage("Voting")
	tester.Frame()
	message := dedupeLines("Failed to load voting data: " + localizeServiceError(app.voting.loadErr))
	for _, reason := range []string{"voters table missing", "candidates table missing", "sessions table missing"} {
		require.Containsf(t, message, reason, "the strip must keep %q", reason)
	}
}

// TestDedupeLinesKeepsTheFirstOfEachAndItsOrder pins the helper itself, because
// it is what every load warning now goes through.
func TestDedupeLinesKeepsTheFirstOfEachAndItsOrder(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"one reason", "one reason"},
		{"a\na\na", "a"},
		{"a\nb\na", "a\nb"},
		{"heading: a\na", "heading: a"},
		{"a\n\na", "a"},
		{"heading: a\nb", "heading: a\nb"},
		{"", ""},
	} {
		require.Equalf(t, tc.want, dedupeLines(tc.in), "dedupeLines(%q)", tc.in)
	}
}

// TestFailedDetailReadKeepsTheLastValue covers the detail reads the list-level test
// above cannot reach: a token's info and balance, a token's history, an NFT's
// details and history, an oracle reading, the latest value, feed health and the
// session roster. None of the fake's error hooks for these was ever set by a test,
// so each card's "keep the last value and say why" path was unexercised.
//
// A detail read is the case where losing the value matters most: the card holds one
// number, and a blank readout beside a failure says the account is empty.
func TestFailedDetailReadKeepsTheLastValue(t *testing.T) {
	boom := errors.New("database is locked")
	fake := &fakeServices{
		tokenRows: []TokenRow{{ID: "t1", Name: "Gold", Symbol: "GLD", TotalSupply: "100", Owner: "owner-pk"}},
		sources:   []SourceRow{{ID: "s1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true}},
	}
	fake.loadErrs.oracleLatestRow = &DataRow{Value: "42"}
	app := New(fake)
	tester := ui.NewTester(app.View, 1400, 7000)

	// Load the balances and readings once, successfully.
	app.ShowPage("Token")
	app.token.tokenID, app.token.owner = "t1", "owner-pk"
	app.loadTokenInfo("t1")
	require.NoError(t, app.loadTokenBalance("t1", "owner-pk"))
	app.ShowPage("Oracle")
	app.oracle.queryID = "s1"
	require.NoError(t, app.loadOracleLatest("s1"))
	tester.Frame()

	before := app.token.balance
	require.NotEmpty(t, before, "the fixture must load a balance, or this proves nothing")

	// Now every detail read fails; the values must survive.
	fake.loadErrs.tokenBalance = boom
	fake.loadErrs.tokenInfo = boom
	fake.loadErrs.oracleLatest = boom
	_ = app.loadTokenBalance("t1", "owner-pk")
	app.loadTokenInfo("t1")
	_ = app.loadOracleLatest("s1")

	require.Equal(t, before, app.token.balance,
		"a failed balance read must keep the number it showed")
	require.NotNil(t, app.token.info,
		"a failed info read must keep the token it showed")
	require.NotNil(t, app.oracle.latest,
		"a failed latest read must keep the value it showed")

	// And each card says the read failed rather than showing a value as current.
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("database is locked"))
	require.True(t, tester.HasText("Showing the last successful load."))
}
