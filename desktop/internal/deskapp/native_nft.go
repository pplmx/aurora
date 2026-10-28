package deskapp

import (
	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/nftapp"
	"github.com/pplmx/aurora/internal/domain/nft"
)

// --- NFT -----------------------------------------------------------------

func (s *nativeServices) MintNFT(name, description, imageURL, tokenURI, creator string) (nativeui.NFTRow, error) {
	uc := nftapp.NewMint(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	resp, err := uc.Execute(&nftapp.MintNFTRequest{
		Name: name, Description: description, ImageURL: imageURL,
		TokenURI: tokenURI, Creator: creator,
	})
	if err != nil {
		return nativeui.NFTRow{}, err
	}
	return nativeui.NFTRow{
		ID: resp.ID, Name: resp.Name, Description: resp.Description,
		Owner: resp.Owner, Creator: resp.Creator, ImageURL: resp.ImageURL,
		TokenURI: resp.TokenURI, BlockHeight: resp.BlockHeight,
	}, nil
}

func (s *nativeServices) GetNFT(id string) (nativeui.NFTRow, error) {
	uc := nftapp.NewGet(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(id)
	if err != nil {
		return nativeui.NFTRow{}, err
	}
	return nativeui.NFTRow{
		ID: resp.ID, Name: resp.Name, Description: resp.Description,
		Owner: resp.Owner, Creator: resp.Creator, ImageURL: resp.ImageURL,
		TokenURI: resp.TokenURI, BlockHeight: resp.BlockHeight,
	}, nil
}

func (s *nativeServices) ListNFTs(owner string) ([]nativeui.NFTRow, error) {
	return s.ListNFTsPage(owner, 0, 0)
}

func (s *nativeServices) ListNFTsPage(owner string, limit, offset int) ([]nativeui.NFTRow, error) {
	uc := nftapp.NewListByOwner(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(&nftapp.ListNFTsByOwnerRequest{Owner: owner, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.NFTRow, 0, len(resp))
	for _, n := range resp {
		rows = append(rows, nativeui.NFTRow{
			ID: n.ID, Name: n.Name, Description: n.Description,
			Owner: n.Owner, Creator: n.Creator, BlockHeight: n.BlockHeight,
		})
	}
	return rows, nil
}

func (s *nativeServices) ListAllNFTs() ([]nativeui.NFTRow, error) {
	return s.ListAllNFTsPage(0, 0)
}

func (s *nativeServices) ListAllNFTsPage(limit, offset int) ([]nativeui.NFTRow, error) {
	uc := nftapp.NewListAll(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(&nftapp.ListNFTsRequest{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.NFTRow, 0, len(resp))
	for _, n := range resp {
		rows = append(rows, nativeui.NFTRow{
			ID: n.ID, Name: n.Name, Description: n.Description,
			Owner: n.Owner, Creator: n.Creator, BlockHeight: n.BlockHeight,
		})
	}
	return rows, nil
}

func (s *nativeServices) TransferNFT(id, from, to, priv string) error {
	uc := nftapp.NewTransferNFTUseCase(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	_, err := uc.Execute(&nftapp.TransferNFTRequest{NFTID: id, From: from, To: to, PrivateKey: priv})
	return err
}

func (s *nativeServices) BurnNFT(id, owner, priv string) error {
	uc := nftapp.NewBurnNFTUseCase(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	return uc.Execute(&nftapp.BurnNFTRequest{NFTID: id, Owner: owner, PrivateKey: priv})
}

func (s *nativeServices) NFTHistory(id string, limit, offset int) ([]nativeui.NFTOpRow, error) {
	uc := nftapp.NewGetNFTOperationsUseCase(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(&nftapp.GetNFTOperationsRequest{
		NFTID: id, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.NFTOpRow, 0, len(resp))
	for _, op := range resp {
		rows = append(rows, nativeui.NFTOpRow{Type: op.Type, From: op.From, To: op.To, BlockHeight: op.BlockHeight})
	}
	return rows, nil
}

// joinCSV rebuilds the comma-separated participants string the lottery use case
// accepts from the slice a view keeps.
