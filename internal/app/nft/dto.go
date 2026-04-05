package nft

import "github.com/pplmx/aurora/internal/domain/nft"

type MintNFTRequest struct {
	Name        string
	Description string
	ImageURL    string
	TokenURI    string
	Creator     string
}

type TransferNFTRequest struct {
	NFTID      string
	From       string
	To         string
	PrivateKey string
}

type BurnNFTRequest struct {
	NFTID      string
	Owner      string
	PrivateKey string
}

type NFTResponse struct {
	ID          string
	Name        string
	Description string
	ImageURL    string
	TokenURI    string
	Owner       string
	Creator     string
	BlockHeight int64
	Timestamp   int64
}

type OperationResponse struct {
	ID          string
	NFTID       string
	Type        string
	From        string
	To          string
	BlockHeight int64
	Timestamp   int64
}

func ToNFTResponse(nft *nft.NFT) *NFTResponse {
	if nft == nil {
		return nil
	}
	return &NFTResponse{
		ID:          nft.ID,
		Name:        nft.Name,
		Description: nft.Description,
		ImageURL:    nft.ImageURL,
		TokenURI:    nft.TokenURI,
		Owner:       string(nft.Owner),
		Creator:     string(nft.Creator),
		BlockHeight: nft.BlockHeight,
		Timestamp:   nft.Timestamp,
	}
}

func ToOperationResponse(op *nft.Operation) *OperationResponse {
	if op == nil {
		return nil
	}
	return &OperationResponse{
		ID:          op.ID,
		NFTID:       op.NFTID,
		Type:        op.Type,
		From:        string(op.From),
		To:          string(op.To),
		BlockHeight: op.BlockHeight,
		Timestamp:   op.Timestamp,
	}
}
