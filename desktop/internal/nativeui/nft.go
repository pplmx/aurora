package nativeui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// nftState is the NFT page's state.
type nftState struct {
	// Mint form.
	name        string
	description string
	imageURL    string
	tokenURI    string
	creator     string

	// Read forms.
	id    string
	owner string

	// filterOwner is the owner filter currently applied to the Collection
	// card, empty for the whole collection. It is separate from the owner
	// input so a refresh re-applies the filter the list is showing instead of
	// silently replacing a filtered list with every NFT while the field still
	// names an owner.
	filterOwner string
	// search narrows the shown collection by id or name, on top of the owner
	// filter.
	search string

	// Each write form names its own NFT: one shared field meant typing an id
	// in the Transfer card silently retargeted Inspect, Burn and History too.
	transferID string
	burnID     string
	historyID  string

	nft     *NFTRow
	list    []NFTRow
	history []NFTOpRow

	// Write forms.
	from      string
	to        string
	priv      string
	burnOwner string
	burnPriv  string

	loadErr error

	// askBurn gates the irreversible burn, as the CLI's --confirm does.
	askBurn bool

	status statusLine

	listTable    ui.ListState
	historyTable ui.ListState
	// selected is the collection row chosen, -1 for none.
	selected int
}

func (a *App) loadNFT() {
	if a.services == nil {
		return
	}
	// The collection is the page's entry point: an asset minted in another
	// session is otherwise reachable only by an id nobody wrote down. A filter
	// that is applied stays applied across refreshes and page switches.
	var (
		rows []NFTRow
		err  error
	)
	if a.nft.filterOwner != "" {
		rows, err = a.services.ListNFTs(a.nft.filterOwner)
	} else {
		rows, err = a.services.ListAllNFTs()
	}
	a.nft.loadErr = err
	if err == nil {
		a.nft.list = rows
	}
}

func (a *App) nftView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		// The collection leads: what exists comes before adding to it, and the
		// cards below act on the row it selects.
		twoColumns(c, func() {
			a.nftRead(c)
			a.nftMint(c)
		}, func() {
			a.nftWrite(c)
		})
		a.renderNFTBurnDialog(c)
	})
}

func (a *App) nftMint(c *ui.Context) {
	formCard(c, "Mint NFT", "Create a unique asset owned by the creator key.", func() {
		a.nftMintBody(c)
	})
}

func (a *App) nftMintBody(c *ui.Context) {
	field(c, "Name", &a.nft.name)
	field(c, "Description (optional)", &a.nft.description)
	field(c, "Image URL (optional)", &a.nft.imageURL)
	field(c, "Token URI (optional)", &a.nft.tokenURI)
	fieldHint(c, "Creator (base64 public key)", "The creator becomes the first owner.", &a.nft.creator)
	ui.Row(c).Children(func() {
		if primaryButton(c, "Mint NFT").Clicked() {
			a.mintNFT()
		}
	})
}

// mintNFT mints the asset described by the Mint card's inputs.
func (a *App) mintNFT() {
	n, err := a.services.MintNFT(
		strings.TrimSpace(a.nft.name), strings.TrimSpace(a.nft.description),
		strings.TrimSpace(a.nft.imageURL), strings.TrimSpace(a.nft.tokenURI),
		strings.TrimSpace(a.nft.creator))
	if err != nil {
		a.nft.status = errStatus(err)
		return
	}
	a.nft.status = okStatus("Minted %s (%s) at block %d", n.Name, n.ID, n.BlockHeight)
	a.loadNFT()
	// The new asset becomes the subject of every other card: the operator just
	// created it, so making them retype the id four times would be busywork.
	a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = n.ID, n.ID, n.ID, n.ID
	a.nft.name, a.nft.description, a.nft.imageURL, a.nft.tokenURI = "", "", "", ""
}

func (a *App) nftRead(c *ui.Context) {
	formCard(c, "Collection", "Every stored asset, newest first. Choose a row to load its id into the cards below.", func() {
		a.nftListBody(c)
	})
	formCard(c, "Inspect", "Look up one NFT by id, or narrow the collection to one owner.", func() {
		a.nftReadBody(c)
	})
}

func (a *App) nftReadBody(c *ui.Context) {
	field(c, "NFT ID", &a.nft.id)
	ui.Row(c).Children(func() {
		if secondaryButton(c, "Get NFT").Clicked() {
			n, err := a.services.GetNFT(strings.TrimSpace(a.nft.id))
			if err != nil {
				a.nft.status = errStatus(err)
				a.nft.nft = nil
			} else {
				a.nft.nft = &n
				a.nft.status = statusLine{}
			}
		}
	})
	if n := a.nft.nft; n != nil {
		rows := []readoutRow{{"Name", n.Name}, {"Owner", n.Owner}, {"Creator", n.Creator}}
		if n.Description != "" {
			rows = append(rows, readoutRow{"About", n.Description})
		}
		readout(c, rows)
	}

	// The Collection card holds the whole collection; this narrows it to one
	// owner, and Show All restores everything.
	fieldHint(c, "Owner (base64 public key)", "Narrow the Collection card to one owner.", &a.nft.owner)
	ui.Row(c).Gap(8).Children(func() {
		if secondaryButton(c, "Filter by Owner").Clicked() {
			a.filterNFTs()
		}
		if secondaryButton(c, "Show All").Clicked() {
			a.showAllNFTs()
		}
	})
}

// filterNFTs applies the owner named in the field. It goes through loadNFT so
// the filter survives refreshes and a failed read keeps the rows already shown
// instead of blanking the collection.
func (a *App) filterNFTs() {
	a.nft.filterOwner = strings.TrimSpace(a.nft.owner)
	a.loadNFT()
}

// showAllNFTs clears the filter and reloads the whole collection.
func (a *App) showAllNFTs() {
	a.nft.owner = ""
	a.nft.filterOwner = ""
	a.loadNFT()
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
		emptyNote(c, "No NFTs yet. Mint one above.")
		return
	}
	ui.SearchField(c, &a.nft.search).Label("Search NFTs")
	list := make([]NFTRow, 0, len(a.nft.list))
	for _, n := range a.nft.list {
		if containsFold(a.nft.search, n.ID, n.Name) {
			list = append(list, n)
		}
	}
	if len(list) == 0 {
		emptyNote(c, "No NFTs match “"+strings.TrimSpace(a.nft.search)+"”.")
		return
	}
	a.nft.listTable.Selected = &a.nft.selected
	table := ui.Table(c, &a.nft.listTable, []ui.TableColumn{
		{Title: "ID", Width: 150},
		{Title: "Name", MinWidth: 110},
		{Title: "Block", Width: 80, Align: ui.End},
	}, len(list), func(row, col int) {
		n := list[row]
		switch col {
		case 0:
			ui.Text(c, shortID(n.ID)).SingleLine()
		case 1:
			ui.Text(c, n.Name).SingleLine()
		case 2:
			ui.Text(c, strconv.FormatInt(n.BlockHeight, 10))
		}
	})
	if table.Changed() && a.nft.selected >= 0 && a.nft.selected < len(list) {
		// Choosing a row fills every card's id with that asset, which is how
		// the full id reaches a form after the table shortened it.
		n := list[a.nft.selected]
		a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = n.ID, n.ID, n.ID, n.ID
	}
}

func (a *App) nftWrite(c *ui.Context) {
	formCard(c, "Transfer", "Move an NFT to a new owner; the transfer is signed by the current one.", func() {
		field(c, "NFT ID", &a.nft.transferID)
		field(c, "From (base64 public key)", &a.nft.from)
		field(c, "To (base64 public key)", &a.nft.to)
		field(c, "From Private Key", &a.nft.priv)
		ui.Row(c).Children(func() {
			if primaryButton(c, "Transfer").Clicked() {
				a.transferNFT()
			}
		})
	})

	formCard(c, "Burn", "Destroy an NFT permanently. This cannot be undone.", func() {
		field(c, "NFT ID", &a.nft.burnID)
		field(c, "Owner (base64 public key)", &a.nft.burnOwner)
		field(c, "Owner Private Key", &a.nft.burnPriv)
		ui.Row(c).Children(func() {
			// A destructive action carries the danger style, and the
			// confirmation still stands between it and the write.
			if dangerButton(c, "Burn NFT…").Clicked() {
				a.nft.askBurn = true
			}
		})
	})

	formCard(c, "History", "Every mint, transfer and burn recorded for this NFT.", func() {
		field(c, "NFT ID", &a.nft.historyID)
		ui.Row(c).Children(func() {
			if secondaryButton(c, "Show History").Clicked() {
				ops, err := a.services.NFTHistory(strings.TrimSpace(a.nft.historyID))
				if err != nil {
					a.nft.status = errStatus(err)
					a.nft.history = nil
				} else {
					a.nft.history = ops
					a.nft.status = statusLine{}
				}
			}
		})
		a.nftHistoryTable(c)
	})
}

// transferNFT moves the asset named by the Transfer card.
func (a *App) transferNFT() {
	err := a.services.TransferNFT(
		strings.TrimSpace(a.nft.transferID), strings.TrimSpace(a.nft.from),
		strings.TrimSpace(a.nft.to), strings.TrimSpace(a.nft.priv))
	a.nft.status = wrapErr(err)
	if err == nil {
		a.refreshNFT(strings.TrimSpace(a.nft.transferID))
	}
}

// refreshNFT re-reads what a write just changed: the collection, plus the
// Inspect readout and history when they are already showing id. A successful
// transfer that left the old owner in the collection read as a failed write.
func (a *App) refreshNFT(id string) {
	a.loadNFT()
	if id == "" {
		return
	}
	if a.nft.nft != nil && a.nft.nft.ID == id {
		if updated, err := a.services.GetNFT(id); err == nil {
			a.nft.nft = &updated
		} else {
			a.nft.nft = nil
		}
	}
	if a.nft.historyID == id && len(a.nft.history) > 0 {
		if ops, err := a.services.NFTHistory(id); err == nil {
			a.nft.history = ops
		}
	}
}

func (a *App) nftHistoryTable(c *ui.Context) {
	if len(a.nft.history) == 0 {
		emptyNote(c, "No history loaded yet.")
		return
	}
	ops := a.nft.history
	ui.Table(c, &a.nft.historyTable, []ui.TableColumn{
		{Title: "Type", Width: 100},
		{Title: "From", Width: 160},
		{Title: "To", Width: 160},
		{Title: "Block", Width: 80, Align: ui.End},
	}, len(ops), func(row, col int) {
		op := ops[row]
		switch col {
		case 0:
			ui.Text(c, op.Type)
		case 1:
			ui.Text(c, op.From).SingleLine()
		case 2:
			ui.Text(c, op.To).SingleLine()
		case 3:
			ui.Text(c, strconv.FormatInt(op.BlockHeight, 10))
		}
	})
}

func (a *App) renderNFTBurnDialog(c *ui.Context) {
	if !a.nft.askBurn {
		return
	}
	// Say which asset is about to be destroyed. A name is friendlier than a
	// UUID, but only when the Inspect card loaded that same asset.
	detail := "The asset is destroyed and cannot be recovered."
	if id := strings.TrimSpace(a.nft.burnID); id != "" {
		target := shortID(id)
		if a.nft.nft != nil && a.nft.nft.ID == id && a.nft.nft.Name != "" {
			target = a.nft.nft.Name + " (" + shortID(id) + ")"
		}
		detail = fmt.Sprintf("Burn %s? The asset is destroyed and cannot be recovered.", target)
	}
	switch ui.AlertDialog(c, &a.nft.askBurn, "Burn this NFT?", detail, "Cancel", "Burn") {
	case 1:
		id := strings.TrimSpace(a.nft.burnID)
		err := a.services.BurnNFT(id, strings.TrimSpace(a.nft.burnOwner), strings.TrimSpace(a.nft.burnPriv))
		a.nft.status = wrapErr(err)
		if err == nil {
			// The same refresh clears the Inspect readout (the asset is gone)
			// and re-reads the history, which now ends with the burn.
			a.refreshNFT(id)
		}
	}
}
