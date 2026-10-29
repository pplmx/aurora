package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// collectionList is the one renderer behind every collection in the window.
// These tests pin the behaviour that used to be reimplemented per page: a
// record says what it holds, a choice becomes the page's subject, and the
// change is spoken because a screen reader cannot see the forms refill.

// sampleRecords is the fixture the collection tests share.
func sampleRecords() []collectionRecord {
	return []collectionRecord{
		{
			title: "Gold",
			facts: []collectionFact{
				{label: "Symbol", value: "GLD"},
				{label: "Supply", value: "1000"},
			},
			fields: []collectionField{{label: "ID", value: "token-1"}},
		},
		{
			title: "Silver",
			facts: []collectionFact{
				{label: "Symbol", value: "SLV"},
				// An empty fact must not take a slot: the record would show a
				// label with nothing beside it.
				{label: "Supply", value: ""},
			},
			fields: []collectionField{{label: "ID", value: "token-2"}},
		},
	}
}

// TestCollectionListDrawsEachRecord pins what a record renders: its title, the
// labelled facts beside it, and the labelled fields under it.
func TestCollectionListDrawsEachRecord(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Tokens", sampleRecords(), -1, nil)
	}, 520, 400)
	tester.Frame()

	for _, want := range []string{"Tokens", "Gold", "Silver", "Symbol", "GLD", "Supply", "1000", "ID", "token-1"} {
		require.Truef(t, tester.HasText(want), "the record must render %q", want)
	}
	require.True(t, tester.HasText("SLV"), "the second record keeps its own facts")
}

// TestCollectionListSkipsEmptyFacts keeps a blank value out of the record. A
// fact with nothing in it drew a label and an em dash, which read as a fact the
// store had lost rather than as one that does not apply.
func TestCollectionListSkipsEmptyFacts(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Tokens", sampleRecords(), -1, nil)
	}, 520, 400)
	tester.Frame()

	// Both records say "Symbol"; only the one with a value says "Supply".
	require.Equal(t, 2, countText(tester.Texts(), "Symbol"),
		"each record draws its own symbol label")
	require.Equal(t, 1, countText(tester.Texts(), "Supply"),
		"a record with no supply must not draw the label at all")
}

// TestCollectionListAnnouncesTheChoice pins the selection feedback. Choosing a
// record refills the page's forms and read cards while focus stays on the
// record's button, so nothing about that change reaches a screen reader unless
// it is announced.
func TestCollectionListAnnouncesTheChoice(t *testing.T) {
	chosen := -1
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Tokens", sampleRecords(), chosen, func(i int) { chosen = i })
	}, 520, 400)
	tester.Frame()
	tester.Announcements()

	require.NoError(t, tester.Click("Silver"))
	tester.Frame()
	require.Equal(t, 1, chosen, "clicking a record must report its index")
	require.Equal(t, []string{tr("Selected Silver")}, tester.Announcements(),
		"the newly chosen record must be spoken once")
}

// TestCollectionListMarksTheChosenRecord pins the visible and spoken state of a
// chosen record: the button says it is selected, so the choice is not carried
// by the tint alone.
func TestCollectionListMarksTheChosenRecord(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Tokens", sampleRecords(), 1, func(int) {})
	}, 520, 400)
	tester.Frame()

	require.True(t, tester.HasText(tr("Selected Silver")),
		"the chosen record must announce its selected state")
	require.True(t, tester.HasText(tr("Select Gold")),
		"an unchosen record must offer itself")
}

// TestCollectionListWithoutSelectionDrawsPlainRecords keeps the read-only form
// working: a history or a reading list has no subject to choose, so its rows
// are records rather than buttons.
func TestCollectionListWithoutSelectionDrawsPlainRecords(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Token transfer history", sampleRecords(), -1, nil)
	}, 520, 400)
	tester.Frame()

	require.True(t, tester.HasText("Gold"), "a plain record still renders")
	require.False(t, tester.HasText(tr("Select Gold")),
		"a read-only collection must not pretend its rows are choices")
}

// TestCollectionRecordNamesItselfInTheWindowsLanguage pins the accessible name
// of a record: it is the record's title, so assistive technology announces the
// same word the record draws.
func TestCollectionRecordNamesItselfInTheWindowsLanguage(t *testing.T) {
	row := collectionRecord{title: "Gold"}
	require.Equal(t, "Gold", collectionRecordName(row))

	untitled := collectionRecord{}
	require.NotEmpty(t, collectionRecordName(untitled),
		"an untitled record must still have a name rather than an empty one")
}
