package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPaletteButtonOpensTheSwitcher covers the pointer entry point the
// keyboard shortcut used to be alone with: the sidebar control opens the same
// palette, and Escape closes it without leaving the window on a dead dialog.
func TestPaletteButtonOpensTheSwitcher(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	require.True(t, tester.HasText("Jump to"), "the sidebar must show its palette control")
	require.True(t, tester.HasText(shortcutChord("K")), "the control must print its chord")
	require.NoError(t, tester.Click("Jump to"))
	tester.Frame()
	require.True(t, app.paletteOpen, "clicking the control must open the palette")
	// The keys are printed as caps with their action beside them, so the
	// arrow/Enter/Escape model is learned where it is used.
	for _, hint := range []string{"↑↓", "Enter", "Esc"} {
		require.Truef(t, tester.HasText(hint),
			"the palette must print the %q key where it is used", hint)
	}
	require.True(t, tester.HasText(tr("move")), "the palette must name what the arrows do")

	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	require.False(t, app.paletteOpen, "Escape must close the palette")
}

// TestSettingsCardsUseTwoColumns keeps the preferences page from leaving half
// the window empty: the cards share two columns at the default width and stack
// below the same threshold as every other two-column page.
func TestSettingsCardsUseTwoColumns(t *testing.T) {
	app := New(&fakeServices{})
	wide := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Settings")
	wide.Frame()

	language, ok := wide.Find("Language")
	require.True(t, ok)
	sessionKeys, ok := wide.Find("Session keys")
	require.True(t, ok)
	require.NotEqual(t, language.X, sessionKeys.X, "wide Settings cards sit side by side")
	require.InDelta(t, float64(language.Y), float64(sessionKeys.Y), 2, "the columns start together")

	narrowApp := New(&fakeServices{})
	narrow := ui.NewTester(narrowApp.View, 900, 5000)
	narrowApp.ShowPage("Settings")
	narrow.Frame()

	language, ok = narrow.Find("Language")
	require.True(t, ok)
	sessionKeys, ok = narrow.Find("Session keys")
	require.True(t, ok)
	require.Equal(t, language.X, sessionKeys.X, "a narrow window stacks the Settings cards")
	require.Greater(t, sessionKeys.Y, language.Y, "the second column follows the first")
}

// TestDashboardHeroLeadsWithIntegrity pins the reading order: the verdict is
// the first fact on the page, the module composition follows it, the timeline
// explains what changed, and the action shortcuts stay last for a store that
// already has content.
func TestDashboardHeroLeadsWithIntegrity(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "lottery", Summary: "draw"}}})
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.Frame()

	verdict, ok := tester.Find("Ledger Integrity")
	require.True(t, ok)
	composition, ok := tester.Find("What the ledger is made of")
	require.True(t, ok)
	recent, ok := tester.Find("Recent activity")
	require.True(t, ok)
	quick, ok := tester.Find("Quick start")
	require.True(t, ok)
	require.Less(t, verdict.Y, composition.Y, "the verdict must lead the composition")
	require.Less(t, composition.Y, recent.Y, "the composition must lead the timeline")
	require.Less(t, recent.Y, quick.Y, "the shortcuts stay below the state")
	require.True(t, tester.HasText("verified"))
	require.True(t, tester.HasText("Open ledger"), "the verdict keeps one route to the full ledger")
	require.False(t, tester.HasText("Inspect ledger"), "the hero must not keep a second ledger route")
}

// TestDashboardCountsChainBlocks keeps the chain total visible even though
// token and NFT writes have no module counter of their own.
func TestDashboardCountsChainBlocks(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	require.True(t, tester.HasText("Ledger Blocks"))
	require.True(t, tester.HasText("Entries on the audit chain"))
	require.True(t, tester.HasText("7"), "the count must come from the services")
}

// TestDashboardCompositionIsMoreThanACounterGrid pins the new information
// architecture. The page leads with one integrity panel, then shows the
// modules as unequal parts of one ledger, then a timeline. Six equal metric
// cards made every fact compete with every other fact.
func TestDashboardCompositionIsMoreThanACounterGrid(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 2400)
	tester.Frame()

	for _, want := range []string{
		"Ledger composition",
		"What the ledger is made of",
		"Module records first; the audit chain carries all of them.",
		"Voting detail",
		"Total Votes",
		"Candidates",
		"Voting Sessions",
	} {
		require.Truef(t, tester.HasText(want), "the dashboard composition must show %q", want)
	}
	require.False(t, tester.HasText("At a glance"), "the equal-card section is gone")
}

// TestDashboardTimelineShowsSequence pins the event stream: every recent
// record gets a module-coloured point on one spine, and the record itself
// remains the navigation target.
func TestDashboardTimelineShowsSequence(t *testing.T) {
	app := New(&fakeServices{blocks: captureBlocks()})
	tester := ui.NewTester(app.View, 1280, 2400)
	tester.Frame()

	first, ok := tester.Find("Alice, Carol")
	require.True(t, ok)
	second, ok := tester.Find("GLD")
	require.True(t, ok)
	require.Greater(t, second.Y, first.Y, "the timeline must run newest to oldest")

	require.NoError(t, tester.Click("Alice, Carol"))
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}

// TestDashboardCompositionStacksAtMinimumWidth keeps the module rail usable at
// the 900px minimum instead of pushing its fourth card off the window.
func TestDashboardCompositionStacksAtMinimumWidth(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 900, 2600)
	tester.Frame()

	lottery, ok := tester.Find("Total Lotteries")
	require.True(t, ok)
	voting, ok := tester.Find("Votes, candidates and sessions")
	require.True(t, ok)
	oracle, ok := tester.Find("Registered data feeds")
	require.True(t, ok)
	ledger, ok := tester.Find("Entries on the audit chain")
	require.True(t, ok)
	require.Greater(t, voting.Y, lottery.Y, "the narrow rail stacks in module order")
	require.Greater(t, oracle.Y, voting.Y, "the oracle follows voting")
	require.Greater(t, ledger.Y, oracle.Y, "the audit chain stays last")
	require.LessOrEqual(t, oracle.X+oracle.W, float32(900),
		"every module tile must stay inside the window")
}

// TestDashboardSharesUseTheVotingTotal keeps voting's three scales in the
// denominator. Before, voting always reached the largest rail while a smaller
// ledger showed a shorter one, even though both sums were stored totals.
func TestDashboardSharesUseTheVotingTotal(t *testing.T) {
	stats := DashboardStats{
		Lotteries:  2,
		Votes:      9,
		Candidates: 2,
		Sessions:   1,
		Oracle:     3,
		Blocks:     5,
	}
	require.Equal(t, float32(100), dashboardShare(pageVoting, stats))
	require.Less(t, dashboardShare(pageBlockchain, stats), float32(100),
		"the ledger rail must not tie voting when voting is the larger stored total")
	require.Less(t, dashboardShare(pageOracle, stats), dashboardShare(pageBlockchain, stats))

	ledgerHeavy := DashboardStats{Lotteries: 1, Oracle: 1, Blocks: 100}
	require.Equal(t, float32(10), dashboardShare(pageVoting, ledgerHeavy),
		"an empty voting rail must stay at its visible floor")
	require.Equal(t, float32(100), dashboardShare(pageBlockchain, ledgerHeavy))
}

// TestDashboardTimelineSpineMatchesEveryRecord keeps the vertical rule tied to
// the row it marks. A fixed 52px segment ran through one-line rows and left
// gaps below wrapped records, so the sequence stopped reading as one spine.
func TestDashboardTimelineRowsContinueAtLargeText(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{
		{
			Height: 2, Module: "oracle", Action: "Observation", Subject: "68421.50",
			Facts: []BlockFact{
				{Label: "Value", Value: "68421.50"},
				{Label: "Source", Value: "source-1"},
				{Label: "Observed", Value: "2026-10-30T09:15:00+08:00"},
			},
		},
		{Height: 1, Module: "lottery", Action: "Draw", Subject: "Alice"},
	}})
	tester := ui.NewTester(app.View, 900, 700)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find("68421.50")
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-260)
	tester.Frame()
	first, ok = tester.Find("68421.50")
	require.True(t, ok)
	second, ok := tester.Find("Alice")
	require.True(t, ok)
	require.Greater(t, second.Y, first.Y+first.H,
		"the next record must follow the wrapped row")
}

// TestDashboardHeroKeepsVerdictWholeAtLargeText catches the regression where
// the action squeezed the verdict row until "verified" and its badge clipped.
func TestDashboardHeroKeepsVerdictWholeAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, want := range []string{"verified", "Hash chain", "Open ledger"} {
		require.Truef(t, tester.HasText(want), "the hero must keep %q readable", want)
	}
	verdict, ok := tester.Find("verified")
	require.True(t, ok)
	badge, ok := tester.Find("Hash chain")
	require.True(t, ok)
	action, ok := tester.Find("Open ledger")
	require.True(t, ok)
	require.LessOrEqual(t, verdict.X+verdict.W, float32(1280))
	require.LessOrEqual(t, badge.X+badge.W, float32(1280))
	require.Greater(t, action.Y, verdict.Y+verdict.H,
		"the ledger route moves below the verdict when text is enlarged")
}

// TestStatChipsKeepFactsBesideTheirCollection pins the ledger page's compact
// summary: two tiles and the verdict share a row above the controls they
// describe, instead of a full-width readout for two short numbers.
func TestStatChipsKeepFactsBesideTheirCollection(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "voting", Summary: "ballot"}}})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Blockchain")
	tester.Frame()

	blocks, ok := tester.Find("Blocks")
	require.True(t, ok)
	height, ok := tester.Find("Height")
	require.True(t, ok)
	verdict, ok := tester.Find("Hash chain verified")
	require.True(t, ok)
	require.InDelta(t, float64(blocks.Y), float64(height.Y), 2, "the tiles share a line")
	require.InDelta(t, float64(height.Y), float64(verdict.Y), 16, "the verdict sits with its facts")

	search, ok := tester.Find("Search blocks")
	require.True(t, ok)
	require.Less(t, verdict.Y, search.Y, "the summary must precede the controls")
}

// TestFreshDashboardExplainsTheEmptyChain covers the first-run wording: a
// verified genesis chain is ready for work, not a summary of nothing, so the
// hero names the next move instead of repeating the hash-chain explanation.
func TestFreshDashboardExplainsTheEmptyChain(t *testing.T) {
	app := New(emptyDashboard{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 1000)
	tester.Frame()

	require.True(t, tester.HasText("The ledger is ready. Start with a quick action below; the first record opens block 1."))
	require.True(t, tester.HasText("Nothing on the chain yet."))
	require.True(t, tester.HasText("Run a draw"))
}

// TestFieldPairKeepsTwoControlsOnOneLine holds the new two-up form row: the
// labels share a baseline and the inputs share a width, so a short-value form
// reads as one row instead of two full-width fields.
func TestFieldPairKeepsTwoControlsOnOneLine(t *testing.T) {
	var a, b string
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(520).Padding(20).Children(func() {
			fieldPair(c, "Left", &a, "Right", &b)
		})
	}, 560, 200)
	tester.Frame()

	left, ok := tester.Find("Left")
	require.True(t, ok)
	right, ok := tester.Find("Right")
	require.True(t, ok)
	require.InDelta(t, float64(left.Y), float64(right.Y), 1, "paired labels share a line")
	require.Greater(t, right.X, left.X, "the second control follows the first")
}

// TestTallyBarsKeepExactVotesAccessible holds the two readings together: the
// bar is a comparison, and each row's accessible label carries the exact
// candidate and count so the picture is never the only source of the numbers.
func TestTallyBarsKeepExactVotesAccessible(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1200)
	app.ShowPage("Voting")
	app.voting.tab = 2
	app.voting.resultsSession = "session-1"
	app.voting.resultsLoadedID = "session-1"
	app.voting.results = []ResultRow{
		{CandidateID: "c1", Name: "Alice", Votes: 7},
		{CandidateID: "c2", Name: "Bob", Votes: 6},
	}
	tester.Frame()

	for _, label := range []string{"Alice: 7 votes", "Bob: 6 votes"} {
		_, ok := tester.Find(label)
		require.Truef(t, ok, "the tally row must expose %q, not only a bar", label)
	}
	require.True(t, tester.HasText("Total votes"))
}

// TestOracleActionsStayAvailableOnEveryStage pins the moved workbench: the
// source actions and the selected feed's details live in the left column, so
// an operator reading Health or Readings can still enable, fetch or delete the
// feed without switching back to Sources. The delete confirmation is mounted
// once for the page, so it must also appear over either stage.
func TestOracleActionsStayAvailableOnEveryStage(t *testing.T) {
	app := New(&fakeServices{sources: []SourceRow{{
		ID: "s1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true, Interval: 60,
	}}})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Oracle")
	tester.Frame()
	require.True(t, tester.HasText("Source Actions"))

	app.oracle.tab = 1
	tester.Frame()
	require.True(t, tester.HasText("Source Actions"), "Health must keep the actions visible")
	require.NoError(t, tester.Click("Enable"))
	tester.Frame()

	app.oracle.tab = 2
	tester.Frame()
	require.True(t, tester.HasText("Source Actions"), "Readings must keep the actions visible")

	app.oracle.fetchID = "s1"
	app.oracle.tab = 1
	tester.Frame()
	require.NoError(t, tester.Click("Delete"))
	tester.Frame()
	require.True(t, tester.HasText("Delete btc (s1)?"),
		"the page-level confirmation must render over a non-Sources stage")
}

// TestVerifyCardOffersTheLatestDraw covers the shortcut beside Verify: the
// History page already holds the draw just recorded, so re-checking it must not
// require copying its id out of the table.
func TestVerifyCardOffersTheLatestDraw(t *testing.T) {
	app := New(&fakeServices{
		draws:        []DrawRow{{ID: "draw-7", BlockHeight: 7}, {ID: "draw-6", BlockHeight: 6}},
		verifyResult: VerifyResult{ID: "draw-7", Valid: true},
	})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Lottery")
	tester.Frame()

	require.NoError(t, tester.Click("Use latest draw"))
	tester.Frame()
	require.Equal(t, "draw-7", app.lottery.verifyID)
	require.NotNil(t, app.lottery.verifyResult, "the latest draw must be verified immediately")

	// The verdict is copyable: a ticket quotes the same words the window shows.
	require.NoError(t, tester.Click("Copy Verdict"))
	tester.Frame()
	require.Equal(t, "Draw ID draw-7: Verified", tester.Clipboard())
	require.True(t, tester.HasText("Verdict is on the clipboard"))
}

// TestOracleStageLeadsInAStackedWindow covers the one page whose first column
// is setup rather than the list its stage acts on: stacked, the tall add form
// used to push the stage selector two screens down, so a click on Oracle showed
// the form and hid Sources, Health and Readings.
func TestOracleStageLeadsInAStackedWindow(t *testing.T) {
	app := New(&fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}})
	tester := ui.NewTester(app.View, 900, 900)
	app.ShowPage("Oracle")
	tester.Frame()

	tab, ok := tester.Find("Sources")
	require.True(t, ok)
	add, ok := tester.Find("Add Source")
	require.True(t, ok)
	require.Less(t, tab.Y, add.Y,
		"the stage selector must come before the add form when the columns stack")
}
