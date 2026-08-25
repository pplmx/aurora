package token

import "errors"

var (
	ErrTokenNotFound         = errors.New("token not found")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrInsufficientAllowance = errors.New("insufficient allowance")
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNonceTooLow           = errors.New("nonce too low")
	ErrAmountMustBePositive  = errors.New("amount must be positive")
	ErrNotTokenOwner         = errors.New("not token owner")
	ErrTokenNotMintable      = errors.New("token not mintable")
	ErrTokenNotBurnable      = errors.New("token not burnable")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrTransferToZero        = errors.New("cannot transfer to zero address")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrDuplicateTransfer     = errors.New("duplicate transfer")
	// ErrTokenExists is returned when a create tries to register a token whose
	// symbol (which is the token's primary key / ID) is already taken. The
	// persistence layer uses INSERT OR REPLACE keyed on the symbol, so without
	// this guard a duplicate create would silently overwrite the existing
	// token's rows and balances (see TokenService.CreateToken).
	ErrTokenExists = errors.New("token already exists")

	// Validation errors for token name, symbol, and key validation.
	// Using sentinels allows API handlers to map these to 400 Bad Request.
	ErrTokenNameRequired       = errors.New("token name is required")
	ErrTokenNameTooLong        = errors.New("token name too long")
	ErrTokenSymbolRequired     = errors.New("token symbol is required")
	ErrTokenSymbolTooLong      = errors.New("token symbol too long")
	ErrPublicKeyRequired       = errors.New("public key is required")
	ErrInvalidPublicKeyLength  = errors.New("invalid public key length")
	ErrPrivateKeyRequired      = errors.New("private key is required")
	ErrInvalidPrivateKeyLength = errors.New("invalid private key length")
	// ErrInvalidBase64 is returned when a base64-encoded key/address field
	// cannot be decoded at all — a client input error (HTTP 400), not a server
	// fault. Previously the raw decode error escaped unclassified and surfaced
	// as 500 INTERNAL_ERROR (TASK-095, ISS-089).
	ErrInvalidBase64 = errors.New("invalid base64 encoding")

	// ErrAmountTooLarge is returned when an amount exceeds the range that
	// the persistence layer can store exactly (signed 64-bit). Accepting
	// larger amounts would let them silently overflow/clamp in SQLite's
	// INTEGER math, corrupting balances, supply, and allowances.
	ErrAmountTooLarge = errors.New("amount too large")
)
