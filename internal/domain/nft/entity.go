// Package nft provides Ed25519-signed NFT (Non-Fungible Token) functionality.
// NFTs are transferred using cryptographic signatures for secure ownership.
package nft

import (
	"time"

	"github.com/google/uuid"
)

// NFT represents a non-fungible token with ownership tracking.
type NFT struct {
	ID          string
	Name        string
	Description string
	ImageURL    string
	TokenURI    string
	Owner       []byte
	Creator     []byte
	BlockHeight int64
	Timestamp   int64
}

// Operation represents a NFT transfer operation with cryptographic signature.
type Operation struct {
	ID          string
	NFTID       string
	Type        string
	From        []byte
	To          []byte
	Signature   []byte
	BlockHeight int64
	Timestamp   int64
}

// Length bounds for NFT free-text fields. Mint previously persisted these
// without bound while the token surface capped name/symbol; the caps here
// mirror the token validator and are enforced at the shared domain edge so
// REST/CLI/web all inherit them (TASK-271, ISS-267).
const (
	MaxNFTNameLength        = 200
	MaxNFTDescriptionLength = 2000
	MaxNFTImageURLLength    = 2000
	MaxNFTTokenURILength    = 2000
)

func (n *NFT) Validate() error {
	if n.Name == "" {
		return ErrNameRequired
	}
	if len(n.Name) > MaxNFTNameLength {
		return ErrNameTooLong
	}
	if len(n.Description) > MaxNFTDescriptionLength {
		return ErrDescriptionTooLong
	}
	if len(n.ImageURL) > MaxNFTImageURLLength {
		return ErrImageURLTooLong
	}
	if len(n.TokenURI) > MaxNFTTokenURILength {
		return ErrTokenURITooLong
	}
	if len(n.Owner) == 0 {
		return ErrOwnerRequired
	}
	return nil
}

func (n *NFT) IsOwner(pubKey []byte) bool {
	if len(n.Owner) != len(pubKey) {
		return false
	}
	for i := range n.Owner {
		if n.Owner[i] != pubKey[i] {
			return false
		}
	}
	return true
}

func (o *Operation) IsTransfer() bool {
	return o.Type == "transfer"
}

func (o *Operation) IsMint() bool {
	return o.Type == "mint"
}

func (o *Operation) IsBurn() bool {
	return o.Type == "burn"
}

func NewNFT(name, description, imageURL, tokenURI string, creator, owner []byte) *NFT {
	return &NFT{
		Name:        name,
		Description: description,
		ImageURL:    imageURL,
		TokenURI:    tokenURI,
		Creator:     creator,
		Owner:       owner,
		Timestamp:   time.Now().Unix(),
	}
}

func NewOperation(nftID, opType string, from, to, signature []byte) *Operation {
	return &Operation{
		// Each operation gets its own UUID id (ISS-072). Without it every row
		// shared the PRIMARY KEY "" and the SQLite INSERT OR REPLACE collapsed
		// the whole per-NFT audit history to a single row.
		ID:        uuid.New().String(),
		NFTID:     nftID,
		Type:      opType,
		From:      from,
		To:        to,
		Signature: signature,
		Timestamp: time.Now().Unix(),
	}
}
