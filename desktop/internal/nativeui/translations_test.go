package nativeui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// translatedCallArgs lists the helpers that translate literal string
// arguments themselves. The test below walks them so a new visible label
// cannot silently fall back to English when Chinese is selected.
//
// The indices count every parameter, including the leading *ui.Context that a
// view helper takes. An off-by-one there is silent rather than loud: the walk
// simply finds a non-literal argument at that position and skips the call, so
// the audit keeps passing while an entire family of labels goes unchecked.
var translatedCallArgs = map[string][]int{
	"tr":                   {0},
	"field":                {1},
	"fieldWithError":       {1},
	"validatedField":       {1},
	"fieldHint":            {1, 2},
	"textAreaWithError":    {1},
	"textAreaHint":         {1, 2},
	"hint":                 {1},
	"emptyNote":            {1},
	"copyRow":              {1},
	"primaryButton":        {1},
	"secondaryButton":      {1},
	"dangerButton":         {1},
	"quietButton":          {1},
	"errorStatus":          {0},
	"tabs":                 {2},
	"sectionHeading":       {1, 2},
	"formCard":             {1, 2},
	"emptyState":           {2, 3},
	"firstRunPath":         {2, 3, 4, 5, 6, 7},
	"badge":                {1},
	"guide":                {2, 3, 4, 5, 6},
	"loadWarning":          {1},
	"preferenceStateLabel": {1, 2},
	"pagerControls":        {1, 3},
	"collectionToolbar":    {2, 3, 4},
	"collectionList":       {1},
	"collectionPager":      {1},
	// Helpers that translate their label parameter: without these entries a
	// label could fall back to English in the Chinese window, invisible to a
	// test that only reads tables.
	"chipPicker":         {1},
	"fieldPair":          {1, 3},
	"destructiveConfirm": {2, 3, 4},
	"eyebrow":            {1},
	"verdictPill":        {2},
	"searchField":        {2},
	"actionSubject":      {2, 3, 4, 5},
	"emptyPanel":         {3},
	"emptyStateIcon":     {3, 4},
	"pagerInline":        {1},
	"pagerButton":        {1},
	"okStatus":           {0},
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
			// One English word, two domains: the token page moves value, the NFT
			// page moves an asset. Sharing the key made the NFT tab read 转账
			// while every sentence around it said 转让.
			"Transfer NFT",
			"转让",
			[]string{"Transfer NFT"},
		},
		{
			// The token page calls the account behind a token its 所有者; the NFT
			// page calls the account holding an asset its 持有者, and uses that
			// word in thirteen sentences. One "Owner" key could only pick one,
			// so the NFT readout names the field Holder.
			"Holder",
			"持有者",
			[]string{"Holder"},
		},
		{
			// The signing rule and the storage rule are separate guide steps so
			// neither sentence has to carry the other's subject; both keep their
			// protocol names untranslated, like the rest of the page.
			"Each write signs with a different key: Mint/Approve/Burn use the owner's, Transfer the sender's, Transfer From the spender's.",
			"铸造、授权和销毁",
			[]string{"Mint", "Transfer", "Approve", "Burn", "owner"},
		},
		{
			// The sentence names the button in quotes, so it reads as a pointer
			// to that control instead of a claim about the key; the quoted name
			// must still reach Chinese.
			"Keys are never stored. “Use generated owner key” only works for the token that key created.",
			"使用已生成的所有者密钥",
			[]string{"Use generated owner key"},
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
			"The badge verifies the chain. Search matches the module, action, subject, a fact or the height across the whole chain; Show sets the page size.",
			"“显示条数”",
			[]string{"Show"},
		},
		{
			"Applies to Dashboard, the Oracle source list, health and open readings; forms keep their text. Saved for the next launch.",
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

// TestTranslatedCallArgsPointAtStringParameters keeps the audit above honest.
// A view helper's first parameter is its *ui.Context, so a label sits one slot
// later than the call might suggest. An index that points at the context, a
// *string or a bool is not an error the walk reports: it simply finds a
// non-literal argument there and skips the call, which is how two untranslated
// labels shipped while the audit kept passing. Every registered index must name
// a string parameter of the helper it is registered against.
func TestTranslatedCallArgsPointAtStringParameters(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	// funcs maps a helper to the parameter types its declaration spells out.
	// Variadic helpers list their variadic element type once, because a label
	// passed through it is still a string.
	funcs := map[string][]string{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Type.Params == nil {
				continue
			}
			var types []string
			for _, field := range fn.Type.Params.List {
				typ := typesExprString(field.Type)
				count := len(field.Names)
				if count == 0 {
					count = 1
				}
				for i := 0; i < count; i++ {
					types = append(types, typ)
				}
			}
			funcs[fn.Name.Name] = types
		}
	}

	for helper, indices := range translatedCallArgs {
		params, ok := funcs[helper]
		require.Truef(t, ok, "%q is registered as a translating helper but has no declaration", helper)
		// A trailing ...string accepts any number of labels past its position:
		// firstRunPath's steps and guide's steps both arrive that way.
		variadic := -1
		if len(params) > 0 && params[len(params)-1] == ".string" {
			variadic = len(params) - 1
		}
		for _, index := range indices {
			if variadic >= 0 && index >= variadic {
				continue
			}
			require.Lessf(t, index, len(params),
				"%s has %d parameters; index %d is past its last argument",
				helper, len(params), index)
			require.Equalf(t, "string", params[index],
				"%s parameter %d is %s, not a string label: the audit would skip every call to it",
				helper, index, params[index])
		}
	}
}

// typesExprString renders the parameter types the audit cares about: a bare
// string, an ellipsis, and the name of everything else.
func typesExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typesExprString(t.X)
	case *ast.Ellipsis:
		return "." + typesExprString(t.Elt)
	case *ast.SelectorExpr:
		return typesExprString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
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

// formatVerb matches one fmt verb: flags, width, precision, an optional argument
// index and the verb letter.
var formatVerb = regexp.MustCompile(`%[-+# 0-9.]*(?:\[[0-9]+\])?[a-zA-Z%]`)

// desktopSourceText returns every non-catalogue Go source under the given
// directories, joined for a substring search.
func desktopSourceText(t *testing.T, dirs ...string) string {
	t.Helper()
	var b strings.Builder
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasPrefix(filepath.Base(path), "messages_") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			b.Write(data)
			b.WriteString("\n")
			return nil
		})
		require.NoError(t, err)
	}
	return b.String()
}

// TestDesktopTranslationHasNoUnreadKeys keeps the desktop table from growing
// dead weight. A key is read either by a literal in the window's own sources or
// at run time: service errors and decoded record facts are translated by their
// English text, so a key that appears anywhere in the service layer counts as
// read. A key in neither place can never be shown.
func TestDesktopTranslationHasNoUnreadKeys(t *testing.T) {
	window := desktopSourceText(t, filepath.Join("..", ".."))
	service := desktopSourceText(t, filepath.Join("..", "..", "..", "internal"),
		filepath.Join("..", "..", "..", "cmd"))
	require.NotEmpty(t, window)
	require.NotEmpty(t, service)

	var unread []string
	for key := range desktopMessages {
		if strings.Contains(window, key) || strings.Contains(service, key) {
			continue
		}
		unread = append(unread, key)
	}
	sort.Strings(unread)
	assert.Emptyf(t, unread, "these desktop keys are never read: %v", unread)
}

// TestDesktopTranslationKeepsThePlaceholders: the catalogue key is the English
// sentence, so a translation that loses a verb prints "%!s(MISSING)" or drops a
// value the sentence promised.
func TestDesktopTranslationKeepsThePlaceholders(t *testing.T) {
	verbs := func(s string) int {
		count := 0
		for _, verb := range formatVerb.FindAllString(s, -1) {
			if verb != "%%" {
				count++
			}
		}
		return count
	}
	for english, chinese := range desktopMessages {
		require.Equalf(t, verbs(english), verbs(chinese),
			"%q keeps a different number of placeholders\n  zh: %q", english, chinese)
	}
}
