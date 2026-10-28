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
	app.nft.transferSubjectID = "n1"
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

	// A different filter is a different question: retaining owner-b's rows
	// under owner-c would falsely imply they answered the new read.
	app.nft.owner = "owner-c"
	app.filterNFTs()
	tester.Frame()
	require.Empty(t, app.nft.list, "a failed read for a new filter must not keep another filter's rows")
	require.False(t, tester.HasText("Second Art"))
	require.True(t, tester.HasText("database is locked"))

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

// TestManualNFTIDsMoveOnlyTheirHolderContext covers typed ids in the two write
// cards. They intentionally have separate ids, so changing Transfer must not
// retarget Burn, but neither may retain the previous asset's holder or secret.
func TestManualNFTIDsMoveOnlyTheirHolderContext(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First", Owner: "owner-1"},
		{ID: "n2", Name: "Second", Owner: "owner-2"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.transferID, app.nft.burnID = "n1", "n1"
	tester.Frame()
	require.Equal(t, "owner-1", app.nft.from)
	require.Equal(t, "owner-1", app.nft.burnOwner)

	app.nft.priv, app.nft.to, app.nft.burnPriv = "transfer-key", "old-recipient", "burn-key"
	app.nft.transferID = "n2"
	tester.Frame()
	require.Equal(t, "owner-2", app.nft.from)
	require.Empty(t, app.nft.priv)
	require.Empty(t, app.nft.to)
	require.Equal(t, "owner-1", app.nft.burnOwner, "Transfer's id must not retarget Burn")
	require.Equal(t, "burn-key", app.nft.burnPriv, "the other card's key must remain untouched")

	app.nft.burnID = "n2"
	tester.Frame()
	require.Equal(t, "owner-2", app.nft.burnOwner)
	require.Empty(t, app.nft.burnPriv)

	app.nft.transferID = "outside-page"
	tester.Frame()
	require.Empty(t, app.nft.from, "an unknown asset must not inherit another holder")
	require.Empty(t, app.nft.priv)
	require.Empty(t, app.nft.to)
	require.Equal(t, "owner-2", app.nft.burnOwner)
}

// TestNFTTransferMovesOnlyItsOwnWriteCard keeps a successful Transfer from
// clearing or retargeting a Burn form aimed at a different asset. When both
// cards do name the transferred asset, both must follow its new holder and
// drop the key that belonged to the previous owner.
func TestNFTTransferMovesOnlyItsOwnWriteCard(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{
		{ID: "n1", Name: "First", Owner: "owner-1"},
		{ID: "n2", Name: "Second", Owner: "owner-2"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.transferID, app.nft.burnID = "n1", "n2"
	tester.Frame()
	app.nft.to, app.nft.priv, app.nft.burnPriv = "owner-new", "transfer-key", "burn-key"
	app.transferNFT()
	tester.Frame()

	require.Equal(t, "owner-new", app.nft.from)
	require.Empty(t, app.nft.priv)
	require.Empty(t, app.nft.to)
	require.Equal(t, "owner-2", app.nft.burnOwner, "a Transfer must not retarget Burn")
	require.Equal(t, "burn-key", app.nft.burnPriv, "a Transfer must not clear Burn's key")

	app.nft.burnID = "n1"
	tester.Frame()
	require.Equal(t, "owner-1", app.nft.burnOwner, "the list still holds pre-transfer data")
	app.nft.to, app.nft.priv, app.nft.burnPriv = "owner-newer", "transfer-key-2", "same-asset-burn-key"
	app.transferNFT()
	tester.Frame()

	require.Equal(t, "owner-newer", app.nft.from)
	require.Equal(t, "owner-newer", app.nft.burnOwner, "Burn must follow a transfer of its own asset")
	require.Empty(t, app.nft.burnPriv, "Burn's key belonged to the previous owner")
}

// TestNFTInspectResolvesManualWriteContext lets an id outside the visible
// collection acquire the holder its Transfer/Burn cards need. A successful
// Inspect is the authority when neither the row nor a previous read knew the
// asset.
func TestNFTInspectResolvesManualWriteContext(t *testing.T) {
	fake := &fakeServices{nftByID: map[string]NFTRow{
		"archived": {ID: "archived", Name: "Archive", Owner: "archived-owner"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.transferID, app.nft.burnID = "archived", "archived"
	tester.Frame()
	require.Empty(t, app.nft.from)

	app.nft.id = "archived"
	require.NoError(t, tester.Click("Get NFT"))
	tester.Frame()
	require.Equal(t, "archived-owner", app.nft.from)
	require.Equal(t, "archived-owner", app.nft.burnOwner)
	require.Equal(t, "archived", app.nft.transferOwnerID)
	require.Equal(t, "archived", app.nft.burnOwnerID)
}

// TestNFTReadoutsTrackTheirCurrentInputs keeps Inspect metadata and History
// rows from describing an asset after the visible id has moved to another one.
func TestNFTReadoutsTrackTheirCurrentInputs(t *testing.T) {
	fake := &fakeServices{nftRows: []NFTRow{{ID: "n1"}, {ID: "n2"}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("NFT")
	tester.Frame()

	app.nft.id = "n1"
	app.nft.nft = &NFTRow{ID: "n1", Name: "Old asset", Owner: "old-owner"}
	app.nft.nftLoadedID = "n1"
	app.nft.historyID = "n1"
	app.nft.history = []NFTOpRow{{Type: "mint", From: "old-from"}}
	app.nft.historyLoadedID = "n1"
	app.nft.opTab = 2
	tester.Frame()
	require.True(t, tester.HasText("Old asset"))
	require.True(t, tester.HasText("old-from"))

	app.nft.id = "n2"
	tester.Frame()
	require.False(t, tester.HasText("Old asset"), "another asset must hide the old metadata")

	app.nft.historyID = "n2"
	tester.Frame()
	require.False(t, tester.HasText("old-from"), "another asset must hide the old history")
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

	app.nft.id = "n1"
	app.nft.nft = &NFTRow{ID: "n1", Name: "OldArt"}
	app.nft.nftLoadedID = "n1"
	app.nft.historyID = "n1"
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
