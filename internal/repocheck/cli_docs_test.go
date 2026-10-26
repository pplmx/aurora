package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
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
// words after `aurora` in every documented command line and rejects a word
// that is not a known subcommand at that depth. Flags, placeholders and
// example values end the walk, so `aurora migrate up 2`, a `<id>` placeholder
// and a `./backups/...` path stay valid. This is how a renamed command leaves
// a dead instruction behind.
func TestCLIDocsPathsMatchTheCommandTree(t *testing.T) {
	root := repoRoot(t)
	docs := readRepoFile(t, "docs", "cli.md")
	known := map[string]bool{}
	for _, path := range cliCommandPaths(t, filepath.Join(root, "cmd", "aurora", "cmd")) {
		known["aurora "+path] = true
	}

	for _, line := range strings.Split(docs, "\n") {
		trimmed := strings.TrimSpace(line)
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
				t.Fatalf("docs/cli.md documents %q, but %q is not a command", trimmed, candidate)
			}
			current = candidate
		}
	}
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
