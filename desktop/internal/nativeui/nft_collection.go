package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// selectNFTRow highlights a record in the collection after a mint.
func (a *App) selectNFTRow(id string) {
	a.nft.selected = -1
	for i, n := range a.nft.list {
		if n.ID == id {
			a.nft.selected = i
			return
		}
	}
}

func (a *App) nftCollection(c *ui.Context) {
	formCard(c, "Collection", "Every stored asset, newest first. Choose a row to load its id into Transfer, Burn and History.", func() {
		a.nftListBody(c)
	})
}

// nftInspect is the id lookup and owner filter. It sits in the workbench
// column so the id it resolves is beside the transfer and burn forms that
// consume it.
func (a *App) nftInspect(c *ui.Context) {
	formCard(c, "Inspect", "Look up one NFT by id, or narrow the collection to one owner.", func() {
		a.nftReadBody(c)
	})
}

func (a *App) nftReadBody(c *ui.Context) {
	field(c, "NFT ID", &a.nft.id)
	id := strings.TrimSpace(a.nft.id)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get NFT", allSet(a.nft.id)).Clicked() {
			a.readNFTDetails()
		}
	})
	if a.nft.nftErr != nil && a.nft.nftLoadedID == id {
		loadWarning(c, "Couldn't load NFT", a.nft.nftErr, a.nft.nft != nil)
	}
	copyRow(c, "NFT ID", strings.TrimSpace(a.nft.id), nil)
	if n := a.nft.nft; n != nil && a.nft.nftLoadedID == id {
		rows := []readoutRow{
			{"Name", n.Name}, {"Owner", n.Owner}, {"Creator", n.Creator},
			{"Block", strconv.FormatInt(n.BlockHeight, 10)},
		}
		if n.Description != "" {
			rows = append(rows, readoutRow{"About", n.Description})
		}
		readout(c, rows)
		// URLs are long enough that the readout would clip them; the copy row
		// shows both ends and makes the stored value retrievable.
		copyRow(c, "Image URL", n.ImageURL, nil)
		copyRow(c, "Token URI", n.TokenURI, nil)
	}

	// The Collection card holds the whole collection; this narrows it to one
	// owner, and Show All restores everything.
	fieldHint(c, "Owner (base64 public key)", "Narrow the Collection card to one owner.", &a.nft.owner)
	ui.Row(c).Gap(8).Children(func() {
		if secondaryButton(c, "Filter by Owner", allSet(a.nft.owner)).Clicked() {
			a.filterNFTs()
		}
		if secondaryButton(c, "Show All").Clicked() {
			a.showAllNFTs()
		}
	})
}

func (a *App) readNFTDetails() {
	id := strings.TrimSpace(a.nft.id)
	if rejectIncomplete(&a.nft.status, "NFT ID is required for this read.", id) {
		return
	}
	a.clearNFTDetailsForAnotherSubject(id)
	if err := a.loadNFTDetails(id); err != nil {
		a.nft.status = errStatus(err)
	} else {
		a.nft.status = statusLine{}
	}
}

// filterNFTs applies the owner named in the field. It goes through loadNFT so
// the filter survives refreshes and a failed read keeps the rows already shown
// instead of blanking the collection.
func (a *App) filterNFTs() {
	next := strings.TrimSpace(a.nft.owner)
	if rejectIncomplete(&a.nft.status, "Owner is required to filter the collection.", next) {
		return
	}
	if next != a.nft.filterOwner {
		a.nft.list = nil
		a.nft.loadErr = nil
		a.nft.offset = 0
		a.nft.hasMore = false
	}
	a.nft.filterOwner = next
	a.nft.selected = -1
	a.loadNFTPage(0)
}

// showAllNFTs clears the filter and reloads the whole collection.
func (a *App) showAllNFTs() {
	a.nft.owner = ""
	if a.nft.filterOwner != "" {
		a.nft.list = nil
		a.nft.loadErr = nil
		a.nft.offset = 0
		a.nft.hasMore = false
	}
	a.nft.filterOwner = ""
	a.nft.selected = -1
	a.loadNFTPage(0)
}

// nftListBody draws the collection table.
func (a *App) nftListBody(c *ui.Context) {
	if a.nft.loadErr != nil && !loadWarning(c, "Failed to load NFTs", a.nft.loadErr, len(a.nft.list) > 0) {
		return
	}
	if len(a.nft.list) == 0 {
		if a.nft.filterOwner != "" {
			emptyNote(c, "No NFTs for this owner. Show All restores the collection.")
			return
		}
		emptyState(c, pageNFT, "No NFTs yet", "Mint an asset below to start the collection; it becomes the first holder’s token.")
		return
	}
	queryChanged := a.nft.search != a.nft.appliedSearch
	if queryChanged {
		a.nft.appliedSearch = a.nft.search
		a.nft.selected = -1
	}
	limit := int(a.nft.limit)
	if limit < 10 {
		limit = nftPageSize
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	previous, next := collectionToolbar(c, &a.nft.search, "Search NFTs",
		"Search narrows the current page; use Previous/Next for older assets.",
		"NFT collection", a.nft.offset, len(a.nft.list), a.nft.hasMore)
	if previous || next {
		target := a.nft.offset
		if previous {
			target = max(0, target-limit)
		} else {
			target += limit
		}
		if a.loadNFTPage(target) {
			a.nft.selected = -1
		}
		return
	}
	list := make([]NFTRow, 0, len(a.nft.list))
	for _, n := range a.nft.list {
		if containsFold(a.nft.search, n.ID, n.Name) {
			list = append(list, n)
		}
	}
	announceSearchResults(c, queryChanged, len(list))
	if len(list) == 0 {
		emptyNote(c, fmt.Sprintf(tr("No NFTs match “%s”."), strings.TrimSpace(a.nft.search)))
		return
	}
	collectionList(c, "NFT collection",
		collectionRecords(list, nftCollectionRecord),
		a.nft.selected, func(i int) { a.chooseNFT(list, i) })
}

// nftCollectionRecord projects a stored asset onto a record. The block is a
// fact beside the name: it is the proof of when the asset was recorded, and
// the id stays a field because it is what the write cards copy.
func nftCollectionRecord(n NFTRow) collectionRecord {
	name := strings.TrimSpace(n.Name)
	if name == "" {
		name = shortID(n.ID)
	}
	return collectionRecord{
		title: name,
		facts: []collectionFact{
			{label: "Block", value: strconv.FormatInt(n.BlockHeight, 10)},
		},
		fields: []collectionField{{label: "ID", value: n.ID}},
	}
}

// chooseNFT is the one path that makes a collection row the subject of every
// asset card. Move and Burn also take the holder from the same row, so the
// operator never retypes a key the collection already knows.
func (a *App) chooseNFT(list []NFTRow, index int) {
	if index < 0 || index >= len(list) {
		return
	}
	a.nft.selected = index
	nft := list[index]
	a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = nft.ID, nft.ID, nft.ID, nft.ID
	a.applyNFTSubject(nft.Owner)
	a.selectNFT(nft.ID)
}

// loadNFTDetails reads one asset into the Inspect card. A failure keeps the
// rows only when they were loaded for the same id; the caller invalidates them
// when the subject changes.
func (a *App) loadNFTDetails(id string) error {
	details, err := a.services.GetNFT(id)
	if err != nil {
		a.nft.nftErr = err
		a.nft.nftLoadedID = id
		return err
	}
	a.nft.nft = &details
	a.nft.nftLoadedID = id
	a.nft.nftErr = nil
	if strings.TrimSpace(a.nft.transferID) == id && a.nft.transferOwnerID != id {
		a.applyNFTTransferSubject(details.Owner)
	}
	if strings.TrimSpace(a.nft.burnID) == id && a.nft.burnOwnerID != id {
		a.applyNFTBurnSubject(details.Owner)
	}
	return nil
}

func (a *App) clearNFTDetailsForAnotherSubject(id string) {
	if a.nft.nftLoadedID == id {
		return
	}
	a.nft.nft = nil
	a.nft.nftErr = nil
	a.nft.nftLoadedID = ""
}

// selectNFT re-reads the Inspect card and history for the asset just chosen:
// both would otherwise keep describing the previously selected NFT. The
// readouts are invalidated before the read, so a failure cannot leave the old
// asset's rows under the new id.
func (a *App) selectNFT(id string) {
	if a.nft.nft != nil || a.nft.nftLoadedID != "" || a.nft.nftErr != nil {
		a.clearNFTDetailsForAnotherSubject(id)
		_ = a.loadNFTDetails(id)
	}
	if len(a.nft.history) > 0 || a.nft.historyLoadedID != "" || a.nft.historyErr != nil {
		a.clearNFTHistoryForAnotherSubject(id)
		_ = a.loadNFTHistory(id)
	}
}
