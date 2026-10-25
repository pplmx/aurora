package nativeui

import (
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
}

func (a *App) loadNFT() {
	a.nft.loadErr = nil
}

func (a *App) nftView(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		renderStatus(c, a.nft.status)
		twoColumns(c, func() {
			a.nftMint(c)
			a.nftRead(c)
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
		if ui.PrimaryButton(c, "Mint NFT").Clicked() {
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
	// The new asset becomes the subject of every other card: the operator just
	// created it, so making them retype the id four times would be busywork.
	a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = n.ID, n.ID, n.ID, n.ID
	a.nft.name, a.nft.description, a.nft.imageURL, a.nft.tokenURI = "", "", "", ""
}

func (a *App) nftRead(c *ui.Context) {
	formCard(c, "Inspect", "Look up one NFT by id, or list everything an owner holds.", func() {
		a.nftReadBody(c)
	})
}

func (a *App) nftReadBody(c *ui.Context) {
	field(c, "NFT ID", &a.nft.id)
	if ui.Button(c, "Get NFT").Clicked() {
		n, err := a.services.GetNFT(strings.TrimSpace(a.nft.id))
		if err != nil {
			a.nft.status = errStatus(err)
			a.nft.nft = nil
		} else {
			a.nft.nft = &n
			a.nft.status = statusLine{}
		}
	}
	if n := a.nft.nft; n != nil {
		rows := []readoutRow{{"Name", n.Name}, {"Owner", n.Owner}, {"Creator", n.Creator}}
		if n.Description != "" {
			rows = append(rows, readoutRow{"About", n.Description})
		}
		readout(c, rows)
	}

	field(c, "Owner (base64 public key)", &a.nft.owner)
	if ui.Button(c, "List NFTs").Clicked() {
		rows, err := a.services.ListNFTs(strings.TrimSpace(a.nft.owner))
		if err != nil {
			a.nft.status = errStatus(err)
			a.nft.list = nil
		} else {
			a.nft.list = rows
			a.nft.status = statusLine{}
		}
	}
	if len(a.nft.list) == 0 {
		emptyNote(c, "No NFTs listed yet.")
		return
	}
	list := a.nft.list
	ui.Table(c, &a.nft.listTable, []ui.TableColumn{
		{Title: "ID", Width: 150},
		{Title: "Name"},
		{Title: "Block", Width: 70, Align: ui.End},
	}, len(list), func(row, col int) {
		n := list[row]
		switch col {
		case 0:
			ui.Text(c, n.ID).SingleLine()
		case 1:
			ui.Text(c, n.Name)
		case 2:
			ui.Text(c, strconv.FormatInt(n.BlockHeight, 10))
		}
	})
}

func (a *App) nftWrite(c *ui.Context) {
	formCard(c, "Transfer", "Move an NFT to a new owner; the transfer is signed by the current one.", func() {
		field(c, "NFT ID", &a.nft.transferID)
		field(c, "From (base64 public key)", &a.nft.from)
		field(c, "To (base64 public key)", &a.nft.to)
		field(c, "From Private Key", &a.nft.priv)
		ui.Row(c).Children(func() {
			if ui.PrimaryButton(c, "Transfer").Clicked() {
				a.transferNFT()
			}
		})
	})

	formCard(c, "Burn", "Destroy an NFT permanently. This cannot be undone.", func() {
		field(c, "NFT ID", &a.nft.burnID)
		field(c, "Owner (base64 public key)", &a.nft.burnOwner)
		field(c, "Owner Private Key", &a.nft.burnPriv)
		ui.Row(c).Children(func() {
			// A destructive action carries the danger colour, and the
			// confirmation still stands between it and the write.
			burn := ui.Button(c, "Burn NFT…").Background(ui.Hex(auroraRed)).TextColor(ui.Hex(night))
			if burn.Clicked() {
				a.nft.askBurn = true
			}
		})
	})

	formCard(c, "History", "Every mint, transfer and burn recorded for this NFT.", func() {
		field(c, "NFT ID", &a.nft.historyID)
		ui.Row(c).Children(func() {
			if ui.Button(c, "Show History").Clicked() {
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
	a.nft.status = wrapErr(a.services.TransferNFT(
		strings.TrimSpace(a.nft.transferID), strings.TrimSpace(a.nft.from),
		strings.TrimSpace(a.nft.to), strings.TrimSpace(a.nft.priv)))
}

func (a *App) nftHistoryTable(c *ui.Context) {
	if len(a.nft.history) == 0 {
		emptyNote(c, "No history loaded yet.")
		return
	}
	ops := a.nft.history
	ui.Table(c, &a.nft.historyTable, []ui.TableColumn{
		{Title: "Type", Width: 90},
		{Title: "From", Width: 160},
		{Title: "To", Width: 160},
		{Title: "Block", Width: 70, Align: ui.End},
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
	switch ui.AlertDialog(c, &a.nft.askBurn,
		"Burn this NFT?",
		"The asset is destroyed and cannot be recovered.",
		"Cancel", "Burn") {
	case 1:
		a.nft.status = wrapErr(a.services.BurnNFT(
			strings.TrimSpace(a.nft.burnID), strings.TrimSpace(a.nft.burnOwner),
			strings.TrimSpace(a.nft.burnPriv)))
	}
}
