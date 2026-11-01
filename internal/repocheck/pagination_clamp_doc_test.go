package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheReferenceSaysHowToDetectAClampedPage covers the consequence of a
// clamp that does not fail: a client asking for 500 rows gets 100 and no error.
//
// The envelope endpoints echo the limit they applied, so the comparison is
// possible; the bare-array endpoints cannot, and a client that assumes it
// received everything silently drops records. The reference has to say which
// shape is which.
func TestTheReferenceSaysHowToDetectAClampedPage(t *testing.T) {
	// The envelope endpoint must still echo the effective values.
	handler := readRepoFile(t, "internal", "api", "handler", "blockchain.go")
	require.Contains(t, handler, "Limit: limit, Offset: offset",
		"the page must echo the limit and offset it applied")

	// The clamp itself must not become an error: a rejected over-limit page
	// would be a different contract, and the paragraph below describes this one.
	page := readRepoFile(t, "internal", "api", "handler", "page.go")
	assert.Contains(t, page, "min(parsed, maxLimit)",
		"an over-limit request must be reduced, not refused")

	doc := readRepoFile(t, "docs", "api.md")
	require.Contains(t, doc, "截断不会报错",
		"docs/api.md must say the clamp is silent")
	for _, want := range []string{"回显生效的", "裸数组"} {
		assert.Containsf(t, doc, want,
			"docs/api.md must distinguish the shape that echoes the applied paging from the one that cannot")
	}
	// And it has to give the reader a way to act, not only the fact.
	assert.True(t, strings.Contains(doc, "返回条数会等于上限"),
		"docs/api.md must tell a client how to notice it was clamped")
}
