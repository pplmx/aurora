package nativeui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestSourceTypeIsTranslatedExceptTheFormatName covers the one field in the
// source row that is a word rather than an operator's data. The type arrives
// from the domain as "custom" or "price", and the Chinese window showed it
// untranslated beside translated labels; "json" names a format, so it keeps its
// spelling and tr() leaves it alone.
func TestSourceTypeIsTranslatedExceptTheFormatName(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	page := func(sourceType string) string {
		t.Helper()
		app := New(&fakeServices{sources: []SourceRow{{
			ID: "src-1", Name: "btc", Type: sourceType,
			URL: "https://example.test/price", Enabled: true,
		}}})
		tester := ui.NewTester(app.View, 1280, 1000)
		app.ShowPage("Oracle")
		tester.Frame()
		return strings.Join(tester.Texts(), " ")
	}

	translator.SetLocale("zh")
	require.Contains(t, page("custom"), "自定义", "the word is translated")
	require.NotContains(t, page("custom"), "custom ", "the domain's spelling is not shown")
	require.Contains(t, page("price"), "价格")
	require.Contains(t, page("json"), "json", "a format name is not a word to translate")

	translator.SetLocale("en")
	require.Contains(t, page("custom"), "custom", "English keeps the domain's word")
}
