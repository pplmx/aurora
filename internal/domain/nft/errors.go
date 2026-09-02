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
	// ErrInvalidBase64 is returned when a base64-encoded key/address field
	// cannot be decoded at all — a client input error (HTTP 400), not a server
	// fault (TASK-095, ISS-089).
	ErrInvalidBase64 = errors.New("invalid base64 encoding")
	// ErrKeyMismatch guards identity authenticity: the private key presented
	// must correspond to the public key it claims to represent (from/owner).
	// Without this, a caller who only knows a public key could forge a
	// transfer/burn with their own key.
	ErrKeyMismatch = errors.New("private key does not match the claimed owner")

	// Length-bound sentinels. NFT mint previously persisted unbounded
	// description/image_url/token_uri strings while the name got only a
	// non-empty check — a key-holding caller could grow rows and list/detail
	// responses without bound, unlike the token surface's caps (TASK-271,
	// ISS-267). These are enforced in NFT.Validate() at the shared domain edge
	// so REST/CLI/web all inherit them.
	ErrNameTooLong        = errors.New("NFT name is too long")
	ErrDescriptionTooLong = errors.New("NFT description is too long")
	ErrImageURLTooLong    = errors.New("NFT image URL is too long")
	ErrTokenURITooLong    = errors.New("NFT token URI is too long")
)
