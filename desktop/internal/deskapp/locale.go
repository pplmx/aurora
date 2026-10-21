package deskapp

import (
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/viper"
)

// The desktop shell has two translators to keep in step. internal/i18n holds
// Aurora's own strings (the CLI's, and now the menu's and the dialogs'), and
// MyGo holds the language the app shows, which it resolved from the user's
// languages and the language saved for the app. MyGo translates the menu
// items that carry a role ("File" as RoleFileMenu, "Quit", "Edit"); it cannot
// translate a label Aurora wrote itself. Without the sync the same menu mixes
// two languages: a zh-CN session showed MyGo's 文件/编辑/退出 beside Aurora's
// "Export Lotteries...", "Back Up Database..." and every dialog in English
// (ISS-291, measured under LANG=zh_CN).

// T reads one of Aurora's strings in the language the desktop currently
// shows. Call it where the string is used, not where a menu is built, so a
// language switch reaches dialogs opened afterwards.
func T(key string) string { return i18n.GetText(key) }

// TFormat is T for a key holding fmt verbs.
func TFormat(key string, args ...any) string { return i18n.GetTextF(key, args...) }

// auroraLocale maps a BCP 47 tag - what MyGo reports, e.g. "zh-Hans",
// "zh-Hant-TW", "en-GB" - onto one of the locales internal/i18n ships.
//
// Aurora is written in English and translated to Chinese only, so every
// Chinese variant reads the simplified table rather than English: a
// Traditional Chinese reader gets readable Chinese, which is closer to right
// than the language MyGo picked for its own menu roles.
func auroraLocale(tag string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(tag)), "zh") {
		return "zh"
	}
	return "en"
}

// InitLocale initializes the translator for the desktop process, the way
// cmd/aurora's main does. Without it the first T() call builds the lazy
// default from the environment and the configured [i18n] locale never
// applies to a window opened by a launcher that exports no LANG.
func InitLocale() { i18n.DetectAndInit() }

// mygoTag maps one of Aurora's locales onto a language MyGo can show. The
// matrix only has "en" and "zh", and MyGo cannot be handed "zh": it is not a
// normalized language on its own. Simplified Chinese is what internal/i18n
// ships, so that is the tag chosen in either direction.
func mygoTag(locale string) string {
	if auroraLocale(locale) == "zh" {
		return "zh-Hans"
	}
	return "en"
}

// ApplyConfiguredLocale makes MyGo show the language [i18n] locale asks for
// when the environment does not say (see DetectAndInit: $LANG wins, the
// config is the fallback). Without this the desktop window followed the
// system while the CLI of the same build followed the config, so
// `[i18n] locale = "zh"` translated `aurora lottery --help` and left the
// window English.
//
// It must run after config.Load and before Run. It is a no-op when LANG is
// set (MyGo already follows the same preference), and MYGO_LOCALE still wins
// for the run because MyGo ignores a choice made before it is ready while
// that variable is set.
func ApplyConfiguredLocale() {
	if os.Getenv("LANG") != "" {
		return
	}
	if cfg := strings.TrimSpace(viper.GetString("i18n.locale")); cfg != "" {
		// Both halves move: MyGo's own strings and roles, and Aurora's.
		mygo.App.SetLocale(mygoTag(cfg))
		syncLocale(cfg)
	}
}

// syncLocale points the translator at tag and reports whether that changed
// the language in use. The package translator is shared with the in-process
// API server, so the change is made in place rather than by re-initializing
// it (which would drop locale files loaded at run time).
func syncLocale(tag string) bool {
	want := auroraLocale(tag)
	tr := i18n.GetTranslator()
	if tr.GetLocale() == want {
		return false
	}
	tr.SetLocale(want)
	return true
}

// FollowAppLocale makes Aurora's own strings say the language MyGo shows, and
// every later one: it syncs once from App.Locale() and then on
// OnLocaleChanged, which is called on the main thread when the user switches
// language at run time.
//
// onChange runs after a change, and only then: MyGo relabels the items that
// carry a role itself, but a label Aurora wrote is baked into the menu when
// NewMenu builds it, so the caller rebuilds the menu. It must be called once
// the app is ready (App.Locale needs the running backend).
func FollowAppLocale(onChange func()) {
	apply := func(tag string) {
		if syncLocale(tag) && onChange != nil {
			onChange()
		}
	}
	apply(mygo.App.Locale())
	mygo.App.OnLocaleChanged(apply)
}
