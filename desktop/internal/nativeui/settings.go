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
			language := ""
			if a.actions.Language != nil {
				language = a.actions.Language()
				current = languageLabel(language)
			}
			readout(c, []readoutRow{{"Current", current}})
			if a.actions.SetLanguage != nil {
				ui.Row(c).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
					languageChooser(c, languageChoice(language), a.actions.SetLanguage)
					follow := secondaryButton(c, "Follow System").
						Description(tr("Use the language selected by the operating system."))
					if follow.Clicked() {
						a.actions.SetLanguage("")
					}
				})
			}
		})

		formCard(c, "Session keys", "Private keys can be kept in memory for this session only.", func() {
			remember := a.rememberKeys
			checkbox := ui.Checkbox(c, &remember, tr("Remember private keys for this session")).
				Description(tr("Nothing is written to disk; quitting Aurora clears them."))
			if checkbox.Changed() {
				switch {
				case remember:
					a.rememberKeys = true
				case a.rememberedKeyCount() > 0:
					// Turning the policy off also clears what is already held:
					// "off" must mean no private key is sitting in memory. Ask
					// before discarding keys that may exist only in this process.
					a.confirmForgetKeys = true
					a.disableKeyMemoryOnForget = true
				default:
					a.rememberKeys = false
				}
			}
			hint(c, "Nothing is written to disk; quitting Aurora clears them.")
			readout(c, []readoutRow{{"Remembered keys", strconv.Itoa(a.rememberedKeyCount())}})
			if a.rememberedKeyCount() > 0 {
				ui.Row(c).Children(func() {
					forget := dangerButton(c, "Forget session keys").
						Description(tr("Remove every private key kept in memory for this session."))
					if forget.Clicked() {
						a.confirmForgetKeys = true
						a.disableKeyMemoryOnForget = false
					}
				})
			}
		})

		formCard(c, "Data", "Aurora keeps the ledger in one local SQLite database.", func() {
			copyRow(c, "Database path", a.actions.DBPath, nil)
			ui.Row(c).Wrap().Gap(8).Children(func() {
				const privacy = "No data leaves this machine; the desktop opens no network listener."
				a.settingsButton(c, "Show Data Folder", privacy, a.actions.ShowDataFolder)
				a.settingsButton(c, "Back Up Database…", privacy, a.actions.BackUpDatabase)
			})
			hint(c, "No data leaves this machine; the desktop opens no network listener.")
		})

		formCard(c, "Live updates", "Keep the open page's facts current on a timer.", func() {
			if ui.Checkbox(c, &a.liveUpdates, tr("Every 10 seconds")).
				Description(tr("Applies to Dashboard, Oracle health and open readings; forms keep their text. Saved for the next launch.")).Changed() {
				a.settingsStatus = statusLine{}
				a.setLiveUpdates(a.liveUpdates)
				if a.actions.SetLiveUpdates != nil {
					if err := a.actions.SetLiveUpdates(a.liveUpdates); err != nil {
						a.settingsStatus = errorStatusText(
							tr("Couldn't save settings") + ": " + localizeServiceError(err))
					}
				}
			}
			hint(c, "Applies to Dashboard, Oracle health and open readings; forms keep their text. Saved for the next launch.")
			// The only Settings action that can fail is this save; keep its
			// result beside the control instead of at the top of a scroll the
			// operator has already moved past.
			renderStatus(c, pageSettings, a.settingsStatus, func() { a.settingsStatus = statusLine{} })
		})

		formCard(c, "Shortcuts", "Navigation and window commands; Reload applies to pages that read stored data.", func() {
			readout(c, []readoutRow{
				{"Pages", shortcutChord("1…8")},
				{"Reload (data pages)", shortcutChord("R")},
				{"Settings", shortcutChord(",")},
				{"Command palette", shortcutChord("K")},
			})
		})

		formCard(c, "Appearance", "Aurora is dark by design; text size, contrast and motion follow the operating system.", func() {
			preferences := c.Preferences()
			readout(c, []readoutRow{
				{"Theme", tr("Dark")},
				{"Text size", textScaleLabel(preferences.TextScale)},
				{"Contrast", preferenceStateLabel(preferences.HighContrast, "Enhanced", "Standard")},
				{"Motion", preferenceStateLabel(preferences.ReduceMotion, "Reduced", "Enabled")},
			})
		})

		formCard(c, "About Aurora", "", func() {
			readout(c, []readoutRow{
				{"Version", valueOr(a.actions.Version)},
				{"Build time", valueOr(a.actions.BuildTime)},
				{"Database", valueOr(a.actions.DBPath)},
			})
			a.settingsButton(c, "Open About Dialog", "", a.actions.ShowAbout)
		})
	})
	a.renderForgetKeysDialog(c)
}

// renderForgetKeysDialog asks before discarding keys that may not exist
// anywhere else. The shared dialog starts on Cancel, so Enter cannot erase
// them; one Tab reaches the red Forget action.
func (a *App) renderForgetKeysDialog(c *ui.Context) {
	if !a.confirmForgetKeys {
		return
	}
	disableMemory := a.disableKeyMemoryOnForget
	title := tr("Forget all session keys?")
	if disableMemory {
		title = tr("Turn off session key memory?")
	}
	confirmed := destructiveConfirm(c, &a.confirmForgetKeys, title,
		tr("All private keys kept only in this session will be removed. If they are not saved elsewhere, they cannot be recovered."),
		tr("Forget keys"))
	if !a.confirmForgetKeys {
		a.disableKeyMemoryOnForget = false
	}
	if !confirmed {
		return
	}
	a.forgetSessionKeys()
	if disableMemory {
		a.rememberKeys = false
	}
}

// languageChooser exposes the installed languages as one keyboard-operable
// radio group. Follow System stays a separate action because it changes the
// selection policy, not which installed language is currently shown.
func languageChooser(c *ui.Context, current string, set func(string)) {
	if set == nil {
		return
	}
	choice := current
	ui.RadioGroup(c, func() {
		for _, option := range []struct {
			label string
			tag   string
		}{
			{"English", "en"},
			{"简体中文", "zh-Hans"},
			{"繁體中文", "zh-Hant"},
		} {
			ui.Radio(c, &choice, option.tag, option.label)
		}
	}).Row().Wrap().Gap(16).Label(tr("Display language"))
	if choice != current {
		set(choice)
	}
}

func (a *App) settingsButton(c *ui.Context, label, description string, fn func()) {
	if fn == nil {
		return
	}
	button := secondaryButton(c, label)
	if description != "" {
		button.Description(tr(description))
	}
	if button.Clicked() {
		fn()
	}
}

// languageLabel renders the language in effect in its own language, matching
// the language choices on this page.
func languageLabel(tag string) string {
	switch languageChoice(tag) {
	case "en":
		return "English"
	case "zh-Hans":
		return "简体中文"
	case "zh-Hant":
		return "繁體中文"
	case "":
		if strings.TrimSpace(tag) == "" {
			return "—"
		}
		return tag
	default:
		return tag
	}
}

func languageChoice(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	switch {
	case strings.HasPrefix(tag, "en"):
		return "en"
	case strings.HasPrefix(tag, "zh-hant"),
		strings.HasPrefix(tag, "zh-tw"),
		strings.HasPrefix(tag, "zh-hk"):
		return "zh-Hant"
	case strings.HasPrefix(tag, "zh"):
		return "zh-Hans"
	default:
		return ""
	}
}

func textScaleLabel(scale float32) string {
	if scale <= 0 {
		scale = 1
	}
	return strconv.FormatFloat(float64(scale*100), 'f', 0, 32) + "%"
}

func preferenceStateLabel(active bool, activeLabel, inactiveLabel string) string {
	if active {
		return tr(activeLabel)
	}
	return tr(inactiveLabel)
}
