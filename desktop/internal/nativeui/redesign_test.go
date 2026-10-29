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
	require.True(t, tester.HasText(tr("↑↓ move · Enter open · Esc close")),
		"the palette must document its keys where they are used")

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
// the first fact on the page, the counters follow it, and the action shortcuts
// stay last for a store that already has content.
func TestDashboardHeroLeadsWithIntegrity(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "lottery", Summary: "draw"}}})
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.Frame()

	verdict, ok := tester.Find("Ledger Integrity")
	require.True(t, ok)
	counters, ok := tester.Find("Total Lotteries")
	require.True(t, ok)
	quick, ok := tester.Find("Quick start")
	require.True(t, ok)
	require.Less(t, verdict.Y, counters.Y, "the verdict must lead the counters")
	require.Less(t, counters.Y, quick.Y, "the shortcuts stay below the state")
	require.True(t, tester.HasText("verified"))
	require.True(t, tester.HasText("Inspect ledger"))
}

// TestDashboardCountsChainBlocks covers the sixth counter: token and NFT
// writes have no module counter of their own, so the chain's entry count is
// the only summary that includes them.
func TestDashboardCountsChainBlocks(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	require.True(t, tester.HasText("Ledger Blocks"))
	require.True(t, tester.HasText("Entries on the audit chain"))
	require.True(t, tester.HasText("7"), "the count must come from the services")
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
