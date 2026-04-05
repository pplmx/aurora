package nft

type Repository interface {
	SaveNFT(nft *NFT) error
	GetNFT(id string) (*NFT, error)
	GetNFTsByOwner(owner []byte) ([]*NFT, error)
	GetNFTsByCreator(creator []byte) ([]*NFT, error)
	UpdateNFT(nft *NFT) error
	DeleteNFT(id string) error
	SaveOperation(op *Operation) error
	GetOperations(nftID string) ([]*Operation, error)
}
