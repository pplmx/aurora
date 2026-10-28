package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Translator holds per-locale message tables.
//
// Concurrency: locale and messages are guarded by mu. Without this
// guard, concurrent callers (e.g. the API server running multiple
// HTTP handlers) could observe a partially-updated map during a
// locale switch or during LoadLocaleFile's bulk insert. The race
// detector trip was historically latent because callers were all
// single-goroutine; it became real when any HTTP handler reached
// across goroutines.
//
// The mutex is RWMutex because reads (T/TFormat) far outnumber
// writes (Init/SetLocale/LoadLocaleFile).
type Translator struct {
	mu       sync.RWMutex
	locale   string
	messages map[string]map[string]string
}

var (
	t       *Translator
	tInitMu sync.Mutex // protects the package-level t pointer itself
)

func Init(locale string) *Translator {
	tr := &Translator{
		locale:   locale,
		messages: make(map[string]map[string]string),
	}
	tr.loadMessages()
	tInitMu.Lock()
	t = tr
	tInitMu.Unlock()
	return tr
}

func GetTranslator() *Translator {
	tInitMu.Lock()
	defer tInitMu.Unlock()
	if t == nil {
		tInitMu.Unlock()
		// The lazy default must follow the environment locale, not hardcode
		// "en". Cobra command help texts (Short/Long/flag usage) are resolved
		// once at package init via GetText, before main() has a chance to call
		// DetectAndInit — locking the default to "en" here froze every --help
		// screen to English regardless of the locale environment
		// (TASK-128, ISS-123).
		t = Init(DetectLocale())
		tInitMu.Lock()
	}
	return t
}

func (tr *Translator) loadMessages() {
	// loadMessages is only called from Init on a fresh Translator,
	// before it is published to the package-level t. So no locking
	// is needed here. If this ever changes, callers must hold tr.mu.
	tr.messages["en"] = cloneMessages(englishMessages)
	tr.messages["zh"] = cloneMessages(chineseMessages)

	// Try to load from config
	tr.loadFromConfig()
}

// cloneMessages gives the translator its own map: LoadLocaleFile mutates the
// selected locale, and writing through to the package-level catalog would leak
// one process's overrides into every later Translator.
func cloneMessages(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func (tr *Translator) loadFromConfig() {
	// Get locale from config if not set. The real key is the [i18n] section's
	// "i18n.locale" (set in config/aurora.toml and setDefaultConfig); the old
	// top-level "locale" key was never written anywhere, so a configured
	// locale was silently ignored (TASK-200, ISS-196).
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.locale == "" {
		tr.locale = viper.GetString("i18n.locale")
	}
	if tr.locale == "" {
		tr.locale = "en"
	}
}

func (tr *Translator) T(key string) string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	if msg, ok := tr.messages[tr.locale][key]; ok {
		return msg
	}
	// Fallback to English
	if msg, ok := tr.messages["en"][key]; ok {
		return msg
	}
	return key
}

func (tr *Translator) TFormat(key string, args ...interface{}) string {
	return fmt.Sprintf(tr.T(key), args...)
}

func (tr *Translator) SetLocale(locale string) {
	tr.mu.Lock()
	tr.locale = locale
	tr.mu.Unlock()
}

func (tr *Translator) GetLocale() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.locale
}

func (tr *Translator) AvailableLocales() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	locales := make([]string, 0, len(tr.messages))
	for k := range tr.messages {
		locales = append(locales, k)
	}
	return locales
}

// EnvironmentLocale returns the locale selected by the process environment
// and whether any locale variable was set. LC_ALL wins, then LANGUAGE, then
// LC_MESSAGES, then LANG. LANGUAGE is a colon-separated preference list; the
// first tag Aurora ships wins. Unsupported locales fall back to English.
func EnvironmentLocale() (string, bool) {
	if value := strings.TrimSpace(os.Getenv("LC_ALL")); value != "" {
		return localeFromTags(value), true
	}
	if value := strings.TrimSpace(os.Getenv("LANGUAGE")); value != "" {
		return localeFromTags(value), true
	}
	if value := strings.TrimSpace(os.Getenv("LC_MESSAGES")); value != "" {
		return localeFromTags(value), true
	}
	if value := strings.TrimSpace(os.Getenv("LANG")); value != "" {
		return localeFromTags(value), true
	}
	return "", false
}

func localeFromTags(value string) string {
	for _, tag := range strings.Split(value, ":") {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		switch {
		case strings.HasPrefix(normalized, "zh"):
			return "zh"
		case strings.HasPrefix(normalized, "en"), normalized == "c", strings.HasPrefix(normalized, "c."),
			normalized == "posix", strings.HasPrefix(normalized, "posix."):
			return "en"
		}
	}
	return "en"
}

func DetectLocale() string {
	if locale, ok := EnvironmentLocale(); ok {
		return locale
	}
	return "en"
}

func GetText(key string) string {
	return GetTranslator().T(key)
}

func GetTextF(key string, args ...interface{}) string {
	return GetTranslator().TFormat(key, args...)
}

func DetectAndInit() *Translator {
	locale, configured := EnvironmentLocale()
	// The locale environment is authoritative (TASK-128/ISS-123 locked the
	// lazy default to it); a configured [i18n] locale is the fallback for
	// environments that declare no locale, so the config knob is no longer
	// inert (TASK-200, ISS-196). viper's defaults are installed during the
	// CLI's initConfig (after main), so on the CLI this only matters when the
	// config file is loaded first (tests / programmatic use) — the knob never
	// overrides an explicit locale environment.
	if !configured {
		if cfg := viper.GetString("i18n.locale"); cfg != "" {
			locale = cfg
		}
	}
	if locale == "" {
		locale = "en"
	}
	return Init(locale)
}

func LoadLocaleFile(path string) error {
	ext := filepath.Ext(path)
	locale := strings.TrimPrefix(ext, ".")

	viper.SetConfigType(ext[1:])
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	trans := GetTranslator()

	// Snapshot all settings under viper's own lock (viper is
	// goroutine-safe for reads but we still want a consistent
	// snapshot), then take the translator's write lock to install
	// the new keys.
	settings := viper.AllSettings()

	trans.mu.Lock()
	defer trans.mu.Unlock()
	if trans.messages[locale] == nil {
		trans.messages[locale] = make(map[string]string)
	}
	for key, value := range settings {
		if str, ok := value.(string); ok {
			trans.messages[locale][key] = str
		}
	}
	return nil
}
