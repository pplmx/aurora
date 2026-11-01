package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	blockedPrefixDecl = regexp.MustCompile(`MustParsePrefix\("([^"]+)"\)`)
	docPrefixCite     = regexp.MustCompile("`([0-9a-f.:]+/[0-9]+)`")
	docPrefixCount    = regexp.MustCompile(`完整的 (\d+) 条前缀清单`)
)

// TestBlockedAddressDocCitesRealPrefixes keeps the SSRF table in the API
// reference tied to the policy it describes. The section used to list prefixes
// inline in prose, which drifted: the prose named about half of them and none of
// the readers could tell which half. The table now groups them by reason and
// names one example per group, so the examples are what has to stay real.
func TestBlockedAddressDocCitesRealPrefixes(t *testing.T) {
	code := readRepoFile(t, "internal", "netguard", "blocked.go")
	declared := map[string]bool{}
	for _, m := range blockedPrefixDecl.FindAllStringSubmatch(code, -1) {
		declared[m[1]] = true
	}
	require.NotEmpty(t, declared, "blocked.go must declare its prefixes with MustParsePrefix")

	doc := readRepoFile(t, "docs", "api.md")
	start := strings.Index(doc, "抓取拒绝内部地址")
	require.GreaterOrEqual(t, start, 0, "docs/api.md must keep its blocked-address section")
	end := strings.Index(doc[start:], "重定向链最多跟随")
	require.Greater(t, end, 0, "the section must end before the redirect note")
	section := doc[start : start+end]

	cited := 0
	for _, m := range docPrefixCite.FindAllStringSubmatch(section, -1) {
		cited++
		assert.Truef(t, declared[m[1]],
			"docs/api.md names %s as a blocked prefix, but netguard does not list it", m[1])
	}
	require.GreaterOrEqual(t, cited, 10,
		"the table should name enough prefixes to be useful, found %d", cited)

	// The count in the prose is a fact about the code, so it is checked rather
	// than trusted: a new prefix must update the sentence, and the sentence is
	// what tells a reader where the authoritative list lives.
	m := docPrefixCount.FindStringSubmatch(section)
	require.NotNil(t, m, "the section must say how many prefixes the policy holds")
	assert.Equalf(t, len(declared), atoiOrZero(m[1]),
		"docs/api.md says %s blocked prefixes, netguard declares %d", m[1], len(declared))
}

// atoiOrZero parses a decimal string, returning 0 for anything else so a
// malformed count reads as a mismatch rather than as a parse error.
func atoiOrZero(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// TestFetchLimitsDocMatchesTheCode pins the two bounds a data source fetch
// runs under. Both are promises an operator relies on: the redirect ceiling
// caps how far a hostile feed can bounce a request, and the response ceiling
// caps how much of it is buffered. The response one was undocumented entirely —
// the section mentioned neither, so a feed that started serving 20 MiB failed
// with no documented reason.
func TestFetchLimitsDocMatchesTheCode(t *testing.T) {
	code := readRepoFile(t, "internal", "infra", "http", "fetcher.go")

	redirects := readConst(t, code, "maxRedirects")
	require.Positive(t, redirects, "a redirect ceiling is required")
	responseBytes := readConst(t, code, "maxResponseBytes")
	require.Positive(t, responseBytes, "a response ceiling is required")

	doc := strings.Join(strings.Fields(readRepoFile(t, "docs", "api.md")), " ")

	assert.Containsf(t, doc, "重定向链最多跟随 "+itoa(redirects)+" 跳",
		"the reference must state the %d-hop redirect ceiling", redirects)

	// The cap is 10 MiB; the document states it in MiB like the request-body
	// limit does, so the two sizes read in one unit.
	mib := responseBytes / (1 << 20)
	require.Equal(t, 10<<20, responseBytes, "the response ceiling is 10 MiB")
	assert.Containsf(t, doc, itoa(mib)+" MiB",
		"the reference must state the %d MiB response ceiling", mib)
	assert.Contains(t, doc, "502 FETCH_FAILED",
		"an oversized response is an upstream failure, and the reference must say so")
}

// itoa formats a non-negative int without importing strconv into this file's
// existing imports.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 8)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
