package repocheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryPackageHasADocComment keeps the rule the tree now follows. Every
// package under internal/ and cmd/ opens with a sentence saying what it is for,
// which is what `go doc` shows and what a reader new to the repository reads
// first. Nineteen packages were missing one until they were filled in, and
// nothing stopped a new package from being added without one.
func TestEveryPackageHasADocComment(t *testing.T) {
	root := repoRoot(t)
	missing := []string{}
	checked := 0

	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return err
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			hasGo := false
			for _, entry := range entries {
				name := entry.Name()
				if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
					continue
				}
				hasGo = true
				body, err := os.ReadFile(filepath.Join(path, name))
				if err != nil {
					return err
				}
				text := strings.TrimLeft(string(body), "\n")
				if strings.HasPrefix(text, "// Package ") || strings.HasPrefix(text, "// Command ") {
					checked++
					return nil
				}
			}
			if hasGo {
				// Reported relative to the repository so the failure names
				// internal/api rather than a temporary path.
				missing = append(missing, strings.TrimPrefix(path, root+string(filepath.Separator)))
			}
			return nil
		})
		require.NoError(t, err)
	}

	assert.Emptyf(t, missing, "packages without a doc comment: %v", missing)
	require.Greater(t, checked, 25, "the scan must reach the packages")
}
