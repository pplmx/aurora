package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// cliCommandPaths parses the cobra command declarations in dir and the
// AddCommand wiring between them, then returns every command path below the
// root as "lottery create"-style strings.
//
// It reads source instead of importing the package because the command
// variables are unexported and importing the CLI would run its init (flag
// registration, config defaults) in a repository check that only needs the
// tree.
func cliCommandPaths(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// use maps a command variable to the first word of its Use field.
	use := map[string]string{}
	// parent maps a child command variable to the variable that adds it.
	parent := map[string]string{}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.GenDecl:
				if n.Tok != token.VAR {
					return true
				}
				for _, spec := range n.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, varName := range value.Names {
						if i >= len(value.Values) {
							continue
						}
						useField := cobraCommandUse(value.Values[i])
						if useField != "" {
							use[varName.Name] = useField
						}
					}
				}
			case *ast.CallExpr:
				sel, ok := n.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "AddCommand" {
					return true
				}
				parentIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				for _, arg := range n.Args {
					if childIdent, ok := arg.(*ast.Ident); ok {
						parent[childIdent.Name] = parentIdent.Name
					}
				}
			}
			return true
		})
	}

	var paths []string
	var walk func(name string, prefix []string)
	walk = func(name string, prefix []string) {
		path := prefix
		if name != "rootCmd" {
			part, ok := use[name]
			if !ok {
				t.Fatalf("command %s has no cobra.Command Use field (or the declaration changed shape)", name)
			}
			path = append(append([]string{}, prefix...), part)
			paths = append(paths, strings.Join(path, " "))
		}
		for child, par := range parent {
			if par == name {
				walk(child, path)
			}
		}
	}
	walk("rootCmd", nil)
	require.GreaterOrEqual(t, len(paths), 50, "the parser must find the CLI's command tree")
	return paths
}

// cobraCommandUse returns the first word of `Use: "..."` when value is a
// `&cobra.Command{...}` literal, or "" when it is not a command declaration.
func cobraCommandUse(value ast.Expr) string {
	unary, ok := value.(*ast.UnaryExpr)
	if !ok {
		return ""
	}
	literal, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return ""
	}
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Command" {
		return ""
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "cobra" {
		return ""
	}
	for _, elt := range literal.Elts {
		pair, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok || key.Name != "Use" {
			continue
		}
		lit, ok := pair.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		val, err := strconv.Unquote(lit.Value)
		if err != nil {
			continue
		}
		if fields := strings.Fields(val); len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

// TestCLIDocsMentionEveryCommand requires every command in the CLI tree to
// have a line in docs/cli.md. `aurora token list` shipped without one: the
// command worked and the reference never mentioned it, and nothing failed.
func TestCLIDocsMentionEveryCommand(t *testing.T) {
	root := repoRoot(t)
	docs := readRepoFile(t, "docs", "cli.md")
	for _, path := range cliCommandPaths(t, filepath.Join(root, "cmd", "aurora", "cmd")) {
		require.Containsf(t, docs, "aurora "+path, "docs/cli.md must document `aurora %s`", path)
	}
}

// TestCLIDocsPathsMatchTheCommandTree is the reverse direction: it walks the
// words after `aurora` in every command line in the current user-facing docs
// and rejects a word that is not a known subcommand at that depth. Flags,
// placeholders and example values end the walk, so `aurora migrate up 2`, a
// `<id>` placeholder and a `./backups/...` path stay valid. This is how a
// renamed or removed command leaves a dead instruction behind. Historical
// design records under docs/superpowers are intentionally excluded.
func TestCLIDocsPathsMatchTheCommandTree(t *testing.T) {
	root := repoRoot(t)
	known := map[string]bool{}
	for _, path := range cliCommandPaths(t, filepath.Join(root, "cmd", "aurora", "cmd")) {
		known["aurora "+path] = true
	}
	// Cobra adds the completion command when the root executes; it has no
	// source declaration, so a documented `aurora completion <shell>` line is
	// legitimate even though the scan above cannot see it.
	known["aurora completion"] = true
	hasChildren := map[string]bool{}
	for path := range known {
		words := strings.Fields(path)
		for i := 1; i < len(words); i++ {
			hasChildren[strings.Join(words[:i], " ")] = true
		}
	}

	for _, docPath := range currentCLIDocs(t, root) {
		docs, err := os.ReadFile(docPath)
		require.NoError(t, err)
		rel, err := filepath.Rel(root, docPath)
		require.NoError(t, err)
		for lineNumber, line := range strings.Split(string(docs), "\n") {
			trimmed := strings.TrimSpace(line)
			trimmed = strings.TrimPrefix(trimmed, "$ ")
			trimmed = strings.TrimPrefix(trimmed, "./")
			if !strings.HasPrefix(trimmed, "aurora ") {
				continue
			}
			current := "aurora"
			for _, word := range strings.Fields(strings.TrimPrefix(trimmed, "aurora ")) {
				if isExampleValue(word) {
					break
				}
				candidate := current + " " + word
				if !known[candidate] {
					// A bare file argument (`draws.json`) is not a subcommand;
					// no command name contains a dot or slash.
					if strings.ContainsAny(word, "./") {
						break
					}
					// A leaf command may take a positional example value that
					// has no punctuation (for example a raw lottery id). A
					// command that still has children cannot: `tui` after
					// `aurora lottery` is a removed subcommand, not an
					// argument.
					if !hasChildren[current] {
						break
					}
					t.Fatalf("%s:%d documents %q, but %q is not a command",
						filepath.ToSlash(rel), lineNumber+1, trimmed, candidate)
				}
				current = candidate
			}
		}
	}
}

// currentCLIDocs returns the Markdown files users consult for the current
// commands. Historical design records under docs/superpowers describe deleted
// surfaces and must not be treated as live instructions.
func currentCLIDocs(t *testing.T, root string) []string {
	t.Helper()
	paths := []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, "AGENTS.md"),
	}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "superpowers" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	require.NoError(t, err)
	return paths
}

// isExampleValue reports whether word is an argument or flag rather than a
// command name: a flag, a <placeholder>, a path, a number, a quoted value, a
// shell expansion or a trailing comment.
func isExampleValue(word string) bool {
	if word == "" {
		return true
	}
	if strings.HasPrefix(word, "-") || strings.HasPrefix(word, "<") ||
		strings.HasPrefix(word, "./") || strings.HasPrefix(word, "/") ||
		strings.HasPrefix(word, "~") || strings.HasPrefix(word, "$") ||
		strings.HasPrefix(word, `"`) || strings.HasPrefix(word, "'") ||
		strings.HasPrefix(word, "#") {
		return true
	}
	_, err := strconv.Atoi(word)
	return err == nil
}
