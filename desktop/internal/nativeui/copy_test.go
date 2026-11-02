package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
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

// TestChineseCopyLabelsHaveNoSpuriousSpace covers the labels built out of two
// translated words. The desktop joined them with a literal space, so Chinese
// showed 复制 诊断信息 next to catalog entries such as 复制载荷 that carry none.
//
// The Latin case is checked too: a Chinese window still spaces a Latin word,
// which is the rule the project's Chinese prose follows everywhere else.
func TestChineseCopyLabelsHaveNoSpuriousSpace(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	for _, tc := range []struct {
		locale string
		want   []string
		avoid  []string
	}{
		{"zh", []string{"复制诊断信息"}, []string{"复制 诊断信息"}},
		{"zh-Hant", []string{"複製診斷信息"}, []string{"複製 診斷信息"}},
	} {
		translator.SetLocale(tc.locale)
		app := New(&fakeServices{})
		app.SetActions(Actions{
			Version: "v1.2.3", BuildTime: "2026-10-26T09:00:00Z", DBPath: "/tmp/aurora/aurora.db",
			ShowDataFolder: func() {}, BackUpDatabase: func() {}, ShowAbout: func() {},
		})
		tester := ui.NewTester(app.View, 1280, 5000)
		app.ShowPage("Settings")
		tester.Frame()

		texts := tester.Texts()
		for _, want := range tc.want {
			require.Containsf(t, texts, want, "%s: the Copy action must read %q", tc.locale, want)
		}
		for _, avoid := range tc.avoid {
			require.NotContainsf(t, texts, avoid,
				"%s: two Chinese words must not be separated by a space", tc.locale)
		}
	}

	// A translated word beside a Latin one keeps its space: the value is the
	// token field, not a second Chinese word.
	translator.SetLocale("zh")
	require.Equal(t, "复制 token-1", copyActionLabel("token-1"))
	// English keeps the plain space between its two words.
	translator.SetLocale("en")
	require.Equal(t, "Copy Diagnostics", copyActionLabel("Diagnostics"))
}

// TestCopiedValuesAreTheWholeValueNotTheShortenedCell covers the half the id test
// above cannot see. That one copies short identifiers, and a copy of a short value
// is also a copy of its shortened cell, so it cannot tell the two apart. A long
// value is where copying the cell would differ from copying the value, and the
// reference promises the whole value: the abbreviation exists to keep the row narrow,
// not to be what the operator pastes.
func TestCopiedValuesAreTheWholeValueNotTheShortenedCell(t *testing.T) {
	const longID = "9f2c1d4e-6a7b-8c9d-0e1f-2a3b4c5d6e7f-extra-tail"
	require.NotEqual(t, abbreviate(longID), longID,
		"the fixture must be long enough to be shortened, or this proves nothing")

	fake := &fakeServices{sources: []SourceRow{{ID: longID, Name: "btc"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	app.oracle.fetchID = longID
	tester.Frame()

	require.NoError(t, tester.Click(copyActionLabel("Source ID")))
	tester.Frame()
	require.Equal(t, longID, tester.Clipboard(),
		"the button copies the value, not the shortened cell")
}
