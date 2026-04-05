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
)
