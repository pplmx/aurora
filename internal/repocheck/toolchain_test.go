package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot resolves the checkout root from this file's location, so the tests
// work whatever directory `go test` runs them from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// readRepoFile resolves a checkout file relative to the repository root.
func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{repoRoot(t)}, parts...)...))
	require.NoError(t, err)
	return string(data)
}

// justRecipe returns one top-level recipe body from the justfile.
func justRecipe(t *testing.T, source, name string) string {
	t.Helper()
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, name+":") {
			continue
		}
		var body []string
		for _, next := range lines[i+1:] {
			if next == "" || (next[0] != ' ' && next[0] != '\t') {
				break
			}
			body = append(body, next)
		}
		require.NotEmpty(t, body, "just recipe %q has no commands", name)
		return strings.Join(body, "\n")
	}
	require.FailNow(t, "just recipe not found", name)
	return ""
}

// TestGoModVersionIsPinnedEverywhere guards the three-way contract between
// go.mod, the CI runners and the release image. These drifted apart before:
// go.mod said 1.26 while CI and Docker installed different versions, and the
// Docker build failed for reasons nobody could reproduce locally. A plain
// `go test ./...` now catches any one of them moving alone.
func TestGoModVersionIsPinnedEverywhere(t *testing.T) {
	goMod := readRepoFile(t, "go.mod")
	m := regexp.MustCompile(`(?m)^go (\d+\.\d+(\.\d+)?)$`).FindStringSubmatch(goMod)
	require.NotNil(t, m, "go.mod must declare a pinned go directive")
	goVersion := m[1]
	majorMinor := strings.Join(strings.Split(goVersion, ".")[:2], ".")
	assert.Equal(t, "1.27.2", goVersion,
		"this assertion pins the release toolchain; update it together with go.mod")

	for _, name := range []string{"ci.yml", "build.yml", "release.yml"} {
		workflow := readRepoFile(t, ".github", "workflows", name)
		assert.Contains(t, workflow, "GO_VERSION: '"+goVersion+"'",
			"%s must install the go.mod toolchain", name)
	}

	dockerfile := readRepoFile(t, "Dockerfile")
	assert.Contains(t, dockerfile, "FROM golang:"+majorMinor+"-alpine",
		"the Docker builder must track go.mod's Go major.minor")
}

// TestDockerBuildStaysPureGo pins the pure-Go image contract: no C
// toolchain, no external link mode, and a distroless/static runtime.
func TestDockerBuildStaysPureGo(t *testing.T) {
	dockerfile := readRepoFile(t, "Dockerfile")
	assert.Contains(t, dockerfile, "ENV CGO_ENABLED=0")
	assert.NotContains(t, dockerfile, "CGO_ENABLED=1")
	assert.NotContains(t, dockerfile, "musl-dev")
	assert.NotContains(t, dockerfile, "-linkmode external")
}

// TestSQLiteStorageStaysPureGo prevents the SQLite driver from drifting back
// to a cgo-only implementation. The root module owns the storage dependency;
// the nested desktop module must resolve the same pure-Go driver through its
// replace directive.
func TestSQLiteStorageStaysPureGo(t *testing.T) {
	goMod := readRepoFile(t, "go.mod")
	assert.Contains(t, goMod, "modernc.org/sqlite v1.61.0")
	assert.NotContains(t, goMod, "github.com/mattn/go-sqlite3")

	desktopGoMod := readRepoFile(t, "desktop", "go.mod")
	assert.Contains(t, desktopGoMod, "modernc.org/sqlite v1.61.0 // indirect")
	assert.NotContains(t, desktopGoMod, "github.com/mattn/go-sqlite3")

	assertGoSourcesUsePureSQLite(t, ".")

	justfile := readRepoFile(t, "justfile")
	assert.Contains(t, justfile, "CGO_ENABLED=0 go build")
	assert.Contains(t, justfile, "CGO_ENABLED=0 go vet")
	assert.Contains(t, justfile, "CGO_ENABLED=0 go test")
}

func assertGoSourcesUsePureSQLite(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "dist" || entry.Name() == "vendor" ||
				strings.HasPrefix(entry.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		rel, err := filepath.Rel(root, path)
		require.NoError(t, err)
		for _, imported := range parsed.Imports {
			if imported.Path.Value == `"github.com/mattn/go-sqlite3"` {
				t.Errorf("%s must not import the cgo-only SQLite driver", rel)
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Open" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "sql" {
				return true
			}
			driver, ok := call.Args[0].(*ast.BasicLit)
			if ok && driver.Kind == token.STRING && driver.Value == `"sqlite3"` {
				t.Errorf("%s must open SQLite through the pure-Go driver", rel)
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
}

// TestDesktopCheckStaysCgoFree separates the mandatory sandbox gate from the
// optional race detector. A C compiler must never be required merely to build,
// vet, and test the pure-Go desktop module.
func TestDesktopCheckStaysCgoFree(t *testing.T) {
	justfile := readRepoFile(t, "justfile")

	check := justRecipe(t, justfile, "desktop-check")
	assert.Contains(t, check, "CGO_ENABLED=0 go test -count=1 ./...")
	assert.NotContains(t, check, "-race",
		"desktop-check must remain runnable where the race detector's cgo runtime is unavailable")

	test := justRecipe(t, justfile, "desktop-test")
	assert.Contains(t, test, "CGO_ENABLED=0 go test -count=1 ./...")

	race := justRecipe(t, justfile, "desktop-race")
	assert.Contains(t, race, "CGO_ENABLED=1 go test -race -count=1 ./...",
		"the cgo requirement of the race detector must be explicit")

	root := justRecipe(t, justfile, "race")
	assert.Contains(t, root, "CGO_ENABLED=1 go test -race -count=1 ./...",
		"the root race recipe must enable cgo explicitly")
}

// TestGolangciLintVersionIsPinned pins the workflow's golangci-lint binary.
// `version: latest` jumped to the v2 line and rejected the config schema the
// repo still used, which is how lint started failing with "additional
// properties 'exclude-rules' not allowed" without any local change.
func TestGolangciLintVersionIsPinned(t *testing.T) {
	ci := readRepoFile(t, ".github", "workflows", "ci.yml")
	m := regexp.MustCompile(`version: (v\S+)`).FindStringSubmatch(ci)
	require.NotNil(t, m, "ci.yml must pin the golangci-lint version, not use latest")
	assert.NotEqual(t, "latest", m[1])
}

// TestDomainLayerHasNoOutboundDependencies enforces the layering the README
// describes. A domain package that imports app, api or infra turns the domain
// into a consumer of the very layers meant to depend on it, which is how the
// token service ended up importing infra/events for its EventBus parameter.
//
// The rule is checked by reading imports, not by convention: a new violating
// import fails here.
func TestDomainLayerHasNoOutboundDependencies(t *testing.T) {
	forbidden := []string{
		"github.com/pplmx/aurora/internal/app",
		"github.com/pplmx/aurora/internal/api",
		"github.com/pplmx/aurora/internal/infra",
	}
	err := filepath.WalkDir(filepath.Join(repoRoot(t), "internal", "domain"),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			require.NoError(t, perr)
			for _, imp := range f.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				for _, bad := range forbidden {
					assert.False(t, strings.HasPrefix(p, bad),
						"%s imports %s: the domain layer must not depend on %s",
						filepath.Base(filepath.Dir(path)), p, bad)
				}
			}
			return nil
		})
	require.NoError(t, err)
}

// TestRecordsPackageStaysPure keeps the shared payload decoder a leaf: the CLI,
// the REST handler and the window all import it, so a dependency on app, api or
// infra would drag one representation layer into the others.
func TestRecordsPackageStaysPure(t *testing.T) {
	err := filepath.WalkDir(filepath.Join(repoRoot(t), "internal", "records"),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			require.NoError(t, perr)
			for _, imp := range f.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				assert.Falsef(t, strings.HasPrefix(p, "github.com/pplmx/aurora/"),
					"%s imports %s: internal/records must stay a leaf package",
					filepath.Base(path), p)
			}
			return nil
		})
	require.NoError(t, err)
}

// TestCIRunsTheDesktopLint: the root lint job runs `golangci-lint run ./...`,
// which never descends into the nested desktop module. Without its own step in
// the desktop job, staticcheck/unused coverage for the window layer silently
// disappears again.
func TestCIRunsTheDesktopLint(t *testing.T) {
	workflow := readRepoFile(t, ".github", "workflows", "ci.yml")
	jobStart := strings.Index(workflow, "  desktop:")
	require.GreaterOrEqual(t, jobStart, 0, "ci.yml must have a desktop job")

	desktopJob := workflow[jobStart:]
	if next := strings.Index(desktopJob[1:], "\n  desktop-package:"); next >= 0 {
		desktopJob = desktopJob[:next+1]
	}
	require.Contains(t, desktopJob, "golangci/golangci-lint-action@",
		"the desktop job must lint the nested module")
	require.Contains(t, desktopJob, "working-directory: desktop",
		"the lint step must run inside desktop/, where the module boundary is")
}

// TestArchitectureTreeListsEveryPackage keeps the target-structure sketch true:
// a package that ships without a line in the tree is invisible to a reader
// orienting themselves, and this list has drifted before (internal/records was
// added to the dependency table but not to the tree).
func TestArchitectureTreeListsEveryPackage(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "internal"))
	require.NoError(t, err)
	tree := readRepoFile(t, "docs", "architecture.md")
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		checked++
		assert.Truef(t,
			strings.Contains(tree, "├── "+name+"/") || strings.Contains(tree, "└── "+name+"/"),
			"docs/architecture.md must list internal/%s in the target tree", name)
	}
	require.GreaterOrEqual(t, checked, 10, "the tree check must see the package set")
}
