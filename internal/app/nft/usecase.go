package nft

import (
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
)

type MintNFTUseCase struct {
	service nft.Service
	chain   blockchain.BlockWriter
}

func NewMintNFTUseCase(service nft.Service, chain blockchain.BlockWriter) *MintNFTUseCase {
	return &MintNFTUseCase{service: service, chain: chain}
}

func (u *MintNFTUseCase) Execute(req *MintNFTRequest) (*NFTResponse, error) {
	if req.Name == "" {
		return nil, nft.ErrNameRequired
	}

	creator, err := decodeKey("creator", req.Creator)
	if err != nil {
		return nil, err
	}

	n := nft.NewNFT(req.Name, req.Description, req.ImageURL, req.TokenURI, creator, creator)
	n.Owner = creator
	n.Creator = creator

	result, err := u.service.Mint(n, u.chain)
	if err != nil {
		return nil, err
	}

	return ToNFTResponse(result), nil
}

type TransferNFTUseCase struct {
	service nft.Service
	chain   blockchain.BlockWriter
}

func NewTransferNFTUseCase(service nft.Service, chain blockchain.BlockWriter) *TransferNFTUseCase {
	return &TransferNFTUseCase{service: service, chain: chain}
}

func (u *TransferNFTUseCase) Execute(req *TransferNFTRequest) (*OperationResponse, error) {
	from, err := decodeKey("from", req.From)
	if err != nil {
		return nil, err
	}

	to, err := decodeKey("to", req.To)
	if err != nil {
		return nil, err
	}

	privateKey, err := decodeKey("privatekey", req.PrivateKey)
	if err != nil {
		return nil, err
	}

	result, err := u.service.Transfer(req.NFTID, from, to, privateKey, u.chain)
	if err != nil {
		return nil, err
	}

	return ToOperationResponse(result), nil
}

type BurnNFTUseCase struct {
	service nft.Service
	chain   blockchain.BlockWriter
}

func NewBurnNFTUseCase(service nft.Service, chain blockchain.BlockWriter) *BurnNFTUseCase {
	return &BurnNFTUseCase{service: service, chain: chain}
}

func (u *BurnNFTUseCase) Execute(req *BurnNFTRequest) error {
	owner, err := decodeKey("owner", req.Owner)
	if err != nil {
		return err
	}

	privateKey, err := decodeKey("privatekey", req.PrivateKey)
	if err != nil {
		return err
	}

	return u.service.Burn(req.NFTID, owner, privateKey, u.chain)
}

type GetNFTUseCase struct {
	service nft.Service
}

func NewGetNFTUseCase(service nft.Service) *GetNFTUseCase {
	return &GetNFTUseCase{service: service}
}

func (u *GetNFTUseCase) Execute(id string) (*NFTResponse, error) {
	result, err := u.service.GetNFTByID(id)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nft.ErrNFTNotFound
	}
	return ToNFTResponse(result), nil
}

// ListNFTsByOwnerRequest pages an owner's NFT collection. Limit <= 0 is
// unbounded (0,0 — the CLI/TUI default); the REST layer always sends a
// bounded limit so a key-holding caller cannot force an unbounded response
// (TASK-101, ISS-093).
type ListNFTsByOwnerRequest struct {
	Owner  string
	Limit  int
	Offset int
}

type ListNFTsByOwnerUseCase struct {
	service nft.Service
}

func NewListNFTsByOwnerUseCase(service nft.Service) *ListNFTsByOwnerUseCase {
	return &ListNFTsByOwnerUseCase{service: service}
}

func (u *ListNFTsByOwnerUseCase) Execute(req *ListNFTsByOwnerRequest) ([]*NFTResponse, error) {
	owner, err := decodeKey("owner", req.Owner)
	if err != nil {
		return nil, err
	}

	results, err := u.service.GetNFTsByOwner(owner, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*NFTResponse, len(results))
	for i, n := range results {
		responses[i] = ToNFTResponse(n)
	}
	return responses, nil
}

type GetNFTOperationsUseCase struct {
	service nft.Service
}

func NewGetNFTOperationsUseCase(service nft.Service) *GetNFTOperationsUseCase {
	return &GetNFTOperationsUseCase{service: service}
}

func (u *GetNFTOperationsUseCase) Execute(nftID string) ([]*OperationResponse, error) {
	results, err := u.service.GetOperations(nftID)
	if err != nil {
		return nil, err
	}

	responses := make([]*OperationResponse, len(results))
	for i, op := range results {
		responses[i] = ToOperationResponse(op)
	}
	return responses, nil
}
