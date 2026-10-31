package nativeui

import (
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestLabelTextFollowsTheLanguage pins the separator the desktop joins its own
// strings with. The window's labels are translated into Chinese without a colon
// — the code added one — so a hand-joined label put a half-width colon beside
// Chinese text in the diagnostics text, the status strip and the screen-reader
// announcements.
func TestLabelTextFollowsTheLanguage(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("en")
	require.Equal(t, "Version: 0.0.1", labelText(tr("Version"), "0.0.1"))
	require.Equal(t, "Failed: Voter Name: required", labelText(labelText(tr("Failed"), tr("Voter Name")), "required"))
	require.Equal(t, ":", colonText())

	translator.SetLocale("zh")
	require.Equal(t, "版本：0.0.1", labelText(tr("Version"), "0.0.1"))
	require.Equal(t, "失败：选民姓名：required", labelText(labelText(tr("Failed"), tr("Voter Name")), "required"))
	require.Equal(t, "：", colonText())
}

// TestStatusAnnouncementUsesTheLanguageSeparator covers the one caller whose
// text a screen reader speaks: it must not read a colon that is not there in
// the language being displayed.
func TestStatusAnnouncementUsesTheLanguageSeparator(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("zh")
	require.Equal(t, "失败：读取失败", statusAnnouncementText(errorStatusText("读取失败")))
	require.Equal(t, "完成：已保存", statusAnnouncementText(okStatus("已保存")))
}
