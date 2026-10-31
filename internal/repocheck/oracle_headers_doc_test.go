package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerSetDecl matches one entry of a rejected-header set in the domain
// package: `"content-length": {},` inside a map literal.
var headerSetDecl = regexp.MustCompile(`"([a-z0-9-]+)":\s*\{\},`)

// TestBlockedHeaderDocListsEveryHeader keeps the API reference's table tied to
// the two sets it describes. The prose used to name six of the seven hop-by-hop
// headers, dropping `Proxy-Connection`, and a reader has no way to notice a
// missing row in a sentence.
func TestBlockedHeaderDocListsEveryHeader(t *testing.T) {
	code := readRepoFile(t, "internal", "domain", "oracle", "service.go")

	// The two map literals are the only places in this file that use this
	// shape, so slicing between them separates transport-managed from
	// hop-by-hop without parsing Go.
	transportAt := strings.Index(code, "transportManagedSourceHeaders")
	require.GreaterOrEqual(t, transportAt, 0, "service.go must declare the transport-managed set")
	hopAt := strings.Index(code, "hopByHopSourceHeaders = map")
	require.Greater(t, hopAt, transportAt, "the hop-by-hop set must follow the transport-managed one")

	transport := headerSetDecl.FindAllStringSubmatch(code[transportAt:hopAt], -1)
	hopByHop := headerSetDecl.FindAllStringSubmatch(code[hopAt:], -1)
	require.NotEmpty(t, transport, "the transport-managed set must list headers")
	require.NotEmpty(t, hopByHop, "the hop-by-hop set must list headers")

	doc := readRepoFile(t, "docs", "api.md")
	start := strings.Index(doc, "有的头不能由数据源设置")
	require.GreaterOrEqual(t, start, 0, "docs/api.md must keep its blocked-header section")
	end := strings.Index(doc[start:], "数据源 URL 必须带非空 hostname")
	require.Greater(t, end, 0, "the section must end before the URL rules")
	section := doc[start : start+end]

	// The table writes names in their canonical casing, so the comparison is
	// case-insensitive.
	lowerSection := strings.ToLower(section)
	for _, m := range append(transport, hopByHop...) {
		assert.Containsf(t, lowerSection, m[1],
			"docs/api.md does not list %q, which the domain rejects", m[1])
	}
}
