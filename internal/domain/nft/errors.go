package nft

import "errors"

var (
	ErrNameRequired     = errors.New("NFT name is required")
	ErrOwnerRequired    = errors.New("NFT owner is required")
	ErrNotOwner         = errors.New("not the owner")
	ErrNFTNotFound      = errors.New("NFT not found")
	ErrStorageNotInit   = errors.New("storage not initialized")
	ErrInvalidSignature = errors.New("invalid signature")
)
