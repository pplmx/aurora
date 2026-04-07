package token

import (
	"crypto/ed25519"
	"fmt"
)

const (
	MaxTokenNameLength   = 100
	MaxTokenSymbolLength = 10
)

func ValidateTokenName(name string) error {
	if name == "" {
		return fmt.Errorf("token name is required")
	}
	if len(name) > MaxTokenNameLength {
		return fmt.Errorf("token name too long")
	}
	return nil
}

func ValidateTokenSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("token symbol is required")
	}
	if len(symbol) > MaxTokenSymbolLength {
		return fmt.Errorf("token symbol too long")
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
		return fmt.Errorf("public key is required")
	}
	if len(pk) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length")
	}
	return nil
}

func ValidateNonce(nonce uint64, currentNonce uint64) error {
	if nonce <= currentNonce {
		return ErrNonceTooLow
	}
	return nil
}
