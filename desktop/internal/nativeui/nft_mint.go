package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) nftMint(c *ui.Context) {
	formCard(c, "Mint NFT", "Create a unique asset owned by the creator key.", func() {
		a.nftMintBody(c)
	})
}

func (a *App) nftMintBody(c *ui.Context) {
	generated := false
	field(c, "Name", &a.nft.name)
	ui.Collapsible(c, tr("Optional details"), &a.nft.mintAdvanced, func() {
		textArea(c, "Description (optional)", &a.nft.description)
		field(c, "Image URL (optional)", &a.nft.imageURL)
		field(c, "Token URI (optional)", &a.nft.tokenURI)
	})
	fieldHint(c, "Creator (base64 public key)", "The creator becomes the first owner.", &a.nft.creator)
	ui.Row(c).Gap(8).Children(func() {
		if secondaryButton(c, "Generate key pair").Bind(&a.nft.generateControl).Clicked() {
			generated = a.generateNFTCreator()
		}
		if a.nft.creatorPriv != "" && quietButton(c, "Dismiss key").Clicked() {
			a.nft.creatorPriv = ""
			a.nft.generateControl.Focus(c)
		}
	})
	if a.nft.creatorPriv != "" {
		copyRow(c, "Creator private key", a.nft.creatorPriv, func() {
			a.nft.status = okStatus("Creator private key copied.")
		}).Bind(&a.nft.creatorKeyCopy)
		ui.Text(c, tr("This is the only copy of the creator key; the store keeps the public half.")).
			FontSize(textSize(c, 11)).TextColor(c.Theme().Warning)
	}
	if generated {
		a.nft.creatorKeyCopy.Focus(c)
	}
	ui.Row(c).Children(func() {
		if primaryButton(c, "Mint NFT", allSet(a.nft.name, a.nft.creator)).Clicked() {
			a.mintNFT()
		}
	})
}

// generateNFTCreator fills the Creator field from a fresh Ed25519 key pair and
// offers the private half for copying, the same handover as a token owner.
func (a *App) generateNFTCreator() bool {
	pub, priv, err := a.services.GenerateKeyPair()
	if err != nil {
		a.nft.status = errStatus(err)
		return false
	}
	a.nft.creator, a.nft.creatorPriv = pub, priv
	// A new pair belongs to no asset yet.
	a.nft.creatorKeyNFTID = ""
	a.nft.status = okStatus("Generated a creator key pair. Copy the private key now; it is not stored.")
	return true
}

// mintNFT mints the asset described by the Mint card's inputs.
func (a *App) mintNFT() {
	name, description := strings.TrimSpace(a.nft.name), strings.TrimSpace(a.nft.description)
	imageURL, tokenURI := strings.TrimSpace(a.nft.imageURL), strings.TrimSpace(a.nft.tokenURI)
	creator, creatorPriv := strings.TrimSpace(a.nft.creator), a.nft.creatorPriv
	if rejectIncomplete(&a.nft.status, "NFT name and creator are required.", name, creator) {
		return
	}
	a.runWrite("Minting NFT…", func() func() {
		n, err := a.services.MintNFT(name, description, imageURL, tokenURI, creator)
		return func() {
			if err != nil {
				a.nft.status = errStatus(err)
				return
			}
			a.rememberSuccessfulKey(n.Creator, creatorPriv)
			a.nft.status = okStatus("Minted %s (%s) at block %d", n.Name, n.ID, n.BlockHeight)
			// A just-minted asset is the page's subject. Clear a filter or search
			// that would hide it, highlight its row, and move the ids and open
			// readouts to it; leaving the collection on the previous owner's filter
			// made a successful mint look like nothing happened.
			a.nft.filterOwner, a.nft.owner = "", ""
			a.nft.search, a.nft.appliedSearch = "", ""
			a.loadNFTPage(0)
			// The new asset becomes the subject of every other card: the operator
			// just created it, so making them retype the id four times would be
			// busywork.
			a.nft.id, a.nft.transferID, a.nft.burnID, a.nft.historyID = n.ID, n.ID, n.ID, n.ID
			a.nft.creatorKeyNFTID = n.ID
			a.applyNFTSubject(n.Owner)
			a.selectNFTRow(n.ID)
			a.selectNFT(n.ID)
			a.nft.name, a.nft.description, a.nft.imageURL, a.nft.tokenURI = "", "", "", ""
		}
	})
}
