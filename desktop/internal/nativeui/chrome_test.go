package nativeui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPageCardsKeepTheirOwnCorners pins the page rhythm. The page body used to
// stack cards with no gap at all: a card's rounded corner met the next card's
// corner and the two read as one slab with a seam, which is why the dashboard
// looked like a single panel with lines through it.
func TestPageCardsKeepTheirOwnCorners(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Lottery")
	tester.Frame()

	create, ok := tester.Find("Create Draw")
	require.True(t, ok)
	history, ok := tester.Find("History")
	require.True(t, ok)
	require.GreaterOrEqualf(t, history.Y-(create.Y+create.H), pageGap-1,
		"the page must leave pageGap between the two halves of the body")
}

// TestHeaderIsABandAboveTheBody pins the fixed chrome's own surface. The
// header used to be the first row of the scrolling column, identical on every
// module; the band gives it the module's tint and a hairline that separates it
// from the body, and the body keeps its own padding below that line.
func TestHeaderIsABandAboveTheBody(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1200)
	app.ShowPage("Token")
	tester.Frame()

	// The purpose line is unique to the page header; the title alone would
	// match the sidebar's own row.
	subtitle, ok := tester.Find(pagePurpose[pageToken])
	require.True(t, ok)
	guide, ok := tester.Find("How this works")
	require.True(t, ok)
	require.GreaterOrEqualf(t, guide.Y-(subtitle.Y+subtitle.H), space5,
		"the body must start below the header band, not inside it")

	// The band is part of the fixed chrome: scrolling the body must not move
	// it, or the page title would leave the window while its forms scroll.
	before := subtitle.Y
	tester.Scroll(700, 500, 0, 150)
	tester.Frame()
	after, ok := tester.Find(pagePurpose[pageToken])
	require.True(t, ok)
	require.InDelta(t, float64(before), float64(after.Y), 0.5,
		"the header band must stay in place while the body scrolls")
}

// TestClosedGuideIsOneQuietLine pins the guide's collapsed weight. It used to
// be a bordered card with its own tile, as heavy as the first real card on the
// page; collapsed it is now a single line of guidance, and opening it still
// shows every ordered step.
func TestClosedGuideIsOneQuietLine(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1200)
	app.ShowPage("NFT")
	tester.Frame()

	summary, ok := tester.Find("How this works")
	require.True(t, ok)
	require.LessOrEqual(t, summary.H, float32(40),
		"the collapsed guide must stay a single line")

	before := summary.Y
	require.NoError(t, tester.Click("How this works"))
	tester.Frame()
	step, ok := tester.Find("Mint an asset with a creator key, or select one in Collection. The creator is the first owner.")
	require.True(t, ok, "opening the guide must show its steps")
	require.Greater(t, step.Y, before, "the steps open below the summary")
}

// TestDataTablesSitInTheirOwnFrame pins the table surface. A grid drawn
// directly on a card's gradient had no edge of its own, so its header rule and
// last row floated in the card's padding and a one-row table looked like a
// stray line of text. The frame is what makes the rows read as one object.
func TestDataTablesSitInTheirOwnFrame(t *testing.T) {
	app := New(&fakeServices{tokenRows: []TokenRow{
		{ID: "token-1", Name: "Gold", Symbol: "GLD", TotalSupply: "1000"},
	}})
	tester := ui.NewTester(app.View, 1280, 1400)
	app.ShowPage("Token")
	tester.Frame()

	// The table's accessible name shares the card's box; the grid itself is
	// the element the frame is painted on, so its rows must stay inside it.
	id, ok := tester.Find("token-1")
	require.True(t, ok)
	table, ok := tester.Find("Token collection")
	require.True(t, ok)
	require.GreaterOrEqual(t, id.X, table.X,
		"the first cell must start inside the framed table")
	require.LessOrEqual(t, id.X+id.W, table.X+table.W,
		"the row must end inside the framed table")
}

// TestSectionRuleFadesOut pins the separators above a band of cards. A
// full-width border under "At a glance" ran across the page like a table edge;
// the rule's right end must reach the background, so the heading groups what
// follows instead of closing a region.
func TestSectionRuleFadesOut(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(600).Padding(20).Children(func() {
			sectionRule(c, "At a glance")
		})
	}, 640, 120)
	tester.Frame()
	img := tester.Image()

	label, ok := tester.Find("At a glance")
	require.True(t, ok)
	row := int(label.Y + label.H/2)
	background := img.At(300, int(label.Y-10))
	mid := img.At(300, row)
	end := img.At(600, row)
	luma := func(c color.Color) int {
		r, g, b, _ := c.RGBA()
		return int(r>>8) + int(g>>8) + int(b>>8)
	}
	// The rule lightens the background beside the label and fades to it at
	// the far edge: the near sample must differ more than the far one.
	require.Greater(t, absInt(luma(mid)-luma(background)), absInt(luma(end)-luma(background)),
		"the rule must be visible beside its label and gone at the far edge")
}

// TestTableHeaderKeepsItsRowHeight pins the accidental 68px header: the table
// frame was drawn past the grid's own box, which grew the header row and left
// the body's first row floating in the middle of the frame.
func TestTableHeaderKeepsItsRowHeight(t *testing.T) {
	app := New(&fakeServices{draws: []DrawRow{
		{ID: "draw-1", BlockHeight: 7, Participants: []string{"A"}, Winners: []string{"A"}},
	}})
	tester := ui.NewTester(app.View, 1280, 1600)
	app.ShowPage("Lottery")
	tester.Frame()

	header, ok := tester.Find("Participants")
	require.True(t, ok)
	row, ok := tester.Find("draw-1")
	require.True(t, ok)
	require.Less(t, header.Y+header.H, row.Y,
		"the header must end above the first row")
	require.LessOrEqual(t, row.Y-(header.Y+header.H), space6+2,
		"the rule between header and rows must not leave a band of empty frame")
}

// absInt is the absolute value of n, for pixel comparisons.
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// TestLedgerRecordsShareOneColumn pins the ledger's record stream: each entry
// is a full-width record in one column, newest first, with the next record
// below it. The page used to be a three-column table whose middle cell was the
// raw payload; a record's sentence and facts need the full width, and a grid
// would squeeze the facts into the payload column again.
func TestLedgerRecordsShareOneColumn(t *testing.T) {
	records := ledgerRecords()
	app := New(&fakeServices{blocks: records})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()

	// Each record is a full-width card, so the next one begins below the
	// previous one rather than beside it: the table this replaced put the three
	// pieces of an entry in three columns.
	newest, ok := tester.Find(blockRecordLabel(records[0]))
	require.True(t, ok)
	older, ok := tester.Find(blockRecordLabel(records[1]))
	require.True(t, ok)
	require.Equal(t, newest.X, older.X, "records stack on one left edge")
	require.Equal(t, newest.W, older.W, "records share the card width")
	require.Greater(t, older.Y, newest.Y, "the newest record comes first")
}

// TestLedgerRecordNamesItsModuleInTheWindowsLanguage pins the record's
// provenance mark: the chain stores a lowercase identifier ("lottery"), while
// the pill shows the module's own page name in the current language, so a
// Chinese window says 抽奖 rather than "lottery" beside 抽奖 in the sidebar.
func TestLedgerRecordNamesItsModuleInTheWindowsLanguage(t *testing.T) {
	app := New(&fakeServices{blocks: ledgerRecords()})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Blockchain")
	tester.Frame()

	require.True(t, tester.HasText(tr(pageTitles[pageLottery])),
		"the lottery record must name its module in the window's language")
	require.True(t, tester.HasText(tr(pageTitles[pageToken])),
		"the token record must name its module in the window's language")
}
