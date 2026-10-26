package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// settingsView is the application's own configuration: language, data
// location, appearance behavior and build identity. File operations stay in
// the sidebar's File menu; this page is where a setting or a read-only fact
// about the running app belongs.
func (a *App) settingsView(c *ui.Context) {
	a.scrollPage(c, func() {
		formCard(c, "Language", "Applies to menus, pages, dialogs and guidance.", func() {
			current := "—"
			if a.actions.Language != nil {
				current = languageLabel(a.actions.Language())
			}
			readout(c, []readoutRow{{"Current", current}})
			ui.Row(c).Wrap().Gap(8).Children(func() {
				a.languageButton(c, "Follow System", "")
				a.languageButton(c, "English", "en")
				a.languageButton(c, "简体中文", "zh-Hans")
				a.languageButton(c, "繁體中文", "zh-Hant")
			})
		})

		formCard(c, "Data", "Aurora keeps the ledger in one local SQLite database.", func() {
			copyRow(c, "Database path", a.actions.DBPath, nil)
			ui.Row(c).Wrap().Gap(8).Children(func() {
				a.settingsButton(c, "Show Data Folder", a.actions.ShowDataFolder)
				a.settingsButton(c, "Back Up Database…", a.actions.BackUpDatabase)
			})
			hint(c, "No data leaves this machine; the desktop opens no network listener.")
		})

		formCard(c, "Appearance", "Aurora is dark by design; the operating system still decides size and contrast.", func() {
			readout(c, []readoutRow{
				{"Theme", "Dark"},
				{"Text size", "Follows system"},
				{"Contrast", "Follows system"},
				{"Motion", "Follows system"},
			})
		})

		formCard(c, "About Aurora", "", func() {
			readout(c, []readoutRow{
				{"Version", valueOr(a.actions.Version)},
				{"Build time", valueOr(a.actions.BuildTime)},
				{"Database", valueOr(a.actions.DBPath)},
			})
			a.settingsButton(c, "Open About Dialog", a.actions.ShowAbout)
		})
	})
}

func (a *App) languageButton(c *ui.Context, label, tag string) {
	if a.actions.SetLanguage == nil {
		return
	}
	if secondaryButton(c, label).Clicked() {
		a.actions.SetLanguage(tag)
	}
}

func (a *App) settingsButton(c *ui.Context, label string, fn func()) {
	if fn == nil {
		return
	}
	if secondaryButton(c, label).Clicked() {
		fn()
	}
}

// languageLabel renders the language in effect in its own language, matching
// the Actions language choices.
func languageLabel(tag string) string {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "en":
		return "English"
	case "zh-hans":
		return "简体中文"
	case "zh-hant":
		return "繁體中文"
	case "":
		return "—"
	default:
		return tag
	}
}
