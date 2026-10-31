package i18n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSeparatorFollowsTheLanguage pins the colon each language uses. Chinese
// prose puts a full-width colon after a label and the message tables already
// do; the printed labels were the one place a half-width one survived, on the
// same line as translated text.
func TestSeparatorFollowsTheLanguage(t *testing.T) {
	tr := GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	for _, tc := range []struct{ locale, separator, colon string }{
		{"en", ": ", ":"},
		{"zh", "：", "："},
		{"zh-Hans", "：", "："},
		{"zh-Hant", "：", "："},
	} {
		tr.SetLocale(tc.locale)
		require.Equalf(t, tc.separator, tr.Separator(), "Separator in %s", tc.locale)
		require.Equalf(t, tc.colon, Colon(), "Colon in %s", tc.locale)
	}
}

// TestLabelJoinsWithoutDoublingTheSeparator covers the shape every call site
// relies on: one separator, the value's own text untouched.
func TestLabelJoinsWithoutDoublingTheSeparator(t *testing.T) {
	tr := GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("en")
	require.Equal(t, "Blocks: 12", Label("Blocks", 12))
	require.Equal(t, "Head: 0bf0ca", Label("Head", "0bf0ca"))

	tr.SetLocale("zh")
	require.Equal(t, "区块：12", Label("区块", 12))
	require.Equal(t, "头区块：0bf0ca", Label("头区块", "0bf0ca"))
}
