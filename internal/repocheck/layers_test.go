package repocheck

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			// checked counts the files the rule actually examined. A rule whose
			// directory was renamed, or emptied, would otherwise pass by walking
			// nothing — the failure mode that makes a layering check decorative.
			checked := 0
			err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return walkErrToleratingDisappearance(err)
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
				checked++
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
			require.Greaterf(t, checked, 15,
				"%s: the rule must examine the layer's files, not pass on an empty walk", rule.dir)
		})
	}
}

// TestTheDesktopModuleKeepsItsLayers covers the nested module, which the rules
// above never walked: they list internal/* only. The window is a second
// presentation layer, so it has its own direction — nativeui draws and holds
// view state, deskapp adapts it to the app layer, and neither may open storage.
//
// Production imports only, as above: the deskapp tests build a real SQLite store
// from infra to exercise the adapters, which is the same allowance the root
// module's tests have.
func TestTheDesktopModuleKeepsItsLayers(t *testing.T) {
	repo := repoRoot(t)
	rules := []struct {
		dir    string
		forbid []string
		why    string
	}{
		{
			dir:    "desktop/internal/nativeui",
			forbid: []string{"internal/app", "internal/infra", "internal/api"},
			why:    "the views hold state and draw; they reach the store through the services interface",
		},
		{
			dir:    "desktop/internal/deskapp",
			forbid: []string{"internal/infra"},
			why:    "the adapter uses the container's ports, not the storage implementation",
		},
	}

	for _, rule := range rules {
		rule := rule
		t.Run(rule.dir, func(t *testing.T) {
			base := filepath.Join(repo, rule.dir)
			checked := 0
			err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return walkErrToleratingDisappearance(err)
				}
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
					strings.HasSuffix(entry.Name(), "_test.go") {
					return nil
				}
				checked++
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
			require.Greaterf(t, checked, 5,
				"%s: the rule must examine the package's files, not pass on an empty walk", rule.dir)
		})
	}
}

// TestApiDoesNotKnowAboutTheWindow covers the one-way rule between the two
// presentation layers. The desktop starts an in-process api.Server to reuse the
// REST handlers, so deskapp imports internal/api; the reverse would make the
// gateway depend on a window that may not exist, and a cycle would follow.
//
// The rule is stated in docs/architecture.md, which said "must not depend on
// each other" until the desktop's use of api made that literally false. What
// matters is the direction, so this asserts the direction.
func TestApiDoesNotKnowAboutTheWindow(t *testing.T) {
	repo := repoRoot(t)
	checked := 0
	err := filepath.WalkDir(filepath.Join(repo, "internal", "api"),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return walkErrToleratingDisappearance(err)
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
				strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			checked++
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
				if strings.Contains(pathValue, "/desktop") {
					t.Errorf("%s imports %s; the gateway must not depend on the window", rel, pathValue)
				}
			}
			return nil
		})
	require.NoError(t, err)
	require.Greater(t, checked, 10, "the walk must reach the handler files")

	// The other direction has to stay: the window's adapter is what reuses the
	// REST surface, and a refactor that dropped it would change the design.
	deskapp := readRepoFile(t, "desktop", "internal", "deskapp", "deskapp.go")
	assert.Contains(t, deskapp, `"github.com/pplmx/aurora/internal/api"`,
		"the desktop reuses the in-process REST server; update this check with the design")
}

// TestTheContainerIsTheOnlyAssemblyPoint covers the word "only" in the
// architecture document: the composition root is the one place that imports the
// storage and ledger implementations. The rule above skips that directory, which
// is what makes the exception work — and what would also let a second assem­bly
// point appear unnoticed, since nothing then checks where infra may be imported.
//
// So this walks the same trees without skipping it, and counts the packages that
// reach infra: exactly one may, and it must be the container.
func TestTheContainerIsTheOnlyAssemblyPoint(t *testing.T) {
	repo := repoRoot(t)
	importers := map[string][]string{} // package dir -> infra imports

	for _, layer := range []string{"internal/app", "internal/api"} {
		err := filepath.WalkDir(filepath.Join(repo, layer),
			func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return walkErrToleratingDisappearance(err)
				}
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
					strings.HasSuffix(entry.Name(), "_test.go") {
					return nil
				}
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				dir := filepath.Dir(path)
				for _, imp := range file.Imports {
					pathValue, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						continue
					}
					if strings.Contains(pathValue, "/internal/infra") {
						importers[dir] = append(importers[dir], pathValue)
					}
				}
				return nil
			})
		require.NoError(t, err)
	}

	container := filepath.Join(repo, "internal", "app", "container")
	for dir, imports := range importers {
		if dir == container {
			continue
		}
		rel, _ := filepath.Rel(repo, dir)
		t.Errorf("%s imports %s; the container is the only assembly point",
			rel, strings.Join(imports, ", "))
	}
	require.Contains(t, importers, container,
		"the container must still be the package that wires infra in; update this check with the design")
}
