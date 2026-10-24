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
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			a.nftMint(c)
			a.nftRead(c)
			a.nftWrite(c)
			a.renderNFTBurnDialog(c)
		})
	})
}

func (a *App) nftMint(c *ui.Context) {
	sectionTitle(c, "Mint NFT", "Create a unique asset owned by the creator key.")
	field(c, "Name", &a.nft.name)
	field(c, "Description (optional)", &a.nft.description)
	field(c, "Image URL (optional)", &a.nft.imageURL)
	field(c, "Token URI (optional)", &a.nft.tokenURI)
	fieldHint(c, "Creator (base64 public key)", "The creator becomes the first owner.", &a.nft.creator)
	if ui.PrimaryButton(c, "Mint NFT").Clicked() {
		n, err := a.services.MintNFT(
			strings.TrimSpace(a.nft.name), strings.TrimSpace(a.nft.description),
			strings.TrimSpace(a.nft.imageURL), strings.TrimSpace(a.nft.tokenURI),
			strings.TrimSpace(a.nft.creator))
		if err != nil {
			a.nft.status = errStatus(err)
		} else {
			a.nft.status = okStatus("Minted %s (%s) at block %d", n.Name, n.ID, n.BlockHeight)
			a.nft.id = n.ID
			a.nft.name, a.nft.description, a.nft.imageURL, a.nft.tokenURI = "", "", "", ""
		}
	}
	renderStatus(c, a.nft.status)
}

func (a *App) nftRead(c *ui.Context) {
	ui.Divider(c)
	sectionTitle(c, "Inspect", "Look up one NFT by id, or list everything an owner holds.")
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
		ui.Text(c, n.Name).FontSize(15).Bold()
		if n.Description != "" {
			ui.Text(c, n.Description).FontSize(12)
		}
		ui.Text(c, "Owner "+n.Owner).FontSize(12).TextColor(ui.Hex("#6b7c7f"))
		ui.Text(c, "Creator "+n.Creator).FontSize(12).TextColor(ui.Hex("#6b7c7f"))
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
	ui.Divider(c)
	sectionTitle(c, "Transfer", "Move an NFT to a new owner; the transfer is signed by the current one.")
	field(c, "NFT ID", &a.nft.id)
	field(c, "From (base64 public key)", &a.nft.from)
	field(c, "To (base64 public key)", &a.nft.to)
	field(c, "From Private Key", &a.nft.priv)
	if ui.Button(c, "Transfer").Clicked() {
		a.nft.status = wrapErr(a.services.TransferNFT(
			strings.TrimSpace(a.nft.id), strings.TrimSpace(a.nft.from),
			strings.TrimSpace(a.nft.to), strings.TrimSpace(a.nft.priv)))
	}

	ui.Divider(c)
	sectionTitle(c, "Burn", "Destroy an NFT permanently. This cannot be undone.")
	field(c, "NFT ID", &a.nft.id)
	field(c, "Owner (base64 public key)", &a.nft.burnOwner)
	field(c, "Owner Private Key", &a.nft.burnPriv)
	if ui.Button(c, "Burn NFT").Clicked() {
		a.nft.askBurn = true
	}

	ui.Divider(c)
	sectionTitle(c, "History", "Every mint, transfer and burn recorded for this NFT.")
	field(c, "NFT ID", &a.nft.id)
	if ui.Button(c, "Show History").Clicked() {
		ops, err := a.services.NFTHistory(strings.TrimSpace(a.nft.id))
		if err != nil {
			a.nft.status = errStatus(err)
			a.nft.history = nil
		} else {
			a.nft.history = ops
			a.nft.status = statusLine{}
		}
	}
	if len(a.nft.history) == 0 {
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
			strings.TrimSpace(a.nft.id), strings.TrimSpace(a.nft.burnOwner),
			strings.TrimSpace(a.nft.burnPriv)))
	}
}
