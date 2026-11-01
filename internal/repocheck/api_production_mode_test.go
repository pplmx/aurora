package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTheReferenceNamesTheProductionSwitch covers the one variable that decides
// whether the API refuses an empty or weak key. The reference said "in
// production the key must be at least 32 characters" without ever saying how a
// process enters that mode, so an operator reading it could not tell why their
// container accepted a missing key — nothing sets AURORA_ENV for them.
func TestTheReferenceNamesTheProductionSwitch(t *testing.T) {
	config := readRepoFile(t, "internal", "config", "config.go")
	doc := readRepoFile(t, "docs", "api.md")

	// The switch itself must still be this variable: the check below pins the
	// document to the code, and a rename has to move both.
	require.Contains(t, config, `os.Getenv("AURORA_ENV")`,
		"the production switch must still read AURORA_ENV; update this check with the code")
	require.Contains(t, doc, "AURORA_ENV=production",
		"docs/api.md must say which variable turns on production mode")
	require.Contains(t, doc, "开发模式",
		"docs/api.md must say what happens when the variable is absent")

	// The key requirements the section states have to stay true of the code.
	require.Contains(t, config, "minProductionAPIKeyLength",
		"the 32-character minimum must still be enforced somewhere")
	require.Contains(t, doc, "32 个不含空白的可见 ASCII")

	// A reader must be able to tell that no other channel exists. Naming the
	// config key and the flag as absent is the sentence that keeps someone from
	// hunting for them.
	require.True(t, strings.Contains(doc, "没有对应的配置键或命令行开关"),
		"docs/api.md must say production mode has no config key or flag")
}
