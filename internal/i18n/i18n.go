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
		locale:   canonicalLocale(locale),
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
	tr.messages["zh-Hant"] = cloneMessages(traditionalMessages)

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
	tr.locale = canonicalLocale(tr.locale)
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

// Has reports whether the active locale defines key. Unlike T it does not
// fall back to English, so callers can distinguish a translated value from a
// key that merely exists in the source catalog.
func (tr *Translator) Has(key string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	_, ok := tr.messages[tr.locale][key]
	return ok
}

func (tr *Translator) TFormat(key string, args ...interface{}) string {
	return fmt.Sprintf(tr.T(key), args...)
}

func (tr *Translator) SetLocale(locale string) {
	tr.mu.Lock()
	tr.locale = canonicalLocale(locale)
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
		if locale, ok := localeForTag(tag); ok {
			return locale
		}
	}
	return "en"
}

// localeForTag maps the locale tags Aurora ships onto its three message tables.
// Locale tags can arrive from the environment (en_US.UTF-8, zh-CN) or from
// [i18n] locale (where BCP 47 tags are a natural spelling). A Traditional tag
// selects the zh-Hant catalog; every other Chinese spelling keeps Simplified.
func localeForTag(tag string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	switch {
	case isTraditionalTag(normalized):
		return "zh-Hant", true
	case normalized == "zh" || strings.HasPrefix(normalized, "zh-"), strings.HasPrefix(normalized, "zh_"),
		strings.HasPrefix(normalized, "zh."):
		return "zh", true
	case normalized == "en" || strings.HasPrefix(normalized, "en-"), strings.HasPrefix(normalized, "en_"),
		strings.HasPrefix(normalized, "en."):
		return "en", true
	case normalized == "c" || strings.HasPrefix(normalized, "c."),
		normalized == "posix" || strings.HasPrefix(normalized, "posix."):
		return "en", true
	default:
		return "", false
	}
}

// isTraditionalTag recognizes the spellings that ask for Traditional Chinese:
// zh-Hant (BCP 47), zh-TW, zh-HK and zh-MO, each with an optional suffix such
// as ".UTF-8".
func isTraditionalTag(normalized string) bool {
	normalized = strings.ReplaceAll(normalized, "_", "-")
	for _, prefix := range []string{"zh-hant", "zh-tw", "zh-hk", "zh-mo"} {
		if normalized == prefix ||
			strings.HasPrefix(normalized, prefix+"-") ||
			strings.HasPrefix(normalized, prefix+".") {
			return true
		}
	}
	return false
}

// isChineseLocale reports whether locale reads from a Chinese catalog. Both
// Chinese tables are Chinese, so locale-sensitive wording applies to either.
func isChineseLocale(locale string) bool {
	return locale == "zh" || locale == "zh-Hant"
}

// canonicalLocale keeps an unknown locale available for explicit locale-file
// loading/tests while using the same table for every supported spelling.
func canonicalLocale(locale string) string {
	if canonical, ok := localeForTag(locale); ok {
		return canonical
	}
	return strings.TrimSpace(locale)
}

func DetectLocale() string {
	if locale, ok := EnvironmentLocale(); ok {
		return locale
	}
	// Command declarations call GetText during package initialization, before
	// Cobra runs initConfig. Read the same startup config here so a configured
	// [i18n] locale also reaches --help and flag descriptions, not just
	// command output produced after Cobra has loaded the file.
	if locale := configFileLocale(); locale != "" {
		return canonicalLocale(locale)
	}
	return "en"
}

func GetText(key string) string {
	return GetTranslator().T(key)
}

func GetTextF(key string, args ...interface{}) string {
	return GetTranslator().TFormat(key, args...)
}

func HasText(key string) bool {
	return GetTranslator().Has(key)
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
		} else {
			// The lazy translator and command metadata use the same config
			// fallback during package initialization. Keep the post-init
			// translator in agreement with them, including when Cobra has
			// not loaded the config yet (for example, --help).
			locale = DetectLocale()
		}
	}
	if locale == "" {
		locale = "en"
	}
	return Init(locale)
}

// configFileLocale performs the small, read-only config lookup needed before
// Cobra owns the full config lifecycle. An explicit --config path wins; with
// no flag, the documented $HOME/aurora.toml and ./config/aurora.toml search
// paths are checked in that order. Parse failures are intentionally ignored
// here: command execution will report the actionable config error later,
// while --help can still render in a safe fallback language.
func configFileLocale() string {
	v := viper.New()
	v.SetConfigType("toml")
	if path := configFileArg(os.Args[1:]); path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("aurora")
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			v.AddConfigPath(home)
		}
		v.AddConfigPath("./config")
	}
	if err := v.ReadInConfig(); err != nil {
		return ""
	}
	return strings.TrimSpace(v.GetString("i18n.locale"))
}

func configFileArg(args []string) string {
	path := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return path
		case arg == "--config" && i+1 < len(args):
			path = args[i+1]
			i++
		case strings.HasPrefix(arg, "--config="):
			path = strings.TrimPrefix(arg, "--config=")
		}
	}
	return path
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
