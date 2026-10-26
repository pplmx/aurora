package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// translationKeyCall reports the translation key passed to one of the
// translation helpers, when the call takes a literal key.
//
// i18n.GetText/GetTextF are the package-level helpers; T/TFormat are the
// desktop shell's wrappers over them (pkg.T(...) from outside the package).
// requireConfirm/addConfirmFlag take (cmd, actionKey), and their action phrase
// is itself a translated key rendered through cli.confirm.flag/required, so a
// key missing there prints raw text at a destructive command's prompt.
// A key built at run time is skipped: it cannot be checked statically.
func translationKeyCall(call *ast.CallExpr) (string, bool) {
	var name string
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		name = fun.Name
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	default:
		return "", false
	}

	var arg int
	switch name {
	case "GetText", "GetTextF", "T", "TFormat":
		arg = 0
	case "requireConfirm", "addConfirmFlag":
		arg = 1
	default:
		return "", false
	}
	if len(call.Args) <= arg {
		return "", false
	}
	lit, ok := call.Args[arg].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	key, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return key, true
}

// catalogKeys lists the keys of the message map declared in a catalog file.
func catalogKeys(t *testing.T, path string) map[string]bool {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	require.NoError(t, err)

	keys := map[string]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		pair, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		lit, ok := pair.Key.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		key, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		keys[key] = true
		return true
	})
	require.NotEmpty(t, keys, "%s must declare the message map", path)
	return keys
}

// TestEveryTranslationKeyExists scans non-test Go sources for translation
// calls with a literal key and requires an entry in both shipped catalogs.
//
// A missing key is not a build error and not a compile-time check: the
// translator returns the key itself, so the user sees menu and --help text
// like "lottery.tui.title". That is how these keys were found in practice:
// the lottery module's Short and the repair command's Long both shipped raw.
// Keys built at run time (for example the cli.confirm.<action> family) are
// invisible to this scan by construction, so it checks only what it can see.
func TestEveryTranslationKeyExists(t *testing.T) {
	root := repoRoot(t)
	used := map[string]string{} // key -> first file that used it

	skipDirs := map[string]bool{
		".git": true, ".planning": true, "dist": true, "data": true,
		"logs": true, "node_modules": true, "testdata": true,
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// The catalogs and the translator itself are the implementation, not
		// call sites.
		if rel == filepath.Join("internal", "i18n", "messages_en.go") ||
			rel == filepath.Join("internal", "i18n", "messages_zh.go") ||
			rel == filepath.Join("internal", "i18n", "i18n.go") {
			return nil
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			key, ok := translationKeyCall(call)
			if !ok {
				return true
			}
			if _, seen := used[key]; !seen {
				used[key] = rel
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)

	// A floor keeps the test from passing vacuously if the scan stops finding
	// calls (for example because a helper was renamed).
	require.GreaterOrEqual(t, len(used), 100, "the scan must find the translation calls it exists to check")

	english := catalogKeys(t, filepath.Join(root, "internal", "i18n", "messages_en.go"))
	chinese := catalogKeys(t, filepath.Join(root, "internal", "i18n", "messages_zh.go"))
	for key, file := range used {
		require.Truef(t, english[key], "%s: translation key %q is missing from the English catalog", file, key)
		require.Truef(t, chinese[key], "%s: translation key %q is missing from the Chinese catalog", file, key)
	}
}
