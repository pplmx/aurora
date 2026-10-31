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

	"github.com/pplmx/aurora/internal/i18n"
)

// translatedCallArgs lists the helpers that translate literal string
// arguments themselves. The test below walks them so a new visible label
// cannot silently fall back to English when Chinese is selected.
var translatedCallArgs = map[string][]int{
	"tr":                   {0},
	"field":                {0},
	"fieldWithError":       {0},
	"validatedField":       {0},
	"fieldHint":            {0, 1},
	"textArea":             {0},
	"textAreaHint":         {0, 1},
	"hint":                 {0},
	"emptyNote":            {0},
	"copyRow":              {0},
	"primaryButton":        {0},
	"secondaryButton":      {0},
	"dangerButton":         {0},
	"quietButton":          {0},
	"errorStatus":          {0},
	"tabs":                 {1},
	"sectionHeading":       {0, 1},
	"sectionRule":          {0},
	"formCard":             {0, 1},
	"emptyState":           {2, 3},
	"firstRunPath":         {2, 3, 4, 5, 6, 7},
	"badge":                {0},
	"guide":                {2, 3, 4, 5, 6},
	"loadWarning":          {0},
	"preferenceStateLabel": {1, 2},
	"pagerControls":        {1},
	"collectionToolbar":    {2, 3},
	"collectionList":       {1},
	"collectionPager":      {1},
	// Helpers that translate their label parameter: without these entries a
	// label could fall back to English in the Chinese window, invisible to a
	// test that only reads tables.
	"chipPicker":     {1},
	"eyebrow":        {1},
	"verdictPill":    {2},
	"searchField":    {2},
	"actionSubject":  {2, 3, 4, 5},
	"emptyPanel":     {3},
	"emptyStateIcon": {3, 4},
	"pagerInline":    {1},
	"pagerButton":    {1},
	"okStatus":       {0},
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

// TestChineseTranslationsKeepChromeVocabularyInChinese catches the half-
// translated state the visual audit found: a Chinese window still said
// Count, History, Spender, Show All or Setup inside otherwise translated
// guidance, while protocol names such as NFT, VRF and JSON correctly stay.
func TestChineseTranslationsKeepChromeVocabularyInChinese(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("zh")
	cases := []struct {
		source    string
		want      string
		forbidden []string
	}{
		{
			"Participants and seed determine the draw; the count is how many winners to pick.",
			"中奖人数",
			[]string{"Count"},
		},
		{
			"Choose the new row in History and press Verify to recompute the winners from the stored proof.",
			"“历史”",
			[]string{"History", "Verify"},
		},
		{
			"Let a spender move up to an amount on the owner's behalf.",
			"被授权方",
			[]string{"spender", "Spender"},
		},
		{
			"Spender (base64 public key)",
			"被授权方（base64 公钥）",
			[]string{"Spender"},
		},
		{
			"Mint/Approve/Burn sign with the owner private key; Transfer with the sender's; Transfer From with the spender's. Keys are never stored, and Use generated owner key works only for the token that key created.",
			"铸造、授权和销毁",
			[]string{"Mint", "Transfer", "Approve", "Burn", "Use generated owner key"},
		},
		{
			"No NFTs for this owner. Show All restores the collection.",
			"“显示全部”",
			[]string{"Show All"},
		},
		{
			"Token URI",
			"代币 URI",
			[]string{"Token"},
		},
		{
			"Select a source in Sources to fetch it now, enable/disable it, or delete it.",
			"“数据源”",
			[]string{"Sources"},
		},
		{
			"Health shows the scheduler's attempts and last error; Readings queries stored values or the latest one.",
			"“健康”",
			[]string{"Health", "Readings"},
		},
		{
			"Register voters and candidates in Setup. A voter registration returns the only copy of that voter's signing key.",
			"“准备”",
			[]string{"Setup"},
		},
		{
			"The badge verifies the chain. Search matches module, payload or height across the whole chain; Show sets the page size.",
			"“显示条数”",
			[]string{"Show"},
		},
		{
			"Applies to Dashboard, Oracle health and open readings; forms keep their text. Saved for the next launch.",
			"仪表盘",
			[]string{"Dashboard", "Oracle"},
		},
		{"Standard", "标准", nil},
		{"Enhanced", "已增强", nil},
		{"Enabled", "已启用", nil},
		{"Reduced", "已减弱", nil},
	}

	for _, tc := range cases {
		translated := Translate(tc.source)
		require.Contains(t, translated, tc.want)
		for _, forbidden := range tc.forbidden {
			require.NotContainsf(t, translated, forbidden,
				"%q still contains untranslated UI chrome %q", translated, forbidden)
		}
	}
}

// TestDesktopTranslationCoversLabelLiterals covers the labels that reach tr
// through a struct field instead of a call argument: readoutRow,
// collectionFact, collectionField, the palette hints and the Settings shortcut
// rows all translate their `label` inside the view, so the call-site walk above
// cannot see them and a missing entry would show English in a Chinese window.
func TestDesktopTranslationCoversLabelLiterals(t *testing.T) {
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
			composite, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range composite.Elts {
				pair, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok || (key.Name != "label" && key.Name != "Label") {
					continue
				}
				value, ok := pair.Value.(*ast.BasicLit)
				if !ok || value.Kind != token.STRING {
					continue
				}
				text, err := strconv.Unquote(value.Value)
				require.NoError(t, err)
				if text == "" || literalInEveryLanguage[text] {
					continue
				}
				checked++
				require.Containsf(t, desktopMessages, text,
					"%s:%d has a label %q without a Chinese translation",
					name, fset.Position(value.Pos()).Line, text)
			}
			return true
		})
	}
	require.GreaterOrEqual(t, checked, 30,
		"the label audit must inspect the rendered label surface")
}
