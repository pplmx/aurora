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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	TokenURI    string `json:"token_uri"`
	Owner       string `json:"owner"`
	Creator     string `json:"creator"`
	BlockHeight int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
}

type OperationResponse struct {
	ID          string `json:"id"`
	NFTID       string `json:"nft_id"`
	Type        string `json:"type"`
	From        string `json:"from"`
	To          string `json:"to"`
	BlockHeight int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
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
