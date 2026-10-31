package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkipWalkDirCoversTheCaches pins the list every walk in this package shares.
// The cost of forgetting an entry is not a wrong answer, it is a walk that takes
// twelve seconds — .gocache alone holds 25 GB in a working checkout — and the
// cost of adding a project directory by mistake is worse: a check that stops
// seeing the files it exists to inspect.
func TestSkipWalkDirCoversTheCaches(t *testing.T) {
	for _, cache := range []string{
		".git", ".gocache", ".gomodcache", ".gopath", ".gotmp", ".rumdl_cache",
		"node_modules", "dist", ".screens", "logs", "data", ".planning", "testdata",
	} {
		assert.Truef(t, skipWalkDir(cache), "%q must be skipped by every walk", cache)
	}

	// The directories the checks do read must not be skipped, or they would pass
	// on an empty walk.
	for _, kept := range []string{"internal", "cmd", "docs", "desktop", "tools", "config"} {
		assert.Falsef(t, skipWalkDir(kept), "%q holds sources the checks read; it must be walked", kept)
	}

	// The list is a set of directory names, not a prefix match: a file called
	// dist.go must not be skipped by a rule meant for the dist directory.
	assert.False(t, skipWalkDir("dist.go"), "the rule matches directory names, not prefixes")
	assert.False(t, strings.HasPrefix("internal", "dist"), "sanity")
}

// TestWalkGoSourceReadsOnlyGoFiles checks the helper the package shares, so a
// change to it cannot quietly stop visiting files while every caller still
// passes.
func TestWalkGoSourceReadsOnlyGoFiles(t *testing.T) {
	seen := 0
	require.NoError(t, walkGoSource(repoRoot(t), func(path, body string) {
		seen++
		require.Truef(t, strings.HasSuffix(path, ".go"), "walkGoSource handed back %q", path)
		require.NotEmpty(t, body, "%q came back empty", path)
		require.Falsef(t, strings.Contains(path, "_test.go"),
			"walkGoSource must skip test files, got %q", path)
	}))
	require.Greater(t, seen, 100, "the walk must reach the tree")
}
