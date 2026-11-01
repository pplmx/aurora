package repocheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoUnfinishedMarkersShip keeps a placeholder from reaching a release. The
// tree carries none today, and the point of this test is that it keeps carrying
// none: a TODO in shipped code is a promise nobody tracked, and a FIXME is a
// known defect left in place.
//
// A line may keep the word when it says the state is intended, so a deliberate
// exception stays documented where a reader finds it.
func TestNoUnfinishedMarkersShip(t *testing.T) {
	root := repoRoot(t)
	marker := regexp.MustCompile(`\b(TODO|FIXME|XXX|HACK)\b`)
	allowed := regexp.MustCompile(`(?i)(deliberately|intentionally|permanently|not a TODO)`)

	scanned, offenders := 0, []string{}
	for _, layer := range []string{"internal", "cmd", "desktop/internal"} {
		base := filepath.Join(root, layer)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		require.NoError(t, filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			// This file names the markers in order to forbid them.
			if filepath.Base(path) == "todo_marker_test.go" {
				return nil
			}
			body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
			if readErr != nil {
				return readErr
			}
			scanned++
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			for i, line := range strings.Split(string(body), "\n") {
				if !marker.MatchString(line) || allowed.MatchString(line) {
					continue
				}
				offenders = append(offenders,
					filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+" "+strings.TrimSpace(line))
			}
			return nil
		}))
	}
	require.Greater(t, scanned, 200, "the scan must reach the tree")
	assert.Emptyf(t, offenders, "unfinished markers in shipped code: %v", offenders)
}
