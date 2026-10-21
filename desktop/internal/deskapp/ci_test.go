package deskapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from the test's working directory to the checkout root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "no checkout root above %s", dir)
		dir = parent
	}
}

// TestCIWiresTheDesktopModule is the guard for the gap this module lived in:
// `go test ./...` at the root never enters a nested module, so a broken
// window layer passed every gate silently. The fix is a `desktop` job in
// ci.yml; this test fails if that job disappears or stops running the same
// commands as `just desktop-check`, which is what a contributor runs locally.
func TestCIWiresTheDesktopModule(t *testing.T) {
	root := repoRoot(t)

	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	require.NoError(t, err)
	workflow := string(ci)

	require.Contains(t, workflow, "  desktop:", "ci.yml must define the desktop job")
	require.Contains(t, workflow, "go-version-file: desktop/go.mod",
		"the desktop job needs MyGo's Go version, not the core module's")
	require.Contains(t, workflow, "working-directory: desktop")

	for _, step := range []string{"gofmt -l .", "go build ./...", "go vet ./...", "go test -race ./..."} {
		assert.Contains(t, workflow, step,
			"the desktop job must run %q, like `just desktop-check`", step)
	}

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	local := string(recipe)

	require.Contains(t, local, "desktop-check:", "the local entry point must exist")
	for _, step := range []string{"gofmt -l .", "go build ./...", "go vet ./...", "go test -race ./..."} {
		assert.Contains(t, local, step,
			"`just desktop-check` must run %q, like the CI job", step)
	}
}

// TestRootGateCoversTheNestedModule pins the reason the job exists, so the
// explanation in ci.yml and the justfile does not become folklore: the root
// recipes really do skip desktop/, and it really is a separate module.
func TestRootGateCoversTheNestedModule(t *testing.T) {
	root := repoRoot(t)

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	assert.NotContains(t, string(recipe), "go test ./desktop/",
		"a root-relative path cannot cross into a nested module; use cd desktop")

	gomod, err := os.ReadFile(filepath.Join(root, "desktop", "go.mod"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(gomod), "module github.com/pplmx/aurora/desktop\n"),
		"desktop/ must stay its own module, which is why the root gate cannot see it")
}

// TestDesktopIsPackagedOnItsOwnPlatform pins why the CI job builds instead of
// packaging, and why `just desktop-package` is not run with a foreign
// -platform: the app's storage is cgo, so a cross build fails in runtime/cgo
// long before our own code. If storage ever becomes pure Go this test will
// say so, and the packaging matrix can start cross-building.
func TestDesktopIsPackagedOnItsOwnPlatform(t *testing.T) {
	root := repoRoot(t)

	recipe, err := os.ReadFile(filepath.Join(root, "justfile"))
	require.NoError(t, err)
	assert.Contains(t, string(recipe), "CGO_ENABLED=1 go tool mygo build",
		"MyGo defaults CGO_ENABLED to 0, which cannot build the cgo storage")

	// The reason: the core's database driver is cgo-only, and the desktop
	// module imports the core.
	for _, file := range []string{
		filepath.Join("internal", "infra", "sqlite", "lottery.go"),
		filepath.Join("internal", "domain", "blockchain", "init.go"),
	} {
		src, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err)
		assert.Contains(t, string(src), `"github.com/mattn/go-sqlite3"`,
			"%s is expected to be the cgo SQLite dependency that blocks cross builds", file)
	}
}
