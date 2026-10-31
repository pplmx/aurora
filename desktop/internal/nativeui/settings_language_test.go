package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLanguageLabelNamesEachLanguage keeps the Current row honest. The label is
// written in the language itself, matching the choices beside it, so a reader
// who cannot read the window's current language can still find their own.
func TestLanguageLabelNamesEachLanguage(t *testing.T) {
	for _, tc := range []struct{ tag, want string }{
		{"en", "English"},
		{"en-GB", "English"},
		{"zh-Hans", "简体中文"},
		{"zh-CN", "简体中文"},
		{"zh", "简体中文"},
		{"zh-Hant", "繁體中文"},
		{"zh-TW", "繁體中文"},
		{"zh-HK", "繁體中文"},
	} {
		require.Equalf(t, tc.want, languageLabel(tc.tag), "languageLabel(%q)", tc.tag)
	}
}

// TestLanguageLabelKeepsAnUnknownTag is the branch that matters when Aurora
// gains a language it does not ship: showing "English" for a French setting
// would be a false statement about the window. The tag is passed through
// instead, so the row still says what is in effect.
func TestLanguageLabelKeepsAnUnknownTag(t *testing.T) {
	require.Equal(t, "fr", languageLabel("fr"), "an unsupported tag is shown as itself")
	require.Equal(t, "ja-JP", languageLabel("ja-JP"))
}

// TestLanguageLabelOfAnEmptyChoice is the no-choice case. It is what a window
// shows before the shell has supplied a language — the Settings card starts
// with an em dash rather than claiming a language nobody selected.
func TestLanguageLabelOfAnEmptyChoice(t *testing.T) {
	require.Equal(t, "—", languageLabel(""))
	require.Equal(t, "—", languageLabel("   "))
}
