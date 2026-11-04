package nativeui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

func searchFake() *fakeServices {
	return &fakeServices{blocks: []BlockRow{
		{Height: 5, Module: "lottery", Action: "Draw", Subject: "Alice, Carol", Summary: "alice"},
		{Height: 4, Module: "token", Action: "Mint", Subject: "GLD", Summary: "gld"},
	}}
}

// TestPaletteFindsARecordByItsText covers the case the palette was missing: an
// operator remembers a word from a record — a winner's name, a token's symbol —
// and not its id. Before this, finding it meant opening Blockchain, typing the
// word there, and then working out which module the result belonged to.
func TestPaletteFindsARecordByItsText(t *testing.T) {
	app := New(searchFake())
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "alice"
	tester.Frame()
	matches := app.paletteMatches()
	require.NotEmpty(t, matches)
	require.Equal(t, "lottery · Draw", matches[0].title,
		"the record's module and action are its headline")

	// The entry opens the module that wrote the record.
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}

// TestPaletteSearchRunsOncePerEdit keeps a per-frame rebuild from turning into a
// per-frame database read. The palette rebuilds on every frame, so a search that
// ran there would query storage sixty times a second for one keystroke.
func TestPaletteSearchRunsOncePerEdit(t *testing.T) {
	fake := searchFake()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "alice"
	app.paletteMatches()
	after := fake.blockListCalls
	require.Greater(t, after, 0, "the first read of a query must reach storage")

	for range 5 {
		app.paletteMatches()
		tester.Frame()
	}
	require.Equal(t, after, fake.blockListCalls,
		"an unchanged query must be answered from the cached rows")

	app.paletteQuery = "gld"
	app.paletteMatches()
	require.Greater(t, fake.blockListCalls, after, "an edited query is a new search")
}

// TestPaletteSearchWaitsForOneCharacter: a single character matches most of the
// chain, so the first keystroke would issue a query whose whole answer is
// "everything" — a read whose cost buys nothing.
func TestPaletteSearchWaitsForOneCharacter(t *testing.T) {
	fake := searchFake()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	baseline := fake.blockListCalls
	app.paletteQuery = "a"
	app.paletteMatches()
	require.Equal(t, baseline, fake.blockListCalls, "one character must not query the ledger")

	app.paletteQuery = "al"
	app.paletteMatches()
	require.Greater(t, fake.blockListCalls, baseline, "two characters are a query")
}

// TestPaletteSearchStaysOutOfADatabaseWrite keeps the record search behind the
// same gate as every other database read in the window: while a write owns the
// database, nothing may start a second operation on it.
func TestPaletteSearchStaysOutOfADatabaseWrite(t *testing.T) {
	fake := searchFake()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()
	baseline := fake.blockListCalls

	app.busy = "Importing lotteries…"
	app.paletteQuery = "alice"
	app.paletteMatches()
	require.Equal(t, baseline, fake.blockListCalls, "a search must not read during a write")
}

// TestPaletteSearchFailureOffersNoRecords: a failed search contributes no rows
// and leaves the pages and commands alone. Silently dropping the group is right
// here — the palette still answers what it can, and the failure is a transient
// database condition the reader will see reported if they open Blockchain.
func TestPaletteSearchFailureOffersNoRecords(t *testing.T) {
	fake := searchFake()
	fake.loadErrs.blocks = errors.New("database is locked")
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "alice"
	matches := app.paletteMatches()
	require.Empty(t, matches, "a failed search contributes no rows")
	require.Empty(t, app.paletteRecordEntries("alice"))
}

// TestPaletteSearchSkipsAnIdJump keeps the two gestures apart: "token:<id>" is an
// exact jump and must not also cost a ledger search for the literal text.
func TestPaletteSearchSkipsAnIdJump(t *testing.T) {
	fake := searchFake()
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()
	baseline := fake.blockListCalls

	app.paletteQuery = "token:tok-9"
	matches := app.paletteMatches()
	require.Len(t, matches, 1, "the jump is the only entry")
	require.Equal(t, baseline, fake.blockListCalls, "and it must not run a ledger search")
}

// TestTruncatedSearchSaysSoAndOffersTheRest covers what a cut-short list owes the
// reader. The palette offers a handful of records because it is a jump target, but
// it drew them without saying the chain held more: a query matching forty records
// showed five, and the reader concluded their record was not in the chain rather
// than that the palette showed part of it.
func TestTruncatedSearchSaysSoAndOffersTheRest(t *testing.T) {
	rows := make([]BlockRow, 20)
	for i := range rows {
		rows[i] = BlockRow{
			Height: int64(20 - i), Module: "token", Action: "Mint",
			Summary: fmt.Sprintf("searchable-%d", i),
		}
	}
	app := New(&fakeServices{blocks: rows})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "searchable"
	tester.Frame()
	entries := app.paletteRecordEntries("searchable")
	require.Len(t, entries, paletteSearchLimit+1,
		"the capped rows plus the entry that discloses the rest")
	last := entries[len(entries)-1]
	require.Equal(t, "Show all 20 matching records", last.title)

	// The disclosure is a way out, not a label: it opens the ledger with the
	// query already in its search field, where the pager lives.
	app.showLedgerSearch("searchable")
	require.Equal(t, pageBlockchain, app.page)
	require.Equal(t, "searchable", app.blockchain.search)
}

// TestShortSearchListSaysNothing: a result set that fits needs no disclosure. An
// always-present "show all" row would be one more thing to skip past on every
// successful search.
func TestShortSearchListSaysNothing(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{
		{Height: 2, Module: "token", Action: "Mint", Summary: "searchable"},
	}})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "searchable"
	entries := app.paletteRecordEntries("searchable")
	require.Len(t, entries, 1)
	require.Equal(t, "token · Mint", entries[0].title)
}

// TestEveryMatchingModuleIsRepresented covers the case a newest-first slice gets
// wrong. Six writes from one module and three from another all match the query;
// taking the five newest rows showed five records from the first module and none
// from the second, so a reader whose match was in the second module saw a list
// with none of their records in it — while the ledger total said eight matched.
func TestEveryMatchingModuleIsRepresented(t *testing.T) {
	rows := make([]BlockRow, 0, 9)
	for i := range 6 {
		rows = append(rows, BlockRow{
			Height: int64(20 - i), Module: "token", Action: "Mint", Summary: "shared",
		})
	}
	for i := range 3 {
		rows = append(rows, BlockRow{
			Height: int64(10 - i), Module: "oracle", Action: "Observation", Summary: "shared",
		})
	}

	app := New(&fakeServices{blocks: rows})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "shared"
	entries := app.paletteRecordEntries("shared")

	var tokens, oracles int
	for _, entry := range entries {
		switch entry.title {
		case "token · Mint":
			tokens++
		case "oracle · Observation":
			oracles++
		}
	}
	require.Greater(t, oracles, 0, "a module that matched must appear in the list")
	require.Greater(t, tokens, 0, "and so must the other one")
	require.Len(t, entries, paletteSearchLimit+1,
		"the cap still holds: balanced rows plus the entry that discloses the rest")
}

// TestBalancedResultsKeepTheOrder shows the same search twice, which is what a
// reader does after glancing away. A balancing that shuffled rows would make the
// list move under them.
func TestBalancedResultsKeepTheOrder(t *testing.T) {
	rows := []BlockRow{
		{Height: 9, Module: "token", Action: "Mint", Summary: "shared"},
		{Height: 8, Module: "token", Action: "Mint", Summary: "shared"},
		{Height: 7, Module: "token", Action: "Mint", Summary: "shared"},
		{Height: 6, Module: "oracle", Action: "Observation", Summary: "shared"},
		{Height: 5, Module: "oracle", Action: "Observation", Summary: "shared"},
		{Height: 4, Module: "token", Action: "Mint", Summary: "shared"},
	}
	first := balanceByModule(rows, 4)
	second := balanceByModule(rows, 4)
	require.Equal(t, first, second, "the same rows must produce the same list")

	// Every module appears before any module takes a second slot.
	require.Equal(t, "token", first[0].Module)
	require.Equal(t, "oracle", first[1].Module)
}

// TestRecordRowsCarryTheirModuleHue covers how a mixed list is scanned. Two
// records from different modules read as one kind of row unless the icon says
// which module each came from, and the ledger's rows already use that hue. Pages
// and commands keep the accent: they belong to no module, and colouring them
// would claim they do.
func TestRecordRowsCarryTheirModuleHue(t *testing.T) {
	fake := &fakeServices{blocks: []BlockRow{
		{Height: 5, Module: "token", Action: "Mint", Summary: "shared"},
		{Height: 4, Module: "oracle", Action: "Observation", Summary: "shared"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	app.openPalette()
	tester.Frame()

	app.paletteQuery = "shared"
	entries := app.paletteRecordEntries("shared")
	require.Len(t, entries, 2)
	for _, entry := range entries {
		require.Truef(t, entry.hasAccent, "%q must take its module's hue", entry.title)
	}
	require.Equal(t, pageToken, entries[0].accent)
	require.Equal(t, pageOracle, entries[1].accent)

	// A page entry, matched by its own name, takes the accent instead.
	app.paletteQuery = "dashboard"
	for _, entry := range app.paletteMatches() {
		if entry.page == pageDashboard {
			require.False(t, entry.hasAccent, "a page belongs to no module")
		}
	}
}
