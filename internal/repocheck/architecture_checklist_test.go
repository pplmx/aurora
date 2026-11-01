package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkedRemoval matches a completed checklist item that says a set of
// package-qualified symbols was deleted: "把 `pkg.{A,B,C}` 的调用点改到 ... 并删除转发"
// and the simpler "`pkg.A` 已删除". It captures the package and the brace list.
//
// Only removals are machine-checkable. "Renamed X", "moved Y" and "evaluated
// Z" depend on judgement, and an automated reading of those would either fail
// on a legitimate rename or pass on a stale claim.
var checkedRemoval = regexp.MustCompile(
	"`([a-z][a-z0-9]*(?:/[a-z0-9]+)*)\\.\\{([A-Za-z][A-Za-z0-9_, ]*)\\}`[^\\n]*删除")

// TestCheckedRemovalsAreStillGone keeps the architecture document's completed
// items honest. The document already contradicted itself once: architecture.md
// marked the DBPath/InitDB/OpenDB removal done while ablation.md still listed
// those symbols as open work. A checklist that records deletions is checkable,
// so it should be checked rather than trusted.
func TestCheckedRemovalsAreStillGone(t *testing.T) {
	repo := repoRoot(t)
	text := readRepoFile(t, "docs", "architecture.md")

	checked := 0
	for _, match := range checkedRemoval.FindAllStringSubmatch(text, -1) {
		pkgPath, names := match[1], match[2]
		symbols := splitTrimmed(names, ",")
		require.NotEmptyf(t, symbols, "the removal in %q names no symbol", match[0])
		checked++

		dir := filepath.Join(repo, "internal", pkgPath)
		if _, err := os.Stat(dir); err != nil {
			// A package path in the document may be a logical name ("blockchain"
			// for internal/infra/blockchain). Fall back to a name search.
			dir = findPackageDir(t, repo, pkgPath)
		}
		require.NotEmptyf(t, dir, "the document names package %q, which is not in the checkout", pkgPath)

		declared := exportedNames(t, dir)
		for _, symbol := range symbols {
			assert.Falsef(t, declared[symbol],
				"docs/architecture.md marks the removal of %s.%s as done, but %s still declares it",
				pkgPath, symbol, dir)
		}
	}
	require.GreaterOrEqual(t, checked, 1,
		"the checklist audit must find at least one machine-checkable removal")
}

// findPackageDir locates a package by its last path element under internal/.
func findPackageDir(t *testing.T, repo, name string) string {
	t.Helper()
	var found string
	err := filepath.WalkDir(filepath.Join(repo, "internal"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return walkErrToleratingDisappearance(err)
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	require.NoError(t, err)
	return found
}

// exportedNames lists the package-level names dir declares. A symbol that moved
// out of the package is absent here, which is exactly what a removal means.
func exportedNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	fset := token.NewFileSet()
	names := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					names[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						names[s.Name.Name] = true
					case *ast.ValueSpec:
						for _, ident := range s.Names {
							names[ident.Name] = true
						}
					}
				}
			}
		}
	}
	return names
}

// splitTrimmed splits on sep and trims each part, dropping empty ones.
func splitTrimmed(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
