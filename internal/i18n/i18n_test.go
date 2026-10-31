package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslator_Init(t *testing.T) {
	tr := Init("en")

	require.NotNil(t, tr)
	require.Equal(t, "en", tr.locale)
}

func TestTranslator_NormalizesSupportedLocaleTags(t *testing.T) {
	for _, tt := range []struct {
		tag  string
		want string
	}{
		{tag: "en_US.UTF-8", want: "en"},
		{tag: "en-GB", want: "en"},
		{tag: "zh_CN.UTF-8", want: "zh"},
		{tag: "zh.UTF-8", want: "zh"},
		{tag: "zh-CN", want: "zh"},
		{tag: "zh-Hans", want: "zh"},
		{tag: "zh-Hant", want: "zh-Hant"},
		{tag: "zh-Hant-TW", want: "zh-Hant"},
		{tag: "zh-TW", want: "zh-Hant"},
		{tag: "zh_HK.UTF-8", want: "zh-Hant"},
	} {
		t.Run(tt.tag, func(t *testing.T) {
			tr := Init(tt.tag)
			assert.Equal(t, tt.want, tr.GetLocale())
		})
	}
}

func TestTranslator_SetLocaleNormalizesSupportedLocaleTags(t *testing.T) {
	tr := Init("en")
	tr.SetLocale("zh-Hant-TW")

	assert.Equal(t, "zh-Hant", tr.GetLocale())
	assert.Equal(t, "Aurora - 區塊鏈系統", tr.T("app.name"))
}

// TestTraditionalCatalogCoversEveryChineseKey: the Traditional table is
// generated from the Simplified one, so a key added to only one side would
// silently fall back to Simplified (or English) at run time. Equal key sets
// keep every string present in both scripts.
func TestTraditionalCatalogCoversEveryChineseKey(t *testing.T) {
	for key := range chineseMessages {
		require.Containsf(t, traditionalMessages, key, "%q has no Traditional entry", key)
	}
	assert.Len(t, traditionalMessages, len(chineseMessages),
		"the Traditional table must not carry keys the Simplified table lacks")
}

func TestTranslator_T(t *testing.T) {
	tr := Init("en")

	tests := []struct {
		key      string
		expected string
	}{
		{"app.name", "Aurora - Blockchain System"},
		{"app.version", "Version"},
		{"lottery.create", "Create a new lottery"},
		{"lottery.history", "Show lottery history"},
		{"lottery.participants", "Participant names (comma-separated)"},
		{"lottery.success", "Lottery created successfully!"},
		{"voting.candidate.add", "Add a candidate"},
		{"nft.mint", "Mint a new NFT"},
		{"oracle.fetch", "Fetch data from source"},
		{"error.invalid_input", "Invalid input"},
	}

	for _, tt := range tests {
		result := tr.T(tt.key)
		if result != tt.expected {
			t.Errorf("T(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestTranslator_T_Chinese(t *testing.T) {
	tr := Init("zh")

	tests := []struct {
		key      string
		expected string
	}{
		{"app.name", "Aurora - 区块链系统"},
		{"app.version", "版本"},
		{"lottery.create", "创建新抽奖"},
		{"voting.candidate.add", "添加候选人"},
		{"nft.mint", "铸造新 NFT"},
		{"oracle.fetch", "从数据源获取数据"},
		{"error.invalid_input", "输入无效"},
	}

	for _, tt := range tests {
		result := tr.T(tt.key)
		if result != tt.expected {
			t.Errorf("T(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestTranslator_T_MissingKey(t *testing.T) {
	tr := Init("en")

	result := tr.T("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("T(missing key) = %v, want key as fallback", result)
	}
}

func TestTranslator_HasDoesNotFallBackToEnglish(t *testing.T) {
	tr := Init("en")
	assert.True(t, tr.Has("app.version"))

	tr.SetLocale("fr")
	assert.False(t, tr.Has("app.version"))
	assert.Equal(t, "Version", tr.T("app.version"), "T keeps the English fallback")
}

func TestTranslator_TFormat(t *testing.T) {
	tr := Init("en")

	result := tr.TFormat("lottery.exported", 5, "test.json")
	expected := "Exported 5 lottery records to test.json"

	if result != expected {
		t.Errorf("TFormat = %v, want %v", result, expected)
	}
}

func TestTranslator_SetLocale(t *testing.T) {
	tr := Init("en")
	tr.SetLocale("zh")

	if tr.GetLocale() != "zh" {
		t.Errorf("GetLocale() = %v, want 'zh'", tr.GetLocale())
	}
}

func TestTranslator_AvailableLocales(t *testing.T) {
	tr := Init("en")

	locales := tr.AvailableLocales()

	if len(locales) == 0 {
		t.Error("AvailableLocales should not be empty")
	}
	found := false
	for _, l := range locales {
		if l == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'en' should be in available locales")
	}
}

func TestGetTranslator(t *testing.T) {
	tr := GetTranslator()

	if tr == nil {
		t.Fatal("GetTranslator returned nil")
	}
}

func TestDetectLocale(t *testing.T) {
	locale := DetectLocale()

	if locale != "en" && locale != "zh" {
		t.Errorf("DetectLocale = %v, want 'en' or 'zh'", locale)
	}
}

func clearLocaleEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		t.Setenv(key, "")
	}
}

// TestDetectLocale_UsesPOSIXPrecedence covers a shell that exports LANG=C but
// selects Chinese for messages through LC_ALL or LC_MESSAGES. Reading only
// LANG silently showed an English CLI and desktop help in that environment.
func TestDetectLocale_UsesPOSIXPrecedence(t *testing.T) {
	for _, tt := range []struct {
		name     string
		lcAll    string
		messages string
		lang     string
		language string
		want     string
	}{
		{name: "LC_ALL wins", lcAll: "zh_CN.UTF-8", messages: "en_US.UTF-8", lang: "en_US.UTF-8", want: "zh"},
		{name: "LC_MESSAGES before LANG", messages: "zh_TW.UTF-8", lang: "en_US.UTF-8", want: "zh-Hant"},
		{name: "LANGUAGE preference", language: "fr_FR:zh_CN", lang: "en_US.UTF-8", want: "zh"},
		{name: "LANGUAGE first supported", language: "en_US:zh_CN", lang: "zh_CN.UTF-8", want: "en"},
		{name: "LANG fallback", lang: "zh_CN.UTF-8", want: "zh"},
		{name: "C locale forces English", lcAll: "C", lang: "zh_CN.UTF-8", want: "en"},
		{name: "locale tags are case-insensitive", lang: "ZH_TW.UTF-8", want: "zh-Hant"},
		{name: "nothing configured", want: "en"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clearLocaleEnv(t)
			t.Setenv("LC_ALL", tt.lcAll)
			t.Setenv("LC_MESSAGES", tt.messages)
			t.Setenv("LANG", tt.lang)
			t.Setenv("LANGUAGE", tt.language)

			assert.Equal(t, tt.want, DetectLocale())
		})
	}
}

func TestEnvironmentLocaleReportsWhetherItWasConfigured(t *testing.T) {
	clearLocaleEnv(t)
	locale, configured := EnvironmentLocale()
	assert.False(t, configured)
	assert.Empty(t, locale)

	t.Setenv("LC_MESSAGES", "fr_FR.UTF-8")
	locale, configured = EnvironmentLocale()
	assert.True(t, configured)
	assert.Equal(t, "en", locale, "an unsupported environment locale falls back to English")
}

func TestDetectLocaleReadsDefaultConfigBeforeCobraLoadsIt(t *testing.T) {
	clearLocaleEnv(t)
	t.Setenv("HOME", t.TempDir())
	withWorkingDir(t, t.TempDir())
	require.NoError(t, os.MkdirAll("config", 0o755))
	require.NoError(t, os.WriteFile("config/aurora.toml", []byte("[i18n]\nlocale = \"zh-Hans\"\n"), 0o600))

	previousArgs := os.Args
	os.Args = []string{"aurora"}
	t.Cleanup(func() { os.Args = previousArgs })

	assert.Equal(t, "zh", DetectLocale())
}

func TestDetectLocaleReadsExplicitConfigFlag(t *testing.T) {
	clearLocaleEnv(t)
	t.Setenv("HOME", t.TempDir())
	configPath := filepath.Join(t.TempDir(), "custom.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[i18n]\nlocale = \"zh-CN\"\n"), 0o600))

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "separate value", args: []string{"aurora", "--config", configPath, "lottery", "--help"}},
		{name: "equals value", args: []string{"aurora", "--config=" + configPath, "lottery", "--help"}},
		{name: "last flag wins", args: []string{"aurora", "--config", "/missing.toml", "--config=" + configPath}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previousArgs := os.Args
			os.Args = tc.args
			t.Cleanup(func() { os.Args = previousArgs })

			assert.Equal(t, "zh", DetectLocale())
		})
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})
}

func TestGetText(t *testing.T) {
	result := GetText("app.name")

	if result == "" {
		t.Error("GetText should not return empty string")
	}
}

func TestGetTextF(t *testing.T) {
	result := GetTextF("lottery.exported", 10, "file.json")

	if result == "" {
		t.Error("GetTextF should not return empty string")
	}
}

// TestAlreadyRunningCopyDescribesTheDataDirectory keeps the second-launch
// dialog honest: this launch names the requested --data-dir, and the running
// window can only keep its current database. The old copy claimed the window
// kept a "Web UI", which Aurora has not shipped since the native rewrite.
func TestAlreadyRunningCopyDescribesTheDataDirectory(t *testing.T) {
	tr := Init("en")
	en := tr.TFormat("desktop.error.already_running_message", "--data-dir /tmp/other")
	assert.Contains(t, en, "asked for --data-dir /tmp/other")
	assert.Contains(t, en, "different data directory")
	assert.Contains(t, en, "quit Aurora first")
	assert.NotContains(t, strings.ToLower(en), "web")

	tr.SetLocale("zh")
	zh := tr.TFormat("desktop.error.already_running_message", "--data-dir /tmp/other")
	assert.Contains(t, zh, "本次启动指定了 --data-dir /tmp/other")
	assert.Contains(t, zh, "另一个数据目录")
	assert.Contains(t, zh, "请先退出 Aurora")
	assert.NotContains(t, strings.ToLower(zh), "web")
}

func TestDetectAndInit(t *testing.T) {
	tr := DetectAndInit()

	if tr == nil {
		t.Fatal("DetectAndInit returned nil")
	}
}

func TestDetectAndInitUsesConfigOnlyWithoutLocaleEnvironment(tt *testing.T) {
	clearLocaleEnv(tt)
	previous := GetTranslator()
	tt.Cleanup(func() {
		tInitMu.Lock()
		t = previous
		tInitMu.Unlock()
		viper.Reset()
	})

	viper.Set("i18n.locale", "zh")
	assert.Equal(tt, "zh", DetectAndInit().GetLocale(), "the config is the fallback")

	tt.Setenv("LC_MESSAGES", "en_US.UTF-8")
	assert.Equal(tt, "en", DetectAndInit().GetLocale(), "the environment must beat the config")
}

// TestTranslator_Concurrent_NoRace stresses the Translator's read/write
// surface from multiple goroutines. Without the mutex added in this
// round, this test trips -race within a few iterations.
func TestTranslator_Concurrent_NoRace(t *testing.T) {
	tr := Init("en")

	const readers = 8
	const writers = 4
	const iters = 200

	done := make(chan struct{})
	for i := 0; i < readers; i++ {
		go func() {
			for j := 0; j < iters; j++ {
				_ = tr.T("app.name")
				_ = tr.GetLocale()
				_ = tr.AvailableLocales()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < writers; i++ {
		go func(id int) {
			for j := 0; j < iters; j++ {
				if j%2 == 0 {
					tr.SetLocale("zh")
				} else {
					tr.SetLocale("en")
				}
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < readers+writers; i++ {
		<-done
	}
}

// forceLazyTranslatorForTest nil-outs the package translator, triggers the
// deferred-init path in GetTranslator, and returns the result plus a restore
// closure. The package var cannot be reached as `t` inside a test (that name
// is the *testing.T parameter), so the nil-out lives here.
func forceLazyTranslatorForTest() (*Translator, func()) {
	tInitMu.Lock()
	prev := t
	t = nil
	tInitMu.Unlock()

	tr := GetTranslator()

	restore := func() {
		tInitMu.Lock()
		t = prev
		tInitMu.Unlock()
	}
	return tr, restore
}

// TestGetTranslator_LazyInit covers the deferred-initialization branch of
// GetTranslator: when package t is nil, the first call must spin up an English
// translator instead of returning nil.
func TestGetTranslator_LazyInit(t *testing.T) {
	// The lazy default follows LANG by design (TASK-128/ISS-123), so this
	// test pins the environment it asserts instead of inheriting one.
	clearLocaleEnv(t)
	t.Setenv("LANG", "en_US.UTF-8")

	tr, restore := forceLazyTranslatorForTest()
	defer restore()

	require.NotNil(t, tr)
	require.Equal(t, "en", tr.GetLocale())
}

// TestTranslator_T_FallbackToEnglish covers the second lookup branch: a locale
// that has no message table falls back to the English table instead of
// returning the bare key.
func TestTranslator_T_FallbackToEnglish(t *testing.T) {
	tr := Init("en")
	tr.SetLocale("de")
	require.Equal(t, "Lottery created successfully!", tr.T("lottery.success"))
}

// TestDetectLocale_ChineseEnv covers the zh branch of DetectLocale (LANG
// starting with "zh").
func TestDetectLocale_ChineseEnv(t *testing.T) {
	clearLocaleEnv(t)
	t.Setenv("LANG", "zh_CN.UTF-8")
	require.Equal(t, "zh", DetectLocale())
}

// TestLoadLocaleFile_LoadsKeys covers the happy path of LoadLocaleFile: keys
// from a locale config file land in a per-locale message table and are
// reachable via T once the locale is selected.
func TestLoadLocaleFile_LoadsKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "zh.toml")
	require.NoError(t, os.WriteFile(p, []byte("greeting = \"hello locale\"\n"), 0644))

	tr := Init("en")
	require.NoError(t, LoadLocaleFile(p))
	tr.SetLocale("toml")
	require.Equal(t, "hello locale", tr.T("greeting"))
}

// TestLoadLocaleFile_MissingFile covers the error path: an unreadable/missing
// locale file surfaces the viper error.
func TestLoadLocaleFile_MissingFile(t *testing.T) {
	Init("en")
	err := LoadLocaleFile(filepath.Join(t.TempDir(), "nope.toml"))
	require.Error(t, err)
}

// TestLazyDefaultFollowsLocale pins the TASK-128/ISS-123 fix: the lazy
// translator created on first GetText must adopt the environment locale, not
// lock to "en". Cobra command help texts are resolved once at package init
// (before main calls DetectAndInit), so locking the default to "en" froze
// every --help screen to English regardless of the locale environment.
func TestLazyDefaultFollowsLocale(tt *testing.T) {
	clearLocaleEnv(tt)
	tt.Setenv("LANG", "zh_CN.UTF-8")

	// Reset the package translator so the next GetTranslator call re-creates
	// the lazy default with the (simulated) environment. The test parameter
	// is named tt so the package-level translator var `t` is not shadowed.
	tInitMu.Lock()
	t = nil
	tInitMu.Unlock()

	got := GetTranslator().GetLocale()
	require.Equal(tt, "zh", got, "lazy default must follow LANG=zh for package-init help texts")
}

// TestLocalesHaveMatchingKeys pins the en↔zh parity contract: every key
// present in one shipped locale must exist in the other. A missing zh key
// silently falls back to English (GetText's ok-check), which is exactly how
// hardcoded English leaks into zh-locale sessions one key at a time (TASK-130
// family; the help screen gained tui.help.scroll without a zh twin would have
// regressed this).
func TestLocalesHaveMatchingKeys(t *testing.T) {
	tr := GetTranslator()
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	en := tr.messages["en"]
	zh := tr.messages["zh"]
	require.NotEmpty(t, en, "en locale must be shipped")
	require.NotEmpty(t, zh, "zh locale must be shipped")

	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Errorf("zh is missing key %q (present in en)", key)
		}
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("en is missing key %q (present in zh)", key)
		}
	}
}

// TestCatalogsCoverTheSameKeys keeps the two shipped catalogs in step. A key
// added to English but not Chinese (or the reverse) renders as the key itself
// for every user of that language, which the fallback hides on a developer
// machine running English.
func TestCatalogsCoverTheSameKeys(t *testing.T) {
	for key := range englishMessages {
		_, ok := chineseMessages[key]
		assert.True(t, ok, "english has %q but chinese does not", key)
	}
	for key := range chineseMessages {
		_, ok := englishMessages[key]
		assert.True(t, ok, "chinese has %q but english does not", key)
	}
	assert.NotEmpty(t, englishMessages)
}

// TestDesktopCopyStaysOutOfTheCoreCatalog keeps the layer boundary visible:
// the native labels live in desktop/internal/nativeui/messages_zh.go, not in
// the CLI/service tables.
func TestDesktopCopyStaysOutOfTheCoreCatalog(t *testing.T) {
	for _, key := range []string{"Dashboard", "Quick start", "Use saved key", "Live updates"} {
		_, en := englishMessages[key]
		_, zh := chineseMessages[key]
		assert.Falsef(t, en || zh, "desktop key %q leaked into the core catalog", key)
	}
}

// TestEveryCoreKeyIsUsed keeps the catalogue from growing dead weight: a key no
// caller reads is translated text nobody can ever see. Three families (doctor
// checks, voting statuses, NFT operations) build their key from a runtime value,
// so they are exempt by prefix rather than by listing every suffix.
func TestEveryCoreKeyIsUsed(t *testing.T) {
	root := filepath.Join("..", "..")
	dynamicPrefixes := []string{"doctor.check.", "voting.status.", "nft.operation."}

	var source strings.Builder
	for _, dir := range []string{"cmd", "internal", "tools", "desktop"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasPrefix(filepath.Base(path), "messages_") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			source.Write(data)
			source.WriteString("\n")
			return nil
		})
		require.NoError(t, err)
	}
	text := source.String()
	require.NotEmpty(t, text)

	unused := make([]string, 0)
	for key := range englishMessages {
		if strings.Contains(text, key) {
			continue
		}
		exempt := false
		for _, prefix := range dynamicPrefixes {
			if strings.HasPrefix(key, prefix) {
				exempt = true
				break
			}
		}
		if !exempt {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)
	assert.Emptyf(t, unused, "these keys are never read: %v", unused)
}

// formatVerb matches one fmt verb: flags, width, precision, an optional
// argument index and the verb letter.
var formatVerb = regexp.MustCompile(`%[-+# 0-9.]*(?:\[[0-9]+\])?[a-zA-Z%]`)

// formatVerbs returns the verbs that consume an argument (%% does not).
func formatVerbs(s string) []string {
	var verbs []string
	for _, verb := range formatVerb.FindAllString(s, -1) {
		if verb != "%%" {
			verbs = append(verbs, verb)
		}
	}
	return verbs
}

// TestChineseCatalogKeepsThePlaceholders: a translation has to carry the same
// number of format verbs as its English source, or the CLI prints
// "%!s(MISSING)" (or silently drops a value) where the sentence expects one.
func TestChineseCatalogKeepsThePlaceholders(t *testing.T) {
	for key, english := range englishMessages {
		chinese, ok := chineseMessages[key]
		require.Truef(t, ok, "%s has no Chinese entry", key)
		assert.Equalf(t, len(formatVerbs(english)), len(formatVerbs(chinese)),
			"%s keeps a different number of placeholders\n  en: %q\n  zh: %q", key, english, chinese)
	}
}
