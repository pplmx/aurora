package repocheck

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadmeStructureListsEveryInternalPackage keeps the README's project tree
// usable as a map. It is the first place a new contributor looks for "where
// does this live", and it had drifted by six packages: records, background,
// metrics, netguard, config, logger and tools were all shipped but absent, so
// the tree implied a smaller repository than the one in front of a reader.
// The architecture document has the same check (TestArchitectureTreeListsEveryPackage);
// the README needs its own because it is read first and updated separately.
func TestReadmeStructureListsEveryInternalPackage(t *testing.T) {
	readme := readRepoFile(t, "README.md")
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "internal"))
	require.NoError(t, err)
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		checked++
		assert.Truef(t,
			strings.Contains(readme, "├── "+name+"/") || strings.Contains(readme, "└── "+name+"/"),
			"README.md must list internal/%s in its project tree", name)
	}
	require.GreaterOrEqual(t, checked, 10, "the tree check must see the package set")
}

// TestArchitectureDependencyTableMatchesImports keeps the "current dependency
// facts" table honest. It claims to be checkable against `go list`, and the
// claim had already drifted: domain's row said standard library and same-layer
// only while domain/oracle imported netguard, and nativeui's row listed one
// dependency while the package also imports the lottery domain for its
// pre-write validation. A reader uses this table to decide where new code
// belongs, so a stale row misdirects the next design decision rather than
// merely mislabelling an old one.
//
// The test reads the table's first two columns and compares them to the
// project packages each row's directory actually imports. Third-party and
// standard-library imports are out of scope: the table describes Aurora's own
// layering.
func TestArchitectureDependencyTableMatchesImports(t *testing.T) {
	repo := repoRoot(t)
	rows := architectureDependencyRows(t)

	// A row's package column may be a brace expansion such as
	// internal/domain/{lottery,nft} or a glob such as internal/app/*.
	for _, row := range rows {
		t.Run(row.packages, func(t *testing.T) {
			dirs := expandPackages(t, repo, row.packages)
			// A row may say that one package of the family is counted on its
			// own line. The composition root is that package: it exists to
			// import infra, which the use cases must not.
			if strings.Contains(row.dependencies, "container"+" 另计") {
				dirs = withoutContainer(dirs)
			}
			require.NotEmptyf(t, dirs, "the table row %q names no package under the checkout", row.packages)

			actual := map[string]bool{}
			for _, dir := range dirs {
				for _, dep := range projectImports(t, filepath.Join(repo, dir)) {
					actual[dep] = true
				}
			}

			// The table names layers, not packages: "domain" stands for every
			// domain package a row reaches, "app" for the use cases. A declared
			// name therefore covers itself and everything beneath it.
			declared := splitList(row.dependencies)
			// A parenthetical that names no package is the table saying the row
			// has no Aurora imports of its own; an empty set is then correct.
			if noInternalDeps(row.dependencies) {
				require.Emptyf(t, actual,
					"%s is documented as having no internal dependencies but imports %v",
					row.packages, strings.Join(sortedKeys(actual), ", "))
				return
			}
			// The "/*" spelling says "every package of that layer", so it covers
			// anything under the prefix; a bare name does the same, which is how
			// the table writes a layer it depends on as a whole.
			covers := func(name, imp string) bool {
				name = strings.TrimSuffix(name, "/*")
				return imp == name || strings.HasPrefix(imp, name+"/")
			}

			var missing, extra []string
			for imp := range actual {
				covered := false
				for _, name := range declared {
					if covers(name, imp) {
						covered = true
						break
					}
				}
				if !covered {
					missing = append(missing, imp)
				}
			}
			for _, name := range declared {
				used := false
				for imp := range actual {
					if covers(name, imp) {
						used = true
						break
					}
				}
				if !used {
					extra = append(extra, name)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)
			assert.Emptyf(t, missing,
				"%s imports %s, which its row in docs/architecture.md omits", row.packages, strings.Join(missing, ", "))
			assert.Emptyf(t, extra,
				"%s row in docs/architecture.md lists %s, which the package does not import", row.packages, strings.Join(extra, ", "))
		})
	}
	require.GreaterOrEqual(t, len(rows), 8, "the table must still describe the layering")
}

// noInternalDeps reports whether a row's dependency cell says the package has
// no Aurora imports: "（无内部依赖）" or a note about third-party only. Those
// rows are checked the other way round, so an import appearing there fails.
func noInternalDeps(cell string) bool {
	return strings.Contains(cell, "无内部依赖") || strings.Contains(cell, "仅第三方")
}

// sortedKeys lists a set in a stable order for a failure message.
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// dependencyRow is one line of the architecture document's dependency table.
type dependencyRow struct {
	packages     string
	dependencies string
}

// architectureDependencyRows parses the markdown table under the document's
// "当前依赖事实" heading. It stops at the next heading, so a later table about
// something else cannot be mistaken for dependency facts.
func architectureDependencyRows(t *testing.T) []dependencyRow {
	t.Helper()
	text := readRepoFile(t, "docs", "architecture.md")
	lines := strings.Split(text, "\n")

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.Contains(line, "当前依赖事实") {
			start = i + 1
			break
		}
	}
	require.GreaterOrEqual(t, start, 0, "docs/architecture.md must keep its 当前依赖事实 section")

	var rows []dependencyRow
	for _, line := range lines[start:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := tableCells(line)
		if len(cells) < 2 {
			continue
		}
		if strings.HasPrefix(cells[0], "包") || strings.HasPrefix(cells[0], "---") {
			continue
		}
		rows = append(rows, dependencyRow{packages: cells[0], dependencies: cells[1]})
	}
	return rows
}

// tableCells splits one markdown table line into its backtick-stripped cells.
func tableCells(line string) []string {
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|"), "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(strings.ReplaceAll(part, "`", "")))
	}
	return cells
}

// shortenImport is the form the table uses for a root-module package: the
// directory under internal/, without the internal/ prefix. Packages in the
// nested desktop module keep their full path, because their row sits beside a
// column that already talks about desktop/internal.
func shortenImport(path string) string {
	return strings.TrimPrefix(path, "internal/")
}

// expandPackages turns a row's package column into directories relative to the
// repository root: a plain path, a brace expansion listing siblings, or a glob.
func expandPackages(t *testing.T, repo, spec string) []string {
	t.Helper()
	spec = strings.TrimSpace(spec)
	var prefixes []string
	if open := strings.Index(spec, "{"); open >= 0 {
		if close := strings.Index(spec[open:], "}"); close > 0 {
			close += open
			base := spec[:open]
			for _, name := range strings.Split(spec[open+1:close], ",") {
				prefixes = append(prefixes, base+strings.TrimSpace(name)+spec[close+1:])
			}
		}
	}
	if len(prefixes) == 0 {
		prefixes = []string{spec}
	}

	var dirs []string
	for _, prefix := range prefixes {
		if strings.ContainsAny(prefix, "*?") {
			matches, err := filepath.Glob(filepath.Join(repo, prefix))
			require.NoError(t, err)
			// A glob names packages, not files: keep the directories only.
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && info.IsDir() {
					dirs = append(dirs, strings.TrimPrefix(match, repo+string(filepath.Separator)))
				}
			}
			continue
		}
		dirs = append(dirs, prefix)
	}
	sort.Strings(dirs)
	return dirs
}

// withoutContainer drops the app composition root from a package list.
func withoutContainer(dirs []string) []string {
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if filepath.Base(dir) == "container" {
			continue
		}
		out = append(out, dir)
	}
	return out
}

// projectImports lists the repository's own packages that dir imports, in the
// short form the table uses: internal/foo for a root-module package, or the
// module-relative path for a nested one.
func projectImports(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !strings.HasPrefix(value, "github.com/pplmx/aurora/") {
				continue
			}
			// The table writes root-module paths without their "internal/"
			// prefix; normalize both sides so a row reads "domain/token".
			seen[shortenImport(strings.TrimPrefix(value, "github.com/pplmx/aurora/"))] = true
		}
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// splitList reads a table cell such as "`ledger`, `utils`、`viper`（例外）"
// into package names, dropping the parenthetical notes and the brace shorthand.
func splitList(cell string) []string {
	cell = strings.ReplaceAll(cell, "`", "")
	cell = strings.NewReplacer("，", ",", "、", ",", "（", "(", "）", ")").Replace(cell)
	// Drop parentheticals, including the "(container 例外)" style notes.
	var kept []string
	depth := 0
	for _, r := range cell {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				kept = append(kept, string(r))
			}
		}
	}
	var out []string
	for _, part := range strings.Split(strings.Join(kept, ""), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, expandShorthand(part)...)
	}
	return out
}

// expandShorthand turns "domain/{a,b}" into domain/a and domain/b. A wildcard
// such as app/* is kept as written: the table uses it to mean every package of
// that layer, and the comparison treats it as a prefix.
func expandShorthand(name string) []string {
	if strings.ContainsAny(name, "*?") {
		return []string{name}
	}
	open := strings.Index(name, "{")
	if open < 0 {
		return []string{name}
	}
	close := strings.Index(name[open:], "}")
	if close < 0 {
		return []string{name}
	}
	close += open
	base := name[:open]
	suffix := name[close+1:]
	var out []string
	for _, part := range strings.Split(name[open+1:close], ",") {
		out = append(out, base+strings.TrimSpace(part)+suffix)
	}
	return out
}

// TestArchitectureStructureListsEveryDirectory covers the tree in
// docs/architecture.md's "目标结构" section. It describes the same package set the
// README does, and only the README's copy was checked: a directory added to the
// module would make one tree stale and the other not, with the reference the
// place a reader goes for the layering.
//
// The section's tree names internal/ children one level deep, so that is what is
// compared. Sub-packages are listed on a continuation line (domain/errdetail and
// domain/repoerr), which this check reads through their parent.
func TestArchitectureStructureListsEveryDirectory(t *testing.T) {
	doc := readRepoFile(t, "docs", "architecture.md")
	start := strings.Index(doc, "## 目标结构")
	require.NotEqual(t, -1, start, "the document must keep its structure section")
	section := doc[start:]
	if end := strings.Index(section, "\n## "); end > 0 {
		section = section[:end]
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "internal"))
	require.NoError(t, err)
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		checked++
		name := entry.Name()
		assert.Truef(t, strings.Contains(section, "├── "+name+"/") || strings.Contains(section, "└── "+name+"/"),
			"docs/architecture.md must list internal/%s in its structure tree", name)
	}
	require.GreaterOrEqual(t, checked, 10, "the tree check must see the package set")

	// The command and desktop roots are listed too, beside internal/.
	for _, root := range []string{"cmd/", "desktop/", "tools/"} {
		assert.Containsf(t, section, root, "the structure tree must name %s", root)
	}
}
