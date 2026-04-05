package nft

import "time"

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

func (n *NFT) Validate() error {
	if n.Name == "" {
		return ErrNameRequired
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
		NFTID:     nftID,
		Type:      opType,
		From:      from,
		To:        to,
		Signature: signature,
		Timestamp: time.Now().Unix(),
	}
}
