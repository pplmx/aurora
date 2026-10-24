package token

import (
	"bytes"
	"crypto/ed25519"
)

const (
	MaxTokenNameLength   = 100
	MaxTokenSymbolLength = 10
)

func ValidateTokenName(name string) error {
	if name == "" {
		return ErrTokenNameRequired
	}
	if len(name) > MaxTokenNameLength {
		return ErrTokenNameTooLong
	}
	return nil
}

func ValidateTokenSymbol(symbol string) error {
	if symbol == "" {
		return ErrTokenSymbolRequired
	}
	if len(symbol) > MaxTokenSymbolLength {
		return ErrTokenSymbolTooLong
	}
	return nil
}

// ValidateTokenDecimals rejects negative decimals values. 0 is a valid
// explicit value (no fractional units); omission is represented by a nil
// pointer at the request boundary. int8 bounds values to 0..127.
func ValidateTokenDecimals(decimals int8) error {
	if decimals < 0 {
		return ErrTokenDecimalsInvalid
	}
	return nil
}

func ValidateAmount(amount *Amount) error {
	if amount == nil || !amount.IsPositive() {
		return ErrAmountMustBePositive
	}
	// The persistence layer stores balances, supply, and allowances as
	// signed 64-bit SQLite INTEGERs. Reject amounts that would silently
	// overflow/clamp there; BitLen() > 63 means > MaxInt64.
	if amount.BitLen() > 63 {
		return ErrAmountTooLarge
	}
	return nil
}

func ValidatePublicKey(pk PublicKey) error {
	if len(pk) == 0 {
		return ErrPublicKeyRequired
	}
	if len(pk) != ed25519.PublicKeySize {
		return ErrInvalidPublicKeyLength
	}
	return nil
}

// ValidateOwner rejects the all-zero address as a token owner. The owner
// receives the initial supply and is the only key authorized to mint, so no
// signer could ever recover either if this value were accepted.
func ValidateOwner(pk PublicKey) error {
	if err := ValidatePublicKey(pk); err != nil {
		return err
	}
	if isZeroAddress(pk) {
		return ErrZeroOwner
	}
	return nil
}

// ValidateRecipient applies the extra rule that credit-producing operations
// need: the all-zero address is syntactically a 32-byte public key, but no
// signer can control it, so minting or transferring to it destroys the value.
func ValidateRecipient(pk PublicKey) error {
	if err := ValidatePublicKey(pk); err != nil {
		return err
	}
	if isZeroAddress(pk) {
		return ErrTransferToZero
	}
	return nil
}

func isZeroAddress(pk PublicKey) bool {
	for _, b := range pk {
		if b != 0 {
			return false
		}
	}
	return true
}

// ValidatePrivateKey checks that the private key has the correct length
// for an Ed25519 private key (seed + public key = 64 bytes).
func ValidatePrivateKey(priv PrivateKey) error {
	if len(priv) == 0 {
		return ErrPrivateKeyRequired
	}
	if len(priv) != ed25519.PrivateKeySize {
		return ErrInvalidPrivateKeyLength
	}
	return nil
}

// VerifyPrivateKeyMatches checks that the given private key corresponds
// to the given public key. Returns an error if they don't match.
func VerifyPrivateKeyMatches(pub PublicKey, priv PrivateKey) error {
	if err := ValidatePublicKey(pub); err != nil {
		return err
	}
	if err := ValidatePrivateKey(priv); err != nil {
		return err
	}
	pubFromPriv, ok := ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
	if !ok || !bytes.Equal(pubFromPriv, pub) {
		return ErrUnauthorized
	}
	return nil
}
