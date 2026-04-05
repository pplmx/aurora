package nft

import (
	"encoding/base64"
	"fmt"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
)

type MintNFTUseCase struct {
	service nft.Service
}

func NewMintNFTUseCase(service nft.Service) *MintNFTUseCase {
	return &MintNFTUseCase{service: service}
}

func (u *MintNFTUseCase) Execute(req *MintNFTRequest) (*NFTResponse, error) {
	creator, err := base64.StdEncoding.DecodeString(req.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator: %w", err)
	}

	n := nft.NewNFT(req.Name, req.Description, req.ImageURL, req.TokenURI, creator, creator)
	n.Owner = creator
	n.Creator = creator

	chain := blockchain.InitBlockChain()
	result, err := u.service.Mint(n, chain)
	if err != nil {
		return nil, err
	}

	return ToNFTResponse(result), nil
}

type TransferNFTUseCase struct {
	service nft.Service
}

func NewTransferNFTUseCase(service nft.Service) *TransferNFTUseCase {
	return &TransferNFTUseCase{service: service}
}

func (u *TransferNFTUseCase) Execute(req *TransferNFTRequest) (*OperationResponse, error) {
	from, err := base64.StdEncoding.DecodeString(req.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from: %w", err)
	}

	to, err := base64.StdEncoding.DecodeString(req.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to: %w", err)
	}

	privateKey, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	chain := blockchain.InitBlockChain()
	result, err := u.service.Transfer(req.NFTID, from, to, privateKey, chain)
	if err != nil {
		return nil, err
	}

	return ToOperationResponse(result), nil
}

type BurnNFTUseCase struct {
	service nft.Service
}

func NewBurnNFTUseCase(service nft.Service) *BurnNFTUseCase {
	return &BurnNFTUseCase{service: service}
}

func (u *BurnNFTUseCase) Execute(req *BurnNFTRequest) error {
	owner, err := base64.StdEncoding.DecodeString(req.Owner)
	if err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}

	privateKey, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	chain := blockchain.InitBlockChain()
	return u.service.Burn(req.NFTID, owner, privateKey, chain)
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

type ListNFTsByOwnerUseCase struct {
	service nft.Service
}

func NewListNFTsByOwnerUseCase(service nft.Service) *ListNFTsByOwnerUseCase {
	return &ListNFTsByOwnerUseCase{service: service}
}

func (u *ListNFTsByOwnerUseCase) Execute(ownerB64 string) ([]*NFTResponse, error) {
	owner, err := base64.StdEncoding.DecodeString(ownerB64)
	if err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}

	results, err := u.service.GetNFTsByOwner(owner)
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
