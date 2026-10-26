package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTokenSearchNarrowsTheList pins the collection's search field: every
// stored token is listed, and the field narrows by id, name or symbol without
// loading again. Choosing a row after filtering must still be that row.
func TestTokenSearchNarrowsTheList(t *testing.T) {
	fake := &fakeServices{tokenRows: []TokenRow{
		{ID: "t1", Name: "Gold", Symbol: "AU", TotalSupply: "10"},
		{ID: "t2", Name: "Silver", Symbol: "AG", TotalSupply: "20"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("Gold"))
	require.True(t, tester.HasText("Silver"))

	app.token.search = "silver"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a name match must stay listed")
	require.False(t, tester.HasText("Gold"), "a non-match must leave the list")

	app.token.search = "AG"
	tester.Frame()
	require.True(t, tester.HasText("Silver"), "a symbol match must stay listed")

	app.token.search = "nothing-like-this"
	tester.Frame()
	require.True(t, tester.HasText("No tokens match “nothing-like-this”."))

	// Choosing a row from the filtered list selects that token.
	app.token.search = "Gold"
	tester.Frame()
	require.NoError(t, tester.Click("Gold"))
	tester.Frame()
	require.Equal(t, "t1", app.token.tokenID)

	app.token.search = ""
	tester.Frame()
	require.True(t, tester.HasText("Gold"))
	require.True(t, tester.HasText("Silver"))
}

// TestNFTSearchNarrowsTheCollection covers the same field on the collection,
// which already has an owner filter: search applies on top of it.
func TestNFTSearchNarrowsTheCollection(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First Art", Owner: "owner-a"},
		{ID: "n2", Name: "Second Art", Owner: "owner-a"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()
	require.True(t, tester.HasText("First Art"))
	require.True(t, tester.HasText("Second Art"))

	app.nft.search = "second"
	tester.Frame()
	require.True(t, tester.HasText("Second Art"))
	require.False(t, tester.HasText("First Art"))

	app.nft.search = "none"
	tester.Frame()
	require.True(t, tester.HasText("No NFTs match “none”."))
}
