package events

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/errdetail"
)

func base64DecodeField(payload []byte, field string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		// Only ErrInvalidPayload is the contract; the JSON detail is printed but
		// stays unwrappable so a caller cannot branch on the parser's internals.
		return nil, errdetail.Wrap(ErrInvalidPayload, err)
	}
	val, ok := m[field].(string)
	if !ok {
		return nil, fmt.Errorf("%w: field %q is not a string", ErrInvalidPayload, field)
	}
	return base64.StdEncoding.DecodeString(val)
}
