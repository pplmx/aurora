package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTheOracleEnvelopeDifferenceIsDocumented covers the one place where the
// CLI's --json document and the REST response do not have the same outer shape.
// docs/cli.md claimed the fields are outright "consistent with REST"; a script
// written from that sentence and then pointed at /oracle/sources reads a bare
// array where the API sends {"sources": [...]}.
//
// The difference is deliberate — the oracle read endpoints wrap their payloads
// while other modules' lists do not — so the reference has to state it rather
// than let a reader discover it.
func TestTheOracleEnvelopeDifferenceIsDocumented(t *testing.T) {
	// The REST side: the oracle list really does send an envelope.
	dto := readRepoFile(t, "internal", "app", "oracleapp", "dto.go")
	require.Contains(t, dto, `json:"sources"`, "the oracle list response must still be an envelope")
	require.True(t, regexp.MustCompile(`type\s+ListSourcesResponse\s+struct`).MatchString(dto),
		"the envelope type must still exist")

	// The CLI side: `source list --json` really does print the array alone.
	command := readRepoFile(t, "cmd", "aurora", "cmd", "oracle_source.go")
	require.Contains(t, command, "printJSON(resp.Sources)",
		"the CLI must still print the bare array; update this check with the code")

	// And the reference has to say so.
	doc := readRepoFile(t, "docs", "cli.md")
	require.Contains(t, doc, "oracle source list",
		"docs/cli.md must name the command whose shape differs")
	require.Contains(t, doc, `{"sources":[...]}`,
		"docs/cli.md must show the envelope the REST endpoint sends")
	require.True(t, strings.Contains(doc, "字段一致不等于外层一致"),
		"docs/cli.md must distinguish field parity from envelope parity")
}
