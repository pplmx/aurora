package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestParenMatchesTheLanguage covers the bracket around a note. Chinese sets it
// against the preceding word and the Latin spelling keeps a space, so the
// helper owns the space rather than each call site guessing.
func TestParenMatchesTheLanguage(t *testing.T) {
	translator := GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("en")
	require.Equal(t, " (80.1 KiB)", Paren("80.1 KiB"))

	translator.SetLocale("zh")
	require.Equal(t, "（80.1 KiB）", Paren("80.1 KiB"))

	translator.SetLocale("zh-Hant")
	require.Equal(t, "（80.1 KiB）", Paren("80.1 KiB"))
}

// TestJoinWordsClosesOnlyTheCJKBoundary covers the label built out of two
// translated words. The desktop composed "Copy" and "Diagnostics" with a
// literal space, so a Chinese window showed 复制 诊断信息 beside catalog entries
// like 复制载荷 that carry no space.
func TestJoinWordsClosesOnlyTheCJKBoundary(t *testing.T) {
	tr := GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("en")
	require.Equal(t, "Copy Diagnostics", JoinWords("Copy", "Diagnostics"))

	tr.SetLocale("zh")
	require.Equal(t, "复制诊断信息", JoinWords("复制", "诊断信息"), "two Chinese words need no space")
	require.Equal(t, "复制 GLD", JoinWords("复制", "GLD"),
		"a Latin word keeps the space the project's Chinese prose uses")
	require.Equal(t, "GLD 代币", JoinWords("GLD", "代币"),
		"the boundary closes only when both sides are Chinese")

	tr.SetLocale("zh-Hant")
	require.Equal(t, "複製診斷信息", JoinWords("複製", "診斷信息"))

	// A half-built label is a real case: a missing catalog entry falls back to
	// English, and dropping the space then would glue two Latin words together.
	require.Equal(t, "複製 Diagnostics", JoinWords("複製", "Diagnostics"))
	require.Equal(t, "Copy", JoinWords("Copy", ""))
	require.Equal(t, "Diagnostics", JoinWords("", "Diagnostics"))
	require.Equal(t, "", JoinWords("", ""))
}

// TestJoinWordsKeepsTheSeparatorForBytesThatDoNotDecode keeps a corrupt string
// from being glued together: DecodeLastRuneInString reports RuneError for a
// broken byte, and reading that as a Chinese character would drop the space
// between two Latin words.
func TestJoinWordsKeepsTheSeparatorForBytesThatDoNotDecode(t *testing.T) {
	require.Equal(t, "Copy \xff", JoinWords("Copy", "\xff"))
	require.Equal(t, "\xff Diagnostics", JoinWords("\xff", "Diagnostics"))
}

// TestDisplayWidthCountsWhatTheCatalogsPrint covers the column measurement
// printFields relies on. Byte length would be wrong in the other direction (a
// Chinese label is three bytes per character), and rune count would treat 名称
// and "ID" as neighbours when the first is twice as wide on screen.
func TestDisplayWidthCountsWhatTheCatalogsPrint(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"ID", 2},
		{"Owner", 5},
		{"Total Supply", 12},
		{"名称", 4},
		{"所有者", 6},
		// The full-width marks the Chinese catalogs print take two columns each,
		// which is what makes a padded block line up in a terminal.
		{"ID：", 4},
		{"总计：", 6},
		{"余额：", 6},
		{"（可选）", 8},
		{"", 0},
		// Mixed: a Latin word inside a Han label keeps one column per Latin rune.
		{"NFT 详情：", 10},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, DisplayWidth(c.text), "DisplayWidth(%q)", c.text)
	}
}
