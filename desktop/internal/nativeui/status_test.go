package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestExportDestinationErrorsLocalize: the window shares the export path with
// the CLI, and the two destination checks carry a path, so a full-message lookup
// cannot reach them; the sentence around the path is what translates.
func TestExportDestinationErrorsLocalize(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	require.Equal(t, "导出路径为空", localizeServiceErrorMessage("export path is empty"))
	require.Equal(t, "导出目录 /tmp/missing 不存在",
		localizeServiceErrorMessage("export directory /tmp/missing does not exist"))
	// Anything else keeps its words so a diagnostic still matches the log.
	require.Equal(t, "export directory /tmp/x does not exist (wrapped)",
		localizeServiceErrorMessage("export directory /tmp/x does not exist (wrapped)"))
}

// TestServiceMessagesTheWindowCanShowAreTranslated covers the sentinel texts a
// window action can surface (a fetch with no body, an oversized import, a draw
// that failed verification): an untranslated one appears verbatim in a Chinese
// window.
func TestServiceMessagesTheWindowCanShowAreTranslated(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	for message, want := range map[string]string{
		"fetcher returned no data":              "数据源未返回数据",
		"import file too large":                 "导入文件过大",
		"lottery record not found":              "未找到抽奖记录",
		"lottery record failed integrity check": "抽奖记录完整性校验失败",
		"empty public key":                      "公钥为空",
	} {
		require.Equal(t, want, localizeServiceErrorMessage(message), message)
	}
}

// TestWrappedPersistenceErrorsKeepTheirCauseReadable covers the balance and
// supply overflow rejections. The repository wraps the domain cause after a
// long English sentence, so only the tail is in the catalog: the localizer
// tries the head, then the tail, and the sentence before the cause stays as it
// is. Without the wrapped sentinel the whole line arrived untranslated, because
// neither "try add balance" nor "balance would exceed maximum" is a key.
func TestWrappedPersistenceErrorsKeepTheirCauseReadable(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	require.Equal(t,
		"try add to supply: total supply would exceed maximum: 金额过大",
		localizeServiceErrorMessage("try add to supply: total supply would exceed maximum: amount too large"),
		"the supply cause is the translatable part")
	require.Equal(t,
		"try add balance: balance would exceed maximum: 金额过大",
		localizeServiceErrorMessage("try add balance: balance would exceed maximum: amount too large"),
		"the balance cause is the translatable part")

	// The bare form has no key at any level, so it stays verbatim and still
	// matches the log line a reader would search for.
	require.Equal(t,
		"try add balance: balance would exceed maximum",
		localizeServiceErrorMessage("try add balance: balance would exceed maximum"))
}
