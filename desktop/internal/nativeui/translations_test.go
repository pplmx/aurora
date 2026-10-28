package nativeui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// translatedCallArgs lists the helpers that translate literal string
// arguments themselves. The test below walks them so a new visible label
// cannot silently fall back to English when Chinese is selected.
var translatedCallArgs = map[string][]int{
	"tr":              {0},
	"field":           {0},
	"fieldHint":       {0, 1},
	"textArea":        {0},
	"textAreaHint":    {0, 1},
	"hint":            {0},
	"fieldError":      {0},
	"emptyNote":       {0},
	"copyRow":         {0},
	"primaryButton":   {0},
	"secondaryButton": {0},
	"dangerButton":    {0},
	"quietButton":     {0},
	"sectionHeading":  {0, 1},
	"formCard":        {0, 1},
	"emptyState":      {2, 3},
	"actionCard":      {0},
	"metricCard":      {0, 1, 2},
	"badge":           {0},
	"verdictBadge":    {0},
	"guide":           {2, 3, 4, 5, 6},
	"loadWarning":     {0},
}

// These literals intentionally stay identical in every language: a brand,
// a numeric step marker and the em dash used for an empty value.
var literalInEveryLanguage = map[string]bool{
	"Aurora": true,
	"%d.":    true,
	"—":      true,
}

func TestDesktopTranslationCoversStaticCallLabels(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			for _, index := range translatedCallArgs[ident.Name] {
				if index >= len(call.Args) {
					continue
				}
				literal, ok := call.Args[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				key, err := strconv.Unquote(literal.Value)
				require.NoError(t, err)
				if key == "" || literalInEveryLanguage[key] {
					continue
				}
				checked++
				require.Containsf(t, desktopMessages, key, "%s:%d calls %s(%q) without a Chinese translation",
					name, fset.Position(literal.Pos()).Line, ident.Name, key)
			}
			return true
		})
	}
	require.GreaterOrEqual(t, checked, 180,
		"the translation audit must inspect the expected rendered-label surface")
}
