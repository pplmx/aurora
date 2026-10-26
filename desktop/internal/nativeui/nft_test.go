package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestNFTFilterIsTheListState pins what the Collection card is showing. The
// owner field used to be read only at the moment the Filter button was
// clicked, so a Refresh (or re-entering the page) replaced the filtered rows
// with every NFT while the field still named an owner; a failed filtered read
// blanked the collection, and a filter with no matches claimed no NFT existed.
func TestNFTFilterIsTheListState(t *testing.T) {
	fake := &fakeServices{
		nftRows:   []NFTRow{{ID: "n1", Name: "First Art", Owner: "owner-a"}},
		ownerNFTs: []NFTRow{{ID: "n2", Name: "Second Art", Owner: "owner-b"}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 3000)
	app.ShowPage("NFT")
	tester.Frame()
	require.True(t, tester.HasText("First Art"), "the whole collection loads with the page")

	app.nft.owner = "owner-b"
	app.filterNFTs()
	tester.Frame()
	require.Equal(t, "owner-b", app.nft.filterOwner)
	require.True(t, tester.HasText("Second Art"))
	require.False(t, tester.HasText("First Art"), "the filter must narrow the list")

	// Refresh and re-entering the page re-apply the filter the field names.
	app.refresh()
	tester.Frame()
	require.Equal(t, "owner-b", app.nft.filterOwner, "a refresh must keep the applied filter")
	require.Len(t, app.nft.list, 1)
	require.Equal(t, "Second Art", app.nft.list[0].Name)

	app.ShowPage("Dashboard")
	app.ShowPage("NFT")
	tester.Frame()
	require.Equal(t, "owner-b", app.nft.filterOwner, "a page switch must keep the applied filter")
	require.Equal(t, "Second Art", app.nft.list[0].Name)

	// A failed filtered read reports the error and keeps the rows already
	// shown, instead of emptying the collection.
	fake.loadErrs.nfts = errors.New("database is locked")
	app.filterNFTs()
	tester.Frame()
	require.Len(t, app.nft.list, 1, "a failed refresh must keep the previous rows")
	require.Equal(t, "Second Art", app.nft.list[0].Name)
	require.True(t, tester.HasText("database is locked"))
	require.True(t, tester.HasText("Second Art"))

	// Show All clears both the input and the applied filter.
	fake.loadErrs.nfts = nil
	app.showAllNFTs()
	tester.Frame()
	require.Empty(t, app.nft.owner)
	require.Empty(t, app.nft.filterOwner)
	require.True(t, tester.HasText("First Art"))

	// An owner with no assets says so, without claiming the store is empty.
	fake.ownerNFTs = nil
	app.nft.owner = "owner-with-nothing"
	app.filterNFTs()
	tester.Frame()
	require.True(t, tester.HasText("No NFTs for this owner. Show All restores the collection."))
	require.False(t, tester.HasText("No NFTs yet. Mint one above."))
}
