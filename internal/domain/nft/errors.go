package nft

import "errors"

var (
	ErrNameRequired      = errors.New("NFT name is required")
	ErrOwnerRequired     = errors.New("NFT owner is required")
	ErrNotOwner          = errors.New("not the owner")
	ErrNFTNotFound       = errors.New("NFT not found")
	ErrStorageNotInit    = errors.New("storage not initialized")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidPrivateKey = errors.New("invalid private key length")
	ErrInvalidPublicKey  = errors.New("invalid public key length")
	// ErrKeyMismatch guards identity authenticity: the private key presented
	// must correspond to the public key it claims to represent (from/owner).
	// Without this, a caller who only knows a public key could forge a
	// transfer/burn with their own key.
	ErrKeyMismatch = errors.New("private key does not match the claimed owner")
)
