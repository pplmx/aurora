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

// TestJumpLandsWhereAClickWould pins the state a jump produces against the state
// a click produces. A selection sets more than the subject id — the owner context
// the write forms reuse and the row highlight among them — so a jump that only
// wrote the id left the page in a state no click can reach: the write cards knew
// no owner, and the row it had just opened was not marked. A reader notices that
// only after signing with a key the form silently did not fill in.
func TestJumpLandsWhereAClickWould(t *testing.T) {
	rows := []TokenRow{
		{ID: "tok-a", Name: "Alpha", Symbol: "A", TotalSupply: "1", Owner: "owner-a-pk"},
		{ID: "tok-b", Name: "Beta", Symbol: "B", TotalSupply: "1", Owner: "owner-b-pk"},
	}

	clicked := New(&fakeServices{tokenRows: rows})
	clickTester := ui.NewTester(clicked.View, 1280, 1800)
	clicked.ShowPage("Token")
	clickTester.Frame()
	require.NoError(t, clickTester.Click("Beta"))
	clickTester.Frame()

	jumped := New(&fakeServices{tokenRows: rows})
	jumpTester := ui.NewTester(jumped.View, 1280, 1800)
	jumped.ShowPage("Token")
	jumpTester.Frame()
	jumped.ShowRecord("token", "tok-b")
	jumpTester.Frame()

	require.Equal(t, clicked.token.tokenID, jumped.token.tokenID)
	require.Equal(t, clicked.token.chosen, jumped.token.chosen,
		"the jumped row must be highlighted the way a click highlights it")
	require.Equal(t, clicked.token.resolvedOwner, jumped.token.resolvedOwner,
		"and the write forms must know the same owner")
	require.NotEmpty(t, jumped.token.resolvedOwner, "the fixture must have an owner to compare")
}

// TestJumpToARecordOutsideTheLoadedPage is the fallback half: a pasted id that is
// not on the page still becomes the subject, so the cards act on it and the
// page's own read fills in what it can. Refusing to move would be worse — the
// operator would have to find the record by hand to learn it is gone.
func TestJumpToARecordOutsideTheLoadedPage(t *testing.T) {
	app := New(&fakeServices{tokenRows: []TokenRow{
		{ID: "tok-a", Name: "Alpha", Symbol: "A", TotalSupply: "1", Owner: "owner-a-pk"},
	}})
	tester := ui.NewTester(app.View, 1280, 1800)
	app.ShowPage("Token")
	tester.Frame()

	app.ShowRecord("token", "tok-pasted")
	tester.Frame()

	require.Equal(t, "tok-pasted", app.token.tokenID, "the pasted id is still the subject")
	require.Equal(t, -1, app.token.chosen, "and no row is highlighted, because none is that record")
}

// TestSessionJumpLoadsTheRoster extends the click-equivalence rule to the third
// collection. A session row's selection does more than copy an id: it loads the
// roster the ballot form needs and invalidates a tally belonging to the previous
// session. A jump that skipped those left the page showing one session's tally
// beside another session's id.
func TestSessionJumpLoadsTheRoster(t *testing.T) {
	session := SessionRow{
		ID: "sess-1", Title: "Budget 2026", Status: "active",
		StartTime: 1_800_000_000, EndTime: 1_800_086_400,
	}

	app := New(&fakeServices{sessions: []SessionRow{session}})
	tester := ui.NewTester(app.View, 1280, 2000)
	app.ShowPage("Voting")
	tester.Frame()

	app.ShowRecord("session", "sess-1")
	tester.Frame()

	require.Equal(t, "sess-1", app.voting.controlID)
	require.Equal(t, 0, app.voting.sessionChoice, "the jumped row is highlighted")
	require.Equal(t, "sess-1", app.voting.rosterSessionID,
		"the roster the ballot form needs is loaded for the jumped session")
	require.Equal(t, 1, app.voting.tab, "and the page opens on the stage that reads a session")
}

// TestOracleJumpSelectsTheSource extends the click-equivalence rule to the source
// list. Choosing a source also highlights its row and re-reads the cards already
// showing it; a jump that left the row unselected showed a page where the actions
// had a target but nothing on screen said which.
func TestOracleJumpSelectsTheSource(t *testing.T) {
	source := SourceRow{ID: "src-1", Name: "btc", URL: "https://example.com", Type: "http", Enabled: true}
	app := New(&fakeServices{sources: []SourceRow{source}})
	tester := ui.NewTester(app.View, 1280, 1800)
	app.ShowPage("Oracle")
	tester.Frame()

	app.ShowRecord("oracle", "src-1")
	tester.Frame()

	require.Equal(t, 0, app.oracle.selected, "the jumped source is the highlighted row")
	require.Equal(t, "src-1", app.oracle.fetchID)
	require.Equal(t, "src-1", app.oracle.latestID)
}
