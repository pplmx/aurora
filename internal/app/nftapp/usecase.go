package nftapp

import (
	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
)

type Mint struct {
	service nft.Service
	chain   blockchain.BlockWriter
}

func NewMint(service nft.Service, chain blockchain.BlockWriter) *Mint {
	return &Mint{service: service, chain: chain}
}

func (u *Mint) Execute(req *MintNFTRequest) (*NFTResponse, error) {
	creator, err := decodeKey("creator", req.Creator, pubKeyLen, nft.ErrInvalidPublicKey)
	if err != nil {
		return nil, err
	}

	n := nft.NewNFT(req.Name, req.Description, req.ImageURL, req.TokenURI, creator, creator)
	n.Owner = creator
	n.Creator = creator
	// Validate via the shared domain entity (not just a non-empty name check)
	// so the name/description/image_url/token_uri length bounds REST/CLI/web
	// all inherit are enforced here too (TASK-271, ISS-267).
	if err := n.Validate(); err != nil {
		return nil, err
	}

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
	from, err := decodeKey("from", req.From, pubKeyLen, nft.ErrInvalidPublicKey)
	if err != nil {
		return nil, err
	}

	to, err := decodeKey("to", req.To, pubKeyLen, nft.ErrInvalidPublicKey)
	if err != nil {
		return nil, err
	}
	if err := nft.ValidateRecipient(to); err != nil {
		return nil, err
	}

	privateKey, err := decodeKey("privatekey", req.PrivateKey, privKeyLen, nft.ErrInvalidPrivateKey)
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
	owner, err := decodeKey("owner", req.Owner, pubKeyLen, nft.ErrInvalidPublicKey)
	if err != nil {
		return err
	}

	privateKey, err := decodeKey("privatekey", req.PrivateKey, privKeyLen, nft.ErrInvalidPrivateKey)
	if err != nil {
		return err
	}

	return u.service.Burn(req.NFTID, owner, privateKey, u.chain)
}

type Get struct {
	service nft.Service
}

func NewGet(service nft.Service) *Get {
	return &Get{service: service}
}

func (u *Get) Execute(id string) (*NFTResponse, error) {
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
// unbounded (0,0 — the CLI default); the REST layer always sends a
// bounded limit so a key-holding caller cannot force an unbounded response
// (TASK-101, ISS-093).
type ListNFTsByOwnerRequest struct {
	Owner  string
	Limit  int
	Offset int
}

// ListNFTsRequest pages the whole collection.
type ListNFTsRequest struct {
	Limit  int
	Offset int
}

type ListByOwner struct {
	service nft.Service
}

func NewListByOwner(service nft.Service) *ListByOwner {
	return &ListByOwner{service: service}
}

func (u *ListByOwner) Execute(req *ListNFTsByOwnerRequest) ([]*NFTResponse, error) {
	owner, err := decodeKey("owner", req.Owner, pubKeyLen, nft.ErrInvalidPublicKey)
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

// ListAllNFTsUseCase returns every NFT, newest first. ListByOwner needs an
// owner key, which a caller who did not mint the asset does not have.
type ListAllNFTsUseCase struct {
	service nft.Service
}

func NewListAll(service nft.Service) *ListAllNFTsUseCase {
	return &ListAllNFTsUseCase{service: service}
}

func (u *ListAllNFTsUseCase) Execute(req *ListNFTsRequest) ([]*NFTResponse, error) {
	results, err := u.service.ListNFTs(req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	responses := make([]*NFTResponse, 0, len(results))
	for _, n := range results {
		responses = append(responses, ToNFTResponse(n))
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
	// Burn deletes the NFT row but deliberately keeps nft_operations as its
	// permanent audit trail. Read the trail first so a burned NFT remains
	// inspectable through `nft history` and GET /nft/{id}/history.
	results, err := u.service.GetOperations(nftID)
	if err != nil {
		return nil, err
	}

	// A history for a truly nonexistent NFT must still 404, not return an
	// empty 200 [] list (ISS-130). An existing NFT with no operations keeps
	// returning 200 [].
	if len(results) == 0 {
		nftByID, err := u.service.GetNFTByID(nftID)
		if err != nil {
			return nil, err
		}
		if nftByID == nil {
			return nil, nft.ErrNFTNotFound
		}
	}

	responses := make([]*OperationResponse, len(results))
	for i, op := range results {
		responses[i] = ToOperationResponse(op)
	}
	return responses, nil
}
