package nft

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/nft"
)

// decodeKey decodes a base64-encoded key/address request field and validates
// that it decodes to exactly wantLen bytes.
//
// A syntactically invalid value (e.g. "!!!") is a client error wrapped in
// nft.ErrInvalidBase64, mapped to HTTP 400 INVALID_BASE64 by the handler
// classification table. Before this helper the raw stdlib decode error
// surfaced as an unclassified 500 (TASK-095, ISS-089).
//
// The length check closes a mint-specific hole: mint used to accept any
// decoded length, so `nft mint -c <any-base64>` happily minted an NFT whose
// owner key was, say, 5 bytes — an NFT that could then never be transferred
// or burned (the domain's atomic ownership checks require a 32-byte/64-byte
// key, so they could never match), reported as minted successfully
// (TASK-112, ISS-104). wrongLen is the domain sentinel to surface: a public
// key (ErrInvalidPublicKey) for 32-byte fields, a private key
// (ErrInvalidPrivateKey) for 64-byte fields.
func decodeKey(field, s string, wantLen int, wrongLen error) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid %s encoding: %w", field, nft.ErrInvalidBase64)
	}
	if len(b) != wantLen {
		return nil, fmt.Errorf("invalid %s: %w", field, wrongLen)
	}
	return b, nil
}

const (
	pubKeyLen  = ed25519.PublicKeySize  // 32
	privKeyLen = ed25519.PrivateKeySize // 64
)
