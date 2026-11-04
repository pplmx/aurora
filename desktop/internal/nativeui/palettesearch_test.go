package nativeui

import (
	"errors"
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
