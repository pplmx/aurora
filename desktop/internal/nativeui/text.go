package nativeui

import "github.com/pplmx/aurora/internal/i18n"

// tr translates a desktop label. The English literal is the key: a missing
// translation falls back to that literal, so a new label is readable before
// its Chinese entry lands. Desktop labels live in messages_zh.go; keys that
// belong to the service/CLI layer (errors, menu entries) fall through to the
// shared catalog.
func tr(text string) string { return Translate(text) }

// Translate is tr for callers outside the view package, such as the desktop
// shell's dialogs. It only consults the desktop table when the shared
// translator is in Chinese; English keeps the literal key.
func Translate(text string) string {
	if i18n.GetTranslator().GetLocale() == "zh" {
		if translated, ok := desktopMessages[text]; ok {
			return translated
		}
	}
	return i18n.GetText(text)
}
