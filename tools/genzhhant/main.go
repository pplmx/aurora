// Command genzhhant regenerates the Traditional Chinese catalogs from the
// Simplified ones with OpenCC's Simplified->Traditional dictionaries.
//
//	go run ./tools/genzhhant -dict-dir /path/to/opencc/data/dictionary
//
// The dictionaries are not vendored: fetch STPhrases.txt and STCharacters.txt
// from BYVoid/OpenCC and point -dict-dir at the directory holding them. A
// wording fix belongs in the Simplified table; regeneration updates the
// Traditional one, and -check reports whether the checked-in files are current.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// target is one generated catalog: the Simplified source file and variable,
// and the Traditional file to produce.
type target struct {
	src    string
	pkg    string
	srcVar string
	dstVar string
	header string
}

var targets = []target{
	{
		src:    "internal/i18n/messages_zh.go",
		pkg:    "i18n",
		srcVar: "chineseMessages",
		dstVar: "traditionalMessages",
		header: `// traditionalMessages is the catalog for "zh-Hant". It is generated from
// chineseMessages with OpenCC's Simplified->Traditional phrase and character
// tables (BYVoid/OpenCC data/dictionary/STPhrases.txt and STCharacters.txt,
// s2t), not translated independently, so a wording fix belongs in the
// Simplified table and a regeneration follows. The key set must stay equal to
// chineseMessages (TestTraditionalCatalogCoversEveryChineseKey).`,
	},
	{
		src:    "desktop/internal/nativeui/messages_zh.go",
		pkg:    "nativeui",
		srcVar: "desktopMessages",
		dstVar: "desktopMessagesHant",
		header: `// desktopMessagesHant translates the native desktop's English labels into
// Traditional Chinese. It is generated from desktopMessages with OpenCC's
// Simplified->Traditional phrase and character tables (BYVoid/OpenCC
// data/dictionary/STPhrases.txt and STCharacters.txt, s2t), then reviewed, so a
// wording fix belongs in the Simplified table and a regeneration follows. The
// key set must stay equal to desktopMessages
// (TestTraditionalDesktopCatalogCoversEveryKey).`,
	},
}

func main() {
	dictDir := flag.String("dict-dir", "", "directory holding STPhrases.txt and STCharacters.txt")
	check := flag.Bool("check", false, "report stale catalogs instead of writing them")
	flag.Parse()
	if *dictDir == "" {
		fmt.Fprintln(os.Stderr, "genzhhant: -dict-dir is required")
		os.Exit(2)
	}

	phrases, err := loadDictionary(filepath.Join(*dictDir, "STPhrases.txt"))
	if err != nil {
		fail(err)
	}
	chars, err := loadDictionary(filepath.Join(*dictDir, "STCharacters.txt"))
	if err != nil {
		fail(err)
	}
	conv := newConverter(phrases, chars)

	for _, t := range targets {
		messages, err := parseStringMap(t.src, t.srcVar)
		if err != nil {
			fail(err)
		}
		generated, err := render(t, messages, conv)
		if err != nil {
			fail(err)
		}
		if *check {
			done, err := os.ReadFile(targetPath(t))
			if err != nil {
				fail(err)
			}
			if !bytes.Equal(done, generated) {
				fail(fmt.Errorf("%s is stale; run genzhhant without -check", targetPath(t)))
			}
			fmt.Println("current:", targetPath(t))
			continue
		}
		if err := os.WriteFile(targetPath(t), generated, 0o644); err != nil {
			fail(err)
		}
		fmt.Println("wrote:", targetPath(t))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "genzhhant:", err)
	os.Exit(1)
}

// targetPath is the generated file beside the source catalog.
func targetPath(t target) string {
	return filepath.Join(filepath.Dir(t.src), "messages_zh_hant.go")
}

// render builds the generated file and formats it the way gofmt would, so a
// regeneration and a hand edit produce the same bytes.
func render(t target, messages map[string]string, conv converter) ([]byte, error) {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n%s\nvar %s = map[string]string{\n", t.pkg, t.header, t.dstVar)
	for _, key := range keys {
		fmt.Fprintf(&buf, "\t%s: %s,\n", strconv.Quote(key), strconv.Quote(conv.convert(messages[key])))
	}
	buf.WriteString("}\n")
	return format.Source(buf.Bytes())
}

// parseStringMap reads a string-keyed, string-valued map declaration from a Go
// file, preserving the entries. Comments and layout are not needed: the
// generated file has its own.
func parseStringMap(path, name string) (map[string]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != name || len(value.Values) != 1 {
				continue
			}
			lit, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("%s: %s is not a map literal", path, name)
			}
			out := make(map[string]string, len(lit.Elts))
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					return nil, fmt.Errorf("%s: %s has a non key-value entry", path, name)
				}
				key, err := stringLiteral(kv.Key)
				if err != nil {
					return nil, err
				}
				value, err := stringLiteral(kv.Value)
				if err != nil {
					return nil, err
				}
				out[key] = value
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("%s: var %s not found", path, name)
}

func stringLiteral(expr ast.Expr) (string, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", fmt.Errorf("expected a string literal, got %T", expr)
		}
		return strconv.Unquote(e.Value)
	case *ast.BinaryExpr:
		// A long sentence may be split across lines with "+".
		if e.Op != token.ADD {
			return "", fmt.Errorf("unsupported operator %s in a catalog value", e.Op)
		}
		left, err := stringLiteral(e.X)
		if err != nil {
			return "", err
		}
		right, err := stringLiteral(e.Y)
		if err != nil {
			return "", err
		}
		return left + right, nil
	case *ast.ParenExpr:
		return stringLiteral(e.X)
	default:
		return "", fmt.Errorf("expected a string literal, got %T", expr)
	}
}

// loadDictionary reads an OpenCC dictionary: each line is
// "key<TAB>candidate candidate ...". The first candidate is the preferred
// conversion; a key that would map to itself is dropped.
func loadDictionary(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}
	table := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		first, _, _ := strings.Cut(rest, " ")
		if key != first {
			table[key] = first
		}
	}
	return table, nil
}

// converter applies OpenCC's longest-match phrase table, then the character
// table, then two modern-form overrides: OpenCC's s2t tables still prefer 爲 and
// 燬, which a modern UI should render as 為 and 毀.
type converter struct {
	phrases map[string]string
	chars   map[string]string
	maxLen  int
}

func newConverter(phrases, chars map[string]string) converter {
	maxLen := 0
	for key := range phrases {
		if n := len([]rune(key)); n > maxLen {
			maxLen = n
		}
	}
	return converter{phrases: phrases, chars: chars, maxLen: maxLen}
}

var overrides = map[string]string{"爲": "為", "燬": "毀"}

func (c converter) convert(text string) string {
	runes := []rune(text)
	var out strings.Builder
	for i := 0; i < len(runes); {
		matched := false
		for n := min(c.maxLen, len(runes)-i); n >= 2; n-- {
			if replacement, ok := c.phrases[string(runes[i:i+n])]; ok {
				out.WriteString(replacement)
				i += n
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		ch := string(runes[i])
		if replacement, ok := c.chars[ch]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(ch)
		}
		i++
	}
	converted := out.String()
	for old, modern := range overrides {
		converted = strings.ReplaceAll(converted, old, modern)
	}
	return converted
}
