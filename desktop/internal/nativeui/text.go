package nativeui

import "github.com/pplmx/aurora/internal/i18n"

// tr translates a desktop label. The English literal is the key: a missing
// translation falls back to that literal, so a new label is readable before
// its Chinese entry lands. Desktop labels live in messages_zh.go; keys that
// belong to the service/CLI layer (errors, menu entries) fall through to the
// shared catalog.
func tr(text string) string { return Translate(text) }

// labelText joins a label and its value with the separator the language uses,
// the same rule the CLI applies: Chinese puts a full-width colon after a label,
// and the desktop tables already translate every label into Chinese without
// one. Joining by hand hard-coded the half-width form, so a Chinese window
// showed 版本: 0.0.1 in its diagnostics text and announced "失败: 读取: …".
func labelText(label, value string) string {
	return label + i18n.GetTranslator().Separator() + value
}

// copyActionLabel names a Copy button after the thing it copies: "Copy
// Diagnostics" in English, 复制诊断信息 in Chinese.
//
// The two words are joined by i18n.JoinWords rather than a literal space.
// Chinese does not space two Chinese words apart, so building the label as
// tr("Copy") + " " + tr("Diagnostics") printed 复制 诊断信息 while the catalog's
// own composed entries — 复制载荷, 复制代币 ID — had no space at all.
func copyActionLabel(target string) string {
	return i18n.JoinWords(tr("Copy"), tr(target))
}

// colonText is the separator alone, for a heading that names a group of lines
// rather than a single value.
func colonText() string {
	return i18n.Colon()
}

// Translate is tr for callers outside the view package, such as the desktop
// shell's dialogs. It only consults a desktop table when the shared translator
// is in Chinese; English keeps the literal key. The desktop vocabulary is not
// in the core catalog, so a missing entry falls through to the shared
// translator and finally to the English key.
func Translate(text string) string {
	switch i18n.GetTranslator().GetLocale() {
	case "zh":
		if translated, ok := desktopMessages[text]; ok {
			return translated
		}
	case "zh-Hant":
		if translated, ok := desktopMessagesHant[text]; ok {
			return translated
		}
	}
	return i18n.GetText(text)
}
