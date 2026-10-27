package nativeui

import (
	"strconv"
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

		formCard(c, "Session keys", "Private keys can be kept in memory for this session only.", func() {
			if ui.Checkbox(c, &a.rememberKeys, tr("Remember private keys for this session")).Changed() && !a.rememberKeys {
				// Turning the policy off also clears what is already held: "off"
				// must mean no private key is sitting in memory.
				a.forgetSessionKeys()
			}
			hint(c, "Nothing is written to disk; quitting Aurora clears them.")
			readout(c, []readoutRow{{"Remembered keys", strconv.Itoa(a.rememberedKeyCount())}})
			if a.rememberedKeyCount() > 0 && secondaryButton(c, "Forget session keys").Clicked() {
				a.forgetSessionKeys()
			}
		})

		formCard(c, "Data", "Aurora keeps the ledger in one local SQLite database.", func() {
			copyRow(c, "Database path", a.actions.DBPath, nil)
			ui.Row(c).Wrap().Gap(8).Children(func() {
				a.settingsButton(c, "Show Data Folder", a.actions.ShowDataFolder)
				a.settingsButton(c, "Back Up Database…", a.actions.BackUpDatabase)
			})
			hint(c, "No data leaves this machine; the desktop opens no network listener.")
		})

		formCard(c, "Live updates", "Keep the open page's facts current on a timer.", func() {
			if ui.Checkbox(c, &a.liveUpdates, tr("Every 10 seconds")).Changed() {
				a.settingsStatus = statusLine{}
				a.setLiveUpdates(a.liveUpdates)
				if a.actions.SetLiveUpdates != nil {
					if err := a.actions.SetLiveUpdates(a.liveUpdates); err != nil {
						a.settingsStatus = statusLine{
							text:  tr("Couldn't save settings") + ": " + localizeServiceError(err),
							isErr: true,
						}
					}
				}
			}
			hint(c, "Applies to Dashboard, Oracle health and open readings; forms keep their text. Saved for the next launch.")
			// The only Settings action that can fail is this save; keep its
			// result beside the control instead of at the top of a scroll the
			// operator has already moved past.
			renderStatus(c, a.settingsStatus, func() { a.settingsStatus = statusLine{} })
		})

		formCard(c, "Shortcuts", "Window-wide keys; they work from any page.", func() {
			readout(c, []readoutRow{
				{"Pages", shortcutChord("1…8")},
				{"Reload", shortcutChord("R")},
				{"Settings", shortcutChord(",")},
				{"Command palette", shortcutChord("K")},
			})
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
// the language choices on this page.
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
