package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThePageEnvelopeDocumentsItsZeroLimit covers one field that means two
// things. The CLI's `-l 0` lists everything and echoes `limit: 0`; the gateway
// treats `limit=0` (or a value it cannot parse) as "use the default" and echoes
// `20`. A script that compares len(blocks) with limit to decide whether a page
// was truncated reads the CLI's 0 as "an empty page" — the opposite of what it
// means.
//
// The document has to say so, because the envelope is otherwise identical.
func TestThePageEnvelopeDocumentsItsZeroLimit(t *testing.T) {
	command := readRepoFile(t, "cmd", "aurora", "cmd", "blockchain.go")
	require.Contains(t, command, "printBlocksJSON(entries, total, limit, offset)",
		"the CLI must still echo the limit it used; update this check with the code")
	// The CLI's zero must still mean "no cap" rather than being rewritten.
	assert.NotContains(t, command, "limit == 0 {\n\t\t\tlimit = 20",
		"the CLI's zero means no cap; rewriting it would break the documented behaviour")

	handler := readRepoFile(t, "internal", "api", "handler", "page.go")
	rest := regexp.MustCompile(`parsed\s*>\s*0`)
	assert.True(t, rest.MatchString(handler),
		"the gateway must still treat a non-positive limit as unset; update this check with the code")

	doc := readRepoFile(t, "docs", "cli.md")
	for _, want := range []string{"`-l 0`", "不限", "默认 20"} {
		assert.Containsf(t, doc, want,
			"docs/cli.md must explain how a zero limit differs between the two surfaces")
	}
	assert.True(t, strings.Contains(doc, "limit == 0"),
		"docs/cli.md must tell a script how to handle the zero when comparing page sizes")
}

// TestOracleDataIsTheDocumentedLimitZeroException pins the one list command that
// does not read `-l 0` as "everything". Every other list defaults the flag to 0
// and lists the whole remainder; `oracle data` clamps to [1,100] and falls back
// to 10, because unbounded would scan the whole readings table. The reference
// has to say so, or a script that passes 0 for "all" silently gets ten rows.
func TestOracleDataIsTheDocumentedLimitZeroException(t *testing.T) {
	flag := readRepoFile(t, "cmd", "aurora", "cmd", "oracle.go")
	require.Contains(t, flag, `dataCmd.Flags().IntP("limit", "l", 10`,
		"oracle data must keep its non-zero default")

	clamp := readRepoFile(t, "cmd", "aurora", "cmd", "oracle_read.go")
	require.Contains(t, clamp, "func clampQueryLimit(")
	require.Contains(t, clamp, "return 10", "the clamp falls back to the default")

	doc := readRepoFile(t, "docs", "cli.md")
	assert.Contains(t, doc, "--limit 0` 与不传一样回落", "the reference must state the exception")
	assert.Contains(t, doc, "`-l 0` 仍表示列出全部", "and that the other lists keep the usual reading")
}
