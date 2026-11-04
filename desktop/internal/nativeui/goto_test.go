package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestParseRecordQuery pins the one syntax the palette accepts for a jump. The
// prefixes are the page ids the sidebar and deep links already use, so an
// operator who has seen "token" in a link does not have to learn a second word
// for the same place.
func TestParseRecordQuery(t *testing.T) {
	for _, tc := range []struct {
		query string
		page  page
		id    string
		ok    bool
	}{
		{"token:tok-9", pageToken, "tok-9", true},
		{"  TOKEN : tok-9 ", pageToken, "tok-9", true},
		{"nft:n-1", pageNFT, "n-1", true},
		{"oracle:s-1", pageOracle, "s-1", true},
		{"session:sess-1", pageVoting, "sess-1", true},
		{"lottery:draw-1", pageLottery, "draw-1", true},
		// Ordinary palette text is not a jump.
		{"token", 0, "", false},
		{"token:", 0, "", false},
		{"nonsense:abc", 0, "", false},
		{"", 0, "", false},
	} {
		t.Run(tc.query, func(t *testing.T) {
			p, id, ok := parseRecordQuery(tc.query)
			require.Equal(t, tc.ok, ok)
			if tc.ok {
				require.Equal(t, tc.page, p)
				require.Equal(t, tc.id, id)
			}
		})
	}
}

// TestPaletteJumpsToAPastedRecord covers the gesture: a record id copied into a
// ticket, chat message or colleague's window is pasted into the palette, and
// Enter opens the page with that record as the subject. Before this the palette
// could reach a page but not a record, so a pasted id meant opening the page and
// hunting the row by eye.
func TestPaletteJumpsToAPastedRecord(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "tok-9", Name: "Gold", Symbol: "GLD", TotalSupply: "1"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	tester.Type("token:tok-9")
	tester.Frame()

	matches := app.paletteMatches()
	require.Len(t, matches, 1, "a parsed jump is offered on its own")
	require.Equal(t, "Open Token", matches[0].title)
	require.Equal(t, "tok-9", matches[0].detail, "the row shows which record will open")

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageToken, app.page)
	require.Equal(t, "tok-9", app.token.tokenID,
		"the pasted id must become the page's subject, not just its page")
}

// TestJumpedRecordReachesEveryKind keeps the mapping honest for the collections
// an operator is most likely to cite: an NFT, an oracle source and a voting
// session each have their own page fields, and a jump that set only some of them
// would leave the cards acting on the previous record.
func TestJumpedRecordReachesEveryKind(t *testing.T) {
	fake := &fakeServices{}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	app.ShowRecord("nft", "n-7")
	require.Equal(t, pageNFT, app.page)
	for _, id := range []string{app.nft.id, app.nft.transferID, app.nft.burnID, app.nft.historyID} {
		require.Equal(t, "n-7", id, "every NFT card must act on the jumped record")
	}

	app.ShowRecord("oracle", "src-3")
	require.Equal(t, pageOracle, app.page)
	require.Equal(t, "src-3", app.oracle.fetchID)
	require.Equal(t, "src-3", app.oracle.latestID)

	app.ShowRecord("session", "sess-2")
	require.Equal(t, pageVoting, app.page)
	require.Equal(t, "sess-2", app.voting.resultsSession)
}

// TestJumpToAnUnknownRecordStillMoves: the id may have been deleted on another
// machine. Opening the page and letting its own read report "not found" is more
// useful than a palette that silently refuses to move — the operator at least
// lands where the answer lives.
func TestJumpToAnUnknownRecordStillMoves(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	app.ShowRecord("token", "gone")
	require.Equal(t, pageToken, app.page)
	require.Equal(t, "gone", app.token.tokenID)
}

// TestJumpToAnUnknownKindDoesNothing keeps a typo from navigating somewhere
// arbitrary: an unknown prefix is ordinary palette text, which the filter
// handles, not a jump to the dashboard.
func TestJumpToAnUnknownKindDoesNothing(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	app.ShowPage("Oracle")
	tester.Frame()

	app.ShowRecord("nonsense", "x")
	app.ShowRecord("token", "   ")
	require.Equal(t, pageOracle, app.page, "neither an unknown kind nor an empty id may navigate")
}
