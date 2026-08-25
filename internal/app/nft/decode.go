package nft

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/nft"
)

// decodeKey decodes a base64-encoded key/address request field. A
// syntactically invalid value (e.g. "!!!") is a client error wrapped in
// nft.ErrInvalidBase64, mapped to HTTP 400 INVALID_BASE64 by the handler
// classification table. Before this helper the raw stdlib decode error
// surfaced as an unclassified 500 (TASK-095, ISS-089).
func decodeKey(field, s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid %s encoding: %w", field, nft.ErrInvalidBase64)
	}
	return b, nil
}
