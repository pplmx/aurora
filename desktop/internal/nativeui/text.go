package nativeui

import "github.com/pplmx/aurora/internal/i18n"

// tr translates a desktop label. The English literal is the key: a missing
// translation falls back to that literal, so a new label is readable before
// its Chinese entry lands. The desktop's own labels are kept in
// internal/i18n/messages_ui_zh.go rather than in the CLI tables.
func tr(text string) string { return i18n.GetText(text) }
