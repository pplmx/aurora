package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
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

// TestAStatusLineIsSaidInTheCurrentLanguage covers the half of a language switch that a
// frame cannot fix by itself. Every label is read from the translator while the frame is
// built, so those follow a switch; a status line is a value, translated once when it was
// created. Switching to Chinese therefore left "Failed: token name is required" under a
// Chinese "失败", and the line stayed until the next write replaced it.
//
// The line keeps its message and its arguments, not the finished sentence: the catalogs
// translate templates ("Registered %s"), so a formatted string could not be looked up.
func TestAStatusLineIsSaidInTheCurrentLanguage(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("en")
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Token")
	tester.Frame()

	app.token.status = okStatus("Registered %s", "Gold")
	tester.Frame()
	require.Equal(t, "Registered Gold", app.token.status.spoken())

	tr.SetLocale("zh")
	tester.Frame()
	require.Equal(t, "已注册 Gold", app.token.status.spoken(),
		"the line must follow the language the window is now drawing in")
	require.True(t, tester.HasText("已注册 Gold"), "and the redraw must show it")

	// Errors take the same path, through the service-error localizer.
	tr.SetLocale("en")
	app.token.status = errStatus(errors.New("token name is required"))
	require.Equal(t, "token name is required", app.token.status.spoken())
	tr.SetLocale("zh")
	require.Equal(t, "代币名称不能为空", app.token.status.spoken())

	// A fixed sentence carrying a literal percent is not a format.
	tr.SetLocale("en")
	app.token.status = okStatus("Saved 100% of the rows")
	require.Equal(t, "Saved 100% of the rows", app.token.status.spoken(),
		"the percent must survive, not be read as a verb")
}

// TestAnErrorWithNoMessageStillSaysSo covers the one failure that used to look like
// nothing happening. The status line decides whether it is worth drawing by looking at its
// text, so an error whose message is empty was not drawn at all: a click that failed left
// the page exactly as it was, with no word, no tint and no announcement.
//
// No production path builds such an error today. The guard is here because the failure it
// prevents costs the reader every signal at once, and the cost of the guard is one branch.
func TestAnErrorWithNoMessageStillSaysSo(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })
	tr.SetLocale("en")

	for _, message := range []string{"", "   ", "\n"} {
		line := errStatus(errors.New(message))
		require.Truef(t, line.isErr, "%q: a failed operation is still a failure", message)
		require.Falsef(t, line.empty(), "%q: an empty line is not drawn, so it must not be empty", message)
		require.Equal(t, "The operation failed without a message.", line.spoken(),
			"the reader must be told that something failed")
	}

	// The sentence is translated, since it is the window's own words.
	tr.SetLocale("zh")
	require.Equal(t, "操作失败，但没有返回错误信息。", errStatus(errors.New("")).spoken())

	// It reaches the frame, rather than being dropped between the value and the view.
	tr.SetLocale("en")
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1400, 4000)
	app.ShowPage("Token")
	app.token.status = errStatus(errors.New(""))
	tester.Frame()
	require.True(t, tester.HasText("The operation failed without a message."))
	require.True(t, tester.HasText("Failed"), "and the outcome word is beside it")
}
