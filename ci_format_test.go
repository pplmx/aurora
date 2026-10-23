package aurora

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCoreModuleIsGofmtClean is the gate commit 7a7f53d slipped past. That
// commit landed three unformatted files (a misaligned error block and two
// struct literals) because no CI job formatted the core module: only the
// nested desktop job ran gofmt. Running the check here means a plain
// `go test ./...` catches it, whether or not the workflow gate below exists.
//
// The package list comes from `go list ./...`, which stops at the module
// boundary, so desktop/ (its own Go 1.27 module) is intentionally not part of
// this check; `just desktop-check` covers it.
func TestCoreModuleIsGofmtClean(t *testing.T) {
	dirs := goListDirs(t)
	require.NotEmpty(t, dirs, "go list must report this module's packages")

	files, err := gofmtTargets(dirs)
	require.NoError(t, err)
	args := append([]string{"-s", "-l"}, files...)
	out, err := exec.Command("gofmt", args...).CombinedOutput()
	require.NoErrorf(t, err, "gofmt failed:\n%s", out)

	unformatted := strings.Fields(string(out))
	require.Empty(t, unformatted,
		"run `gofmt -s -w` on these files; CI's Format step rejects them")
}

// TestGofmtTargetsOnlyGoFiles keeps the format gate independent of transient
// runtime data. Recursing through package directories made the gate race with
// the e2e package's cleanup of e2e/data; passing explicit Go files keeps the
// check deterministic without narrowing which source files are covered.
func TestGofmtTargetsOnlyGoFiles(t *testing.T) {
	files, err := gofmtTargets(goListDirs(t))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		require.True(t, strings.HasSuffix(file, ".go"), "gofmt target must be a Go source file: %s", file)
		info, err := os.Stat(file)
		require.NoError(t, err, "gofmt target must exist: %s", file)
		require.False(t, info.IsDir(), "gofmt target must be a file: %s", file)
	}
}

// TestCIRunsTheFormatGate pins the workflow half of the same rule. Without it
// the check above still runs locally, but a contributor whose editor formats
// on save would not notice that CI had no format step at all.
func TestCIRunsTheFormatGate(t *testing.T) {
	// This package lives at the checkout root, which is also where the
	// workflow file is, so the relative path is stable for `go test`.
	data, err := os.ReadFile(".github/workflows/ci.yml")
	require.NoError(t, err)
	workflow := string(data)

	require.Contains(t, workflow, "name: Format",
		"ci.yml must run a format gate before vet and test")
	require.Contains(t, workflow, `gofmt -s -l $dirs`,
		"the gate must check this module's packages with the justfile's -s flag")
	require.Contains(t, workflow, `dirs="$(go list -f '{{.Dir}}' ./...)"`,
		"the gate must enumerate this module's packages, not cross into desktop/")

	// Every module must be covered by exactly one gate: the core job checks
	// this module's packages, the desktop job checks desktop/.
	require.Contains(t, workflow, "  desktop:",
		"the nested module needs its own job, which also runs gofmt")
	require.Contains(t, workflow, `test -z "$(gofmt -l .)"`,
		"the desktop job must keep formatting desktop/")
}

func goListDirs(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", "./...").Output()
	require.NoError(t, err)
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			dirs = append(dirs, line)
		}
	}
	return dirs
}

func gofmtTargets(dirs []string) ([]string, error) {
	var files []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read package directory %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files, nil
}
