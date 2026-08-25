package token

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/token"
)

// decodeKey decodes a base64-encoded key/address request field. A
// syntactically invalid value (e.g. "!!!") is a client error wrapped in
// token.ErrInvalidBase64, which the handler classification table maps to
// HTTP 400 INVALID_BASE64. Before this helper the raw stdlib decode error was
// wrapped in an unclassified fmt.Errorf and surfaced as 500 INTERNAL_ERROR,
// polluting server metrics and hiding the actionable message (TASK-095,
// ISS-089). (An empty value is not an error here: it decodes to zero bytes and
// flows on to the existing ErrPublicKeyRequired / ErrOwnerRequired checks,
// which are already classified as 400.)
func decodeKey(field, s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid %s encoding: %w", field, token.ErrInvalidBase64)
	}
	return b, nil
}
