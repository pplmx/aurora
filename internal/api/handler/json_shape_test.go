package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireSnakeCaseJSON rejects object keys that came from Go field names. The
// API answers in snake_case everywhere, so a handler that encodes a domain
// struct (GET /voting/session/{id} did, returning "ID"/"Title"/"StartTime")
// breaks every client parsing the documented shape. The scan walks nested
// arrays and objects, which is where list responses keep theirs.
func requireSnakeCaseJSON(t *testing.T, body []byte) {
	t.Helper()
	var doc any
	require.NoErrorf(t, json.Unmarshal(body, &doc), "response is not JSON: %s", bytes.TrimSpace(body))

	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch node := v.(type) {
		case map[string]any:
			for key, child := range node {
				require.NotEmptyf(t, key, "empty key at %s", path)
				require.NotEqualf(t, strings.ToUpper(key[:1]), key[:1],
					"key %q at %s looks like a Go field name; the API contract is snake_case", key, path)
				walk(child, path+"."+key)
			}
		case []any:
			for i, child := range node {
				walk(child, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walk(doc, "$")
}
