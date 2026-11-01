package repocheck

import (
	"io/fs"
	"os"
	"path/filepath"
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

// TestWalkErrToleratingDisappearance covers the tolerance the walks share. A
// directory removed between WalkDir's readdir and its descent used to fail the
// whole scan: the suite runs the repocheck package alongside cmd/aurora/cmd,
// whose tests chdir into temp dirs and call store.ResetForTest, which removes
// ./data. The failure surfaced as "open .../cmd/aurora/cmd/data: no such file
// or directory" — a repository directory that never existed.
func TestWalkErrToleratingDisappearance(t *testing.T) {
	// Vanishing is not an error: there is simply nothing left to read.
	assert.NoError(t, walkErrToleratingDisappearance(fs.ErrNotExist),
		"an entry that vanished mid-walk must not fail the scan")
	assert.NoError(t, walkErrToleratingDisappearance(
		&fs.PathError{Op: "open", Path: "/tmp/gone", Err: fs.ErrNotExist}))

	// Everything else is still a real failure. A permission error in particular
	// must not be swallowed: it would make a check pass on the files it could
	// not open.
	assert.ErrorIs(t, walkErrToleratingDisappearance(fs.ErrPermission), fs.ErrPermission)
	assert.False(t, walkErrToleratingDisappearance(nil) != nil, "nil stays nil")
}

// TestWalkGoSourceSurvivesAFileThatVanished proves the tolerance is wired into
// the walk the checks actually call, not only into the helper.
//
// A dangling symlink is the deterministic version of the race: WalkDir reports
// a normal directory entry (it does not follow the link), and the file read then
// fails with ENOENT, exactly as a file deleted between readdir and open does.
// Producing this by removing files from inside the walk does not work —
// WalkDir re-reads each directory as it descends, so the deletion is simply
// invisible and the test passes against the unfixed code.
func TestWalkGoSourceSurvivesAFileThatVanished(t *testing.T) {
	root := t.TempDir()
	readable := filepath.Join(root, "readable.go")
	require.NoError(t, os.WriteFile(readable, []byte("package probe\n"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(root, "gone.go"), filepath.Join(root, "vanished.go")),
		"this test needs symlinks; skip on platforms without them")

	var walked []string
	err := walkGoSource(root, func(path, _ string) { walked = append(walked, path) })
	require.NoError(t, err, "a file that vanished before its read must not fail the scan")
	require.Equal(t, []string{readable}, walked, "the readable file is still visited")
}
