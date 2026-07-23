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

func ValidateAmount(amount *Amount) error {
	if amount == nil || !amount.IsPositive() {
		return ErrAmountMustBePositive
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

func ValidateNonce(nonce uint64, currentNonce uint64) error {
	if nonce <= currentNonce {
		return ErrNonceTooLow
	}
	return nil
}
