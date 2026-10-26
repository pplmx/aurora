package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestCurrentIdentifiersAreCopyable pins one rule across the pages: an id the
// window shows is also an id the operator can take out of it. Native text is
// not selectable, and the tables shorten UUIDs, so without these buttons the
// only full id was a field that could be read but not copied.
func TestCurrentIdentifiersAreCopyable(t *testing.T) {
	cases := []struct {
		name   string
		page   string
		fake   *fakeServices
		setup  func(*App)
		button string
		want   string
	}{
		{
			name: "token", page: "Token", fake: &fakeServices{},
			setup:  func(a *App) { a.token.tokenID = "t1" },
			button: "Copy token ID", want: "t1",
		},
		{
			name: "nft", page: "NFT", fake: &fakeServices{},
			setup:  func(a *App) { a.nft.id = "n1" },
			button: "Copy NFT ID", want: "n1",
		},
		{
			name: "lottery", page: "Lottery", fake: &fakeServices{},
			setup:  func(a *App) { a.lottery.verifyID = "d1" },
			button: "Copy Draw ID", want: "d1",
		},
		{
			name: "oracle", page: "Oracle",
			fake:   &fakeServices{sources: []SourceRow{{ID: "s1", Name: "btc"}}},
			setup:  func(a *App) { a.oracle.fetchID = "s1" },
			button: "Copy Source ID", want: "s1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := New(tc.fake)
			tester := ui.NewTester(app.View, 1280, 5000)
			app.ShowPage(tc.page)
			tc.setup(app)
			tester.Frame()

			require.NoError(t, tester.Click(tc.button))
			tester.Frame()
			require.Equal(t, tc.want, tester.Clipboard())
		})
	}
}

// TestSelectedVotingIDsAreCopyable covers the two rows whose full id is the
// input to every other form: the selected session and candidate.
func TestSelectedVotingIDsAreCopyable(t *testing.T) {
	const sessionID = "11111111-2222-3333-4444-555555555555"
	const candidateID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	fake := &fakeServices{
		sessions:   []SessionRow{{ID: sessionID, Title: "Budget", Status: "active"}},
		candidates: []CandidateRow{{ID: candidateID, Name: "Alice"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	app.voting.sessionChoice, app.voting.candChoice = 0, 0
	tester.Frame()

	require.NoError(t, tester.Click("Copy Selected session ID"))
	tester.Frame()
	require.Equal(t, sessionID, tester.Clipboard())

	require.NoError(t, tester.Click("Copy Selected candidate ID"))
	tester.Frame()
	require.Equal(t, candidateID, tester.Clipboard())
}
