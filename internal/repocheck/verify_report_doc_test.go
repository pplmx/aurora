package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheVerifyReportAlwaysCarriesItsIndex covers the one JSON field whose
// absence changes its meaning. `first_broken_index` is -1 for an intact chain
// and the index of the break otherwise, so 0 is a real answer — the genesis
// block. It carried `omitempty`, and a break at index 0 serialized without the
// field at all, leaving a caller to decode the zero value it also uses for
// "intact".
//
// The tag is checked in the source because the difference only shows up in one
// of the three cases, and a table-driven test of the report would look identical
// for the two that already worked.
func TestTheVerifyReportAlwaysCarriesItsIndex(t *testing.T) {
	types := readRepoFile(t, "internal", "ledger", "types.go")

	require.Contains(t, types, "FirstBrokenIndex",
		"the report must still carry the index")
	for _, line := range strings.Split(types, "\n") {
		if !strings.Contains(line, "FirstBrokenIndex") || !strings.Contains(line, "json:") {
			continue
		}
		assert.Containsf(t, line, `json:"first_broken_index"`,
			"the index must not be omitted: %s", strings.TrimSpace(line))
		assert.NotContainsf(t, line, "omitempty",
			"omitempty drops index 0, which is a real answer (a broken genesis block)")
	}

	doc := readRepoFile(t, "docs", "api.md")
	assert.Contains(t, doc, "first_broken_index",
		"docs/api.md must document the field")
	assert.True(t, strings.Contains(doc, "总是存在") || strings.Contains(doc, "always present"),
		"docs/api.md must say the field is always present, since its zero value is meaningful")
}
