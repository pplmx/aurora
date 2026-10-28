package nativeui

import (
	"errors"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestUseGeneratedCreatorKey pins the same shortcut for an asset minted with a
// generated key: Transfer/Burn can fill their private-key field while the
// creator is still the holder and this asset is the subject.
func TestUseGeneratedCreatorKey(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.creator, app.nft.creatorPriv = "creator-pub", "creator-priv"
	app.nft.creatorKeyNFTID = "n1"
	app.nft.transferID, app.nft.from = "n1", "creator-pub"
	tester.Frame()
	require.True(t, tester.HasText("Use generated creator key"))

	require.NoError(t, tester.Click("Use generated creator key"))
	tester.Frame()
	require.Equal(t, "creator-priv", app.nft.priv)

	app.nft.from = "another-owner"
	tester.Frame()
	require.False(t, tester.HasText("Use generated creator key"),
		"the creator key must not be offered after the asset changes hands")
}

// TestMintingAnNftSelectsItsRow pins the post-mint context: a successful mint
// used to leave a filter or search hiding the new asset. The collection now
// clears both, highlights the row and makes the new id the subject.
func TestMintingAnNftSelectsItsRow(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{{
		ID: "n1", Name: "New", Owner: "creator-key", Creator: "creator-key",
	}}}
	app := New(fake)
	app.ShowPage("NFT")

	app.nft.filterOwner, app.nft.owner = "other-owner", "other-owner"
	app.nft.search, app.nft.appliedSearch = "old-query", "old-query"
	app.nft.from, app.nft.burnOwner = "old-owner", "old-owner"
	app.nft.to = "old-recipient"
	app.nft.priv, app.nft.burnPriv = "old-transfer-key", "old-burn-key"
	app.nft.name, app.nft.creator = "New", "creator-key"
	app.mintNFT()

	require.Empty(t, app.nft.filterOwner, "a mint must show the new asset")
	require.Empty(t, app.nft.search, "a mint must clear a search that hides it")
	require.Equal(t, "n1", app.nft.id)
	require.Equal(t, 0, app.nft.selected, "the new asset must be the highlighted row")
	require.Equal(t, "creator-key", app.nft.from, "the creator is the new asset's current holder")
	require.Equal(t, "creator-key", app.nft.burnOwner, "Burn must start from the same holder")
	require.Empty(t, app.nft.to, "the previous asset's recipient must not survive")
	require.Empty(t, app.nft.priv, "the previous asset's transfer key must not survive")
	require.Empty(t, app.nft.burnPriv, "the previous asset's burn key must not survive")
}

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

// TestNFTReadoutFailureKeepsSameSubjectRows applies the stale-load policy to
// the Inspect and History cards: a failed refresh of the same asset keeps the
// last successful rows, marks them stale, and never claims the history is
// empty.
func TestNFTReadoutFailureKeepsSameSubjectRows(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.id = "n1"
	app.nft.nft = &NFTRow{ID: "n1", Name: "Star"}
	app.nft.nftLoadedID = "n1"
	app.nft.historyID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint"}}
	app.nft.historyLoadedID = "n1"
	app.nft.opTab = 2
	tester.Frame()

	boom := errors.New("database is locked")
	fake := app.services.(*fakeServices)
	fake.loadErrs = fakeLoadErrors{nftDetails: boom, nftHistory: boom}
	app.refresh()
	tester.Frame()

	require.Equal(t, "Star", app.nft.nft.Name)
	require.Len(t, app.nft.history, 1)
	require.ErrorIs(t, app.nft.nftErr, boom)
	require.ErrorIs(t, app.nft.historyErr, boom)
	require.True(t, tester.HasText("Couldn't load NFT"))
	require.True(t, tester.HasText("Couldn't load NFT history"))
	require.True(t, tester.HasText("Showing the last successful load."))
	require.False(t, tester.HasText("No history loaded yet."))
}

// TestNFTSelectionDropsReadoutsWhenTheNewReadFails pins the key side: choosing
// another asset invalidates the previous details and history before reading,
// so a failure cannot leave them under the new id.
func TestNFTSelectionDropsReadoutsWhenTheNewReadFails(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "FirstArt", Owner: "owner-a"},
		{ID: "n2", Name: "SecondArt", Owner: "owner-b"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.nft = &NFTRow{ID: "n1", Name: "OldArt"}
	app.nft.nftLoadedID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint-from-old"}}
	app.nft.historyLoadedID = "n1"
	app.nft.opTab = 2
	tester.Frame()
	require.True(t, tester.HasText("OldArt"))

	boom := errors.New("database is locked")
	fake.loadErrs = fakeLoadErrors{nftDetails: boom, nftHistory: boom}
	require.NoError(t, tester.Click("n2"))
	tester.Frame()

	require.Equal(t, "n2", app.nft.id)
	require.Nil(t, app.nft.nft, "the old asset's details must be gone")
	require.Empty(t, app.nft.history, "the old asset's history must be gone")
	require.True(t, tester.HasText("Couldn't load NFT"))
	require.True(t, tester.HasText("Couldn't load NFT history"))
	require.False(t, tester.HasText("OldArt"))
	require.False(t, tester.HasText("No history loaded yet."))
}

// TestNFTInspectShowsLinksAndBlock pins the fields the Mint form collects but
// Inspect used to hide. The URLs are long, so the copy row shows both ends and
// the button hands over the stored value.
func TestNFTInspectShowsLinksAndBlock(t *testing.T) {
	const (
		imageURL = "https://images.example.com/star.png"
		tokenURI = "ipfs://bafy-star-metadata"
	)
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.id = "n1"
	app.nft.nft = &NFTRow{
		ID: "n1", Name: "Star", ImageURL: imageURL, TokenURI: tokenURI, BlockHeight: 42,
	}
	app.nft.nftLoadedID = "n1"
	tester.Frame()

	require.True(t, tester.HasText("Image URL"))
	require.True(t, tester.HasText("Token URI"))
	require.True(t, tester.HasText("42"), "the block that minted the asset must be shown")

	require.NoError(t, tester.Click("Copy Image URL"))
	tester.Frame()
	require.Equal(t, imageURL, tester.Clipboard())
	require.NoError(t, tester.Click("Copy Token URI"))
	tester.Frame()
	require.Equal(t, tokenURI, tester.Clipboard())
}

// TestNFTCollectionPages covers the collection's real limit/offset page:
// only the newest hundred are loaded, and Next reaches the remainder.
func TestNFTCollectionPages(t *testing.T) {
	nfts := make([]NFTRow, 150)
	for i := range nfts {
		nfts[i] = NFTRow{ID: "n" + strconv.Itoa(i), Name: "Asset " + strconv.Itoa(i), Owner: "owner"}
	}
	app := New(&fakeServices{nftRows: nfts})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("NFT")
	tester.Frame()

	require.Len(t, app.nft.list, 100)
	require.Equal(t, 0, app.nft.offset)
	require.True(t, tester.HasText("Entries 1–100"))
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, 100, app.nft.offset)
	require.Len(t, app.nft.list, 50)
	require.True(t, tester.HasText("Entries 101–150"))
}
