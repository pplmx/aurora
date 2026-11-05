package repocheck

import (
	"bytes"
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

	// The shared list covers the build and module caches; without them this
	// walk spends twelve seconds inside .gocache and .gomodcache.
	skipDir := skipWalkDir
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return walkErrToleratingDisappearance(err)
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
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

		// A literal pre-filter before the parser: 551 Go files hold 29 that call
		// a translation helper, and parsing the other 522 costs twelve seconds.
		// The filter is deliberately generous — any occurrence of one of the
		// helper names — so a call the parser would recognise is never skipped.
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !mayHoldTranslationCall(body) {
			return nil
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, body, 0)
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

// mayHoldTranslationCall reports whether the source mentions any helper name
// translationKeyCall accepts. It is a superset check: a file it rejects holds no
// call the parser would have found.
func mayHoldTranslationCall(source []byte) bool {
	for _, name := range []string{"GetText", "GetTextF", "TFormat", "requireConfirm", "addConfirmFlag", "T"} {
		if bytes.Contains(source, []byte(name)) {
			return true
		}
	}
	return false
}

// TestTheSupplyFlagDoesNotPromiseDecimals pins the one place a help string
// contradicted the code. `aurora token create --supply` parsed with
// token.NewAmountFromString, which takes a whole number of base units, while the
// flag's description said "positive decimal amount" — so a reader who followed
// the help got a parse error. The description is now true, and this keeps a
// later edit from quietly promising decimals again.
func TestTheSupplyFlagDoesNotPromiseDecimals(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"messages_en.go", "messages_zh.go", "messages_zh_hant.go"} {
		body, err := os.ReadFile(filepath.Join(root, "internal", "i18n", name))
		require.NoError(t, err)
		text := string(body)

		start := strings.Index(text, `"token.supply":`)
		require.GreaterOrEqualf(t, start, 0, "%s must define token.supply", name)
		end := strings.Index(text[start:], "\n")
		require.Greater(t, end, 0)
		line := text[start : start+end]

		require.NotContainsf(t, strings.ToLower(line), "decimal",
			"%s: the supply flag takes a whole number, so its help must not say decimal", name)
		require.NotContainsf(t, line, "正数金额",
			"%s: 金额 暗示可以带小数，实际只接受整数", name)
		require.NotContainsf(t, line, "正數金額",
			"%s: 金額 暗示可以帶小數，實際只接受整數", name)
	}

	// The code side of the same fact: the parser refuses a decimal, so the help
	// above is describing what really happens.
	entity, err := os.ReadFile(filepath.Join(root, "internal", "domain", "token", "entity.go"))
	require.NoError(t, err)
	require.Contains(t, string(entity), "new(big.Int).SetString(s, 10)",
		"a base-ten integer parse is what makes the help text accurate")
}

// TestKeyLabelsInErrorSentencesAreOneWord pins the label each app layer splices
// into an English sentence. decodeKey builds "invalid %s encoding", so the label
// is read by a person: the NFT layer passed "privatekey" and the token layer
// "privateKey", which produced two different misspellings of the same field in
// two different messages. The Chinese rewriter matched all three spellings, so
// nothing failed — only the English reader saw the defect.
func TestKeyLabelsInErrorSentencesAreOneWord(t *testing.T) {
	// The spellings that reach a sentence, once per label the layers share.
	bad := []string{`"privatekey"`, `"privateKey"`, `"private_key"`}
	for _, dir := range []string{"app", "domain"} {
		root := repoRoot(t)
		var walk func(string)
		walk = func(rel string) {
			entries, err := os.ReadDir(filepath.Join(root, "internal", rel))
			require.NoError(t, err)
			for _, entry := range entries {
				name := entry.Name()
				path := filepath.Join(root, "internal", rel, name)
				if entry.IsDir() {
					walk(filepath.Join(rel, name))
					continue
				}
				if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
					continue
				}
				body, err := os.ReadFile(path)
				require.NoError(t, err)
				for _, spelling := range bad {
					require.NotContainsf(t, string(body), "decodeKey("+spelling,
						"internal/%s/%s passes %s into a sentence; write it as \"private key\"",
						rel, name, spelling)
				}
			}
		}
		walk(dir)
	}
}

// TestEveryEntryPointReportsUnknownConfigSections holds the three ways Aurora
// starts to the same standard. They read the same file, so a misspelled section
// must not pass silently in one of them: the CLI prints the report, the window
// logs it, and the server does both. A helper with its own test proves nothing if
// its caller stops calling it, which is why this checks the call sites.
func TestEveryEntryPointReportsUnknownConfigSections(t *testing.T) {
	root := repoRoot(t)
	for _, entry := range []struct {
		path string
		want string
	}{
		{"cmd/aurora/cmd/root.go", "config.UnknownSections()"},
		{"desktop/internal/deskapp/deskapp.go", "warnUnknownSections()"},
		{"cmd/api/main.go", "reportUnknownSections(os.Stderr)"},
	} {
		body, err := os.ReadFile(filepath.Join(root, entry.path))
		require.NoError(t, err)
		require.Containsf(t, string(body), entry.want,
			"%s must report the config sections nothing reads, as the other entry points do", entry.path)
	}
}
