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
			// A directory can disappear mid-walk: this package's own tests run
			// alongside cmd/aurora/cmd, whose tests chdir into temp dirs and
			// call store.ResetForTest (which removes ./data). Treat a vanished
			// entry as "nothing here" rather than failing the scan, which
			// reported open .../cmd/aurora/cmd/data: no such file or directory.
			if walkErr != nil {
				return walkErrToleratingDisappearance(walkErr)
			}
			// Share the package's skip list so the build caches and runtime
			// directories (data/, logs/) are never entered. This walk predated
			// that helper and was the one place that reimplemented the rule.
			if entry.IsDir() {
				if skipWalkDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// This file names the markers in order to forbid them.
			if filepath.Base(path) == "todo_marker_test.go" {
				return nil
			}
			body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
			if readErr != nil {
				return walkErrToleratingDisappearance(readErr)
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
