package repocheck

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheApiReferenceExamplesParse covers the fenced JSON in docs/api.md. A
// reader copies one of these to start a request, and an example that does not
// parse teaches the wrong shape: a trailing comma or a stray comment costs a
// debugging round before the reader discovers the server is fine.
func TestTheApiReferenceExamplesParse(t *testing.T) {
	body := readRepoFile(t, "docs", "api.md")
	blocks := regexp.MustCompile("(?s)```json\n(.*?)```").FindAllStringSubmatch(body, -1)
	require.Greater(t, len(blocks), 8, "the reference must carry its examples")

	for i, block := range blocks {
		example := strings.TrimSpace(block[1])
		// A fenced block tagged json that holds nothing is worse than none: it
		// reads as an example the writer meant to fill in.
		require.NotEmptyf(t, example, "docs/api.md example %d is empty", i+1)
		var v any
		assert.NoErrorf(t, json.Unmarshal([]byte(example), &v),
			"docs/api.md example %d does not parse as JSON:\n%s", i+1, example)
	}
}
