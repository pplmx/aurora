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

// TestTheBlockPageExampleIsSelfConsistent covers the arithmetic of one example.
// A page document carries its rows plus total, limit and offset, and those four
// have to agree: a reader who sees two rows implied by `total: 2` beside a
// single row in `blocks` with `limit: 20` learns a shape the server never sends.
// The parse test above accepted it because the JSON was well formed.
func TestTheBlockPageExampleIsSelfConsistent(t *testing.T) {
	body := readRepoFile(t, "docs", "api.md")
	blocks := regexp.MustCompile("(?s)```json\n(.*?)```").FindAllStringSubmatch(body, -1)

	found := 0
	for _, block := range blocks {
		var page struct {
			Blocks []json.RawMessage `json:"blocks"`
			Total  *int              `json:"total"`
			Limit  *int              `json:"limit"`
			Offset *int              `json:"offset"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(block[1])), &page); err != nil {
			continue
		}
		if page.Total == nil || page.Limit == nil || page.Offset == nil {
			continue // not a page document
		}
		found++
		// The rows the page holds are min(limit, total-offset) for a limit that
		// is not "all". A limit of 0 means the whole remainder.
		want := *page.Total - *page.Offset
		if want < 0 {
			want = 0
		}
		if *page.Limit > 0 && want > *page.Limit {
			want = *page.Limit
		}
		assert.Equalf(t, want, len(page.Blocks),
			"a block page with total=%d limit=%d offset=%d holds %d rows, not %d",
			*page.Total, *page.Limit, *page.Offset, want, len(page.Blocks))
	}
	require.GreaterOrEqual(t, found, 1, "the reference must show a block page")
}
