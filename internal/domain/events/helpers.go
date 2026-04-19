package events

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func base64DecodeField(payload []byte, field string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	val, ok := m[field].(string)
	if !ok {
		return nil, fmt.Errorf("%w: field %q is not a string", ErrInvalidPayload, field)
	}
	return base64.StdEncoding.DecodeString(val)
}
