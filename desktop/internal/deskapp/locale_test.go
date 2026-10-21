package deskapp

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/i18n"
)

// withLocale runs fn with Aurora's translator in locale and restores the one
// the process started with: the translator is package state shared with the
// in-process API server, so a test must not leak its language into the next.
func withLocale(t *testing.T, locale string, fn func()) {
	t.Helper()
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	tr.SetLocale(locale)
	t.Cleanup(func() { tr.SetLocale(before) })
	fn()
}

// TestAuroraLocale covers the tag mapping MyGo's answer goes through. MyGo
// reports normalized BCP 47 tags of the languages it was told the app is
// written in; Aurora ships "en" and "zh" only, and a Chinese reader of any
// variant must not fall back to English.
func TestAuroraLocale(t *testing.T) {
	for _, tt := range []struct{ tag, want string }{
		{"en", "en"},
		{"en-GB", "en"},
		{"", "en"},
		{"zh", "zh"},
		{"zh-Hans", "zh"},
		{"zh-Hant-TW", "zh"},
		{"ZH-cn", "zh"},
		{"ja", "en"},
	} {
		assert.Equal(t, tt.want, auroraLocale(tt.tag), "auroraLocale(%q)", tt.tag)
	}
}

// TestMygoTag is the reverse direction, used for the configured locale: MyGo
// only knows "en" and "zh-Hans" here, and every Chinese variant must map onto
// the catalog that exists.
func TestMygoTag(t *testing.T) {
	assert.Equal(t, "en", mygoTag("en"))
	assert.Equal(t, "en", mygoTag(""))
	assert.Equal(t, "zh-Hans", mygoTag("zh"))
	assert.Equal(t, "zh-Hans", mygoTag("zh-CN"))
	assert.Equal(t, "zh-Hant", mygoTag("zh-TW"), "a configured Traditional locale keeps MyGo's roles Traditional")
	assert.Equal(t, "zh-Hant", mygoTag("zh-HK"))
	assert.Equal(t, "zh-Hant", mygoTag("zh-Hant-TW"))
}

func TestSyncLocale(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	assert.True(t, syncLocale("zh-Hans"), "switching to Chinese is a change")
	assert.Equal(t, "zh", tr.GetLocale())
	assert.False(t, syncLocale("zh-TW"), "another Chinese tag is the same language, not a change")
	assert.True(t, syncLocale("en-GB"), "switching back to English is a change")
	assert.Equal(t, "en", tr.GetLocale())
}

// TestMenuLabelsFollowTheLocale is the defect this work exists for: MyGo
// translates the labels of the items that carry a role ("File" as
// RoleFileMenu, "Quit", "Edit"), so a zh-CN session showed 文件/编辑/退出
// beside Aurora's hardcoded "Export Lotteries..." and "Back Up Database...".
// One menu, two languages.
func TestMenuLabelsFollowTheLocale(t *testing.T) {
	label := func(menu *mygo.Menu, id string) string {
		item := menu.ItemByID(id)
		require.NotNil(t, item, "the menu must have %q", id)
		return item.Label
	}

	withLocale(t, "en", func() {
		menu := NewMenu(About{})
		assert.Equal(t, "Export Lotteries...", label(menu, "export-lotteries"))
		assert.Equal(t, "Import Lotteries...", label(menu, "import-lotteries"))
	})

	withLocale(t, "zh", func() {
		menu := NewMenu(About{})
		assert.Equal(t, "导出抽奖记录…", label(menu, "export-lotteries"),
			"the menu Aurora writes itself must not stay English in a zh session")
		assert.Equal(t, "导入抽奖记录…", label(menu, "import-lotteries"))
	})
}

// TestImportReportFollowsTheLocale extends the single-source contract to the
// wording: the three import entry points already share importReport, so
// localizing there localizes all of them at once.
func TestImportReportFollowsTheLocale(t *testing.T) {
	withLocale(t, "zh", func() {
		title, message, detail := importReport(lotteryapp.ImportResult{Total: 2, Imported: 2}, "/tmp/draws.json")
		assert.Equal(t, "抽奖记录已导入", title)
		assert.Equal(t, "已导入 2 条抽奖记录。", message)

		partial := lotteryapp.ImportResult{Total: 3, Imported: 2, Rejected: []int{1}}
		title, message, detail = importReport(partial, "/tmp/draws.json")
		assert.Equal(t, "部分导入失败", title)
		assert.Equal(t, "已导入 2 / 3 条抽奖记录（失败 1 条）。", message)
		for _, want := range []string{"[1]", "/tmp/draws.json"} {
			assert.Contains(t, detail, want, "the detail must survive translation")
		}
	})
}

// TestShellReportsInTheAppLanguage pins the input path: a dropped file is
// reported by the shell, not the menu, and it must use the same wording.
func TestShellReportsInTheAppLanguage(t *testing.T) {
	withLocale(t, "zh", func() {
		shell := NewShell(About{})
		var titles []string
		shell.report = func(_ mygo.MessageType, title, _, _ string) { titles = append(titles, title) }

		shell.OpenURL("https://example.com")
		require.Len(t, titles, 1)
		assert.Equal(t, "Aurora 无法打开该链接", titles[0])
	})
}

// TestUsageFollowsTheLocale covers --help: it is the first thing a user reads,
// and it was the one surface of the CLI (`aurora --help`) that had been
// localized long before the desktop existed.
func TestUsageFollowsTheLocale(t *testing.T) {
	withLocale(t, "zh", func() {
		var b strings.Builder
		Usage(&b, "v1", "t")
		out := b.String()
		for _, want := range []string{"用法：", "--data-dir", "版本"} {
			assert.Contains(t, out, want)
		}
		assert.NotContains(t, out, "Show this help and exit")
	})
}

// TestApplyConfiguredLocale pins the one path where the config file decides:
// with no LANG the CLI already follows [i18n] locale, and the window must
// agree instead of showing the system default.
func TestApplyConfiguredLocale(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() {
		tr.SetLocale(before)
		viper.Reset()
	})

	t.Setenv("LANG", "")
	viper.Set("i18n.locale", "zh")
	ApplyConfiguredLocale()
	assert.Equal(t, "zh", tr.GetLocale())

	// $LANG is authoritative (DetectAndInit's contract), so a configured
	// locale must not override it.
	tr.SetLocale("en")
	t.Setenv("LANG", "en_US.UTF-8")
	ApplyConfiguredLocale()
	assert.Equal(t, "en", tr.GetLocale())
}

// TestStartupDialogAndErrorsFollowTheLocale keeps the failure the user is
// most likely to hit (a bad --web-root) speaking the app's language.
func TestShowErrorUsesTheLocalizedReason(t *testing.T) {
	withLocale(t, "zh", func() {
		original := messageBox
		t.Cleanup(func() { messageBox = original })
		var got mygo.MessageOptions
		messageBox = func(opts mygo.MessageOptions) { got = opts }

		showError(nil, T("desktop.error.backup_failed"), errors.New("boom"))
		assert.Equal(t, "备份失败", got.Title)
		assert.Equal(t, "boom", got.Message)
	})
}

// TestStartupErrorsFollowTheLocale covers the reasons a start fails, which
// end up in the startup-failure dialog: a translated title over an English
// reason is still half a message, and the reason is the part the user has to
// act on.
func TestStartupErrorsFollowTheLocale(t *testing.T) {
	withLocale(t, "zh", func() {
		dir := t.TempDir()

		_, err := ResolveWebRoot(filepath.Join(dir, "missing"), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Web 目录")
		assert.Contains(t, err.Error(), "index.html", "the file name stays literal")

		_, err = ResolveWebRoot("", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "找不到 Web 界面")
	})
}

// TestFlagErrorsFollowTheLocale pins the first thing a mistaken launch prints.
func TestFlagErrorsFollowTheLocale(t *testing.T) {
	withLocale(t, "zh", func() {
		_, _, err := ParseFlags([]string{"extra"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "意外参数")
	})
}
