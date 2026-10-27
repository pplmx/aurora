package repocheck

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLayerDependencies pins the import direction that the architecture
// document promises. It reads production imports only: tests may open SQLite
// fixtures, but a shipped package must not depend on a lower implementation
// layer behind its ports.
func TestLayerDependencies(t *testing.T) {
	repo := repoRoot(t)

	rules := []struct {
		dir    string
		forbid []string
		why    string
	}{
		{
			dir:    "internal/domain",
			forbid: []string{"internal/infra", "internal/app", "internal/api"},
			why:    "domain must not reach into application, transport, storage or the ledger implementation",
		},
		{
			dir:    "internal/app",
			forbid: []string{"internal/infra"},
			why:    "use cases must use domain/ledger ports, not storage, HTTP or the ledger implementation",
		},
		{
			dir:    "internal/api",
			forbid: []string{"internal/infra"},
			why:    "transport handlers must use injected app/ledger ports",
		},
		{
			dir:    "internal/infra",
			forbid: []string{"internal/app", "internal/api"},
			why:    "infrastructure must not depend on its callers",
		},
	}

	for _, rule := range rules {
		rule := rule
		t.Run(rule.dir, func(t *testing.T) {
			base := filepath.Join(repo, rule.dir)
			err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					if path != base && entry.Name() == "container" && rule.dir == "internal/app" {
						return filepath.SkipDir
					}
					return nil
				}
				if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
					return nil
				}
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(repo, path)
				for _, imp := range file.Imports {
					pathValue, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						continue
					}
					for _, forbidden := range rule.forbid {
						if strings.Contains(pathValue, "/"+forbidden) {
							t.Errorf("%s imports %s; %s", rel, pathValue, rule.why)
						}
					}
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}
