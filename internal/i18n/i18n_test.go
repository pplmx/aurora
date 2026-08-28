package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranslator_Init(t *testing.T) {
	tr := Init("en")

	require.NotNil(t, tr)
	require.Equal(t, "en", tr.locale)
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

func TestDetectAndInit(t *testing.T) {
	tr := DetectAndInit()

	if tr == nil {
		t.Fatal("DetectAndInit returned nil")
	}
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
// every --help screen to English regardless of LANG.
func TestLazyDefaultFollowsLocale(tt *testing.T) {
	prevLang, hadLang := os.LookupEnv("LANG")
	tt.Cleanup(func() {
		if hadLang {
			_ = os.Setenv("LANG", prevLang)
		} else {
			_ = os.Unsetenv("LANG")
		}
	})

	// Reset the package translator so the next GetTranslator call re-creates
	// the lazy default with the (simulated) environment. The test parameter
	// is named tt so the package-level translator var `t` is not shadowed.
	tInitMu.Lock()
	t = nil
	tInitMu.Unlock()

	require.NoError(tt, os.Setenv("LANG", "zh_CN.UTF-8"))
	got := GetTranslator().GetLocale()
	require.Equal(tt, "zh", got, "lazy default must follow LANG=zh for package-init help texts")
}
