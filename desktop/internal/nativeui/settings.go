package nativeui

import (
	"fmt"
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
		// Two columns on a normal window: a preferences page is a set of short
		// independent cards, and one full-width column left most of the window
		// empty while the page scrolled for no reason. The split stacks below
		// the same width threshold every other two-column page uses.
		twoColumns(c,
			func() {
				a.settingsLanguageCard(c)
				a.settingsDataCard(c)
				a.settingsShortcutsCard(c)
			},
			func() {
				a.settingsSessionKeysCard(c)
				a.settingsLiveUpdatesCard(c)
				a.settingsAppearanceCard(c)
				a.settingsAboutCard(c)
			},
		)
	})
	a.renderForgetKeysDialog(c)
}

// settingsLanguageCard is the display language: the language in effect, the
// installed choices, and the separate policy of following the system.
func (a *App) settingsLanguageCard(c *ui.Context) {
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
}

// settingsDataCard names where the ledger lives and the two file operations a
// reader can reach from here, so the privacy statement sits beside the path it
// describes.
func (a *App) settingsDataCard(c *ui.Context) {
	formCard(c, "Data", "Aurora keeps the ledger in one local SQLite database.", func() {
		copyRow(c, "Database path", a.actions.DBPath, nil)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			const privacy = "No data leaves this machine; the desktop opens no network listener."
			a.settingsButton(c, "Show Data Folder", privacy, a.actions.ShowDataFolder)
			a.settingsButton(c, "Back Up Database…", privacy, a.actions.BackUpDatabase)
		})
		hint(c, "No data leaves this machine; the desktop opens no network listener.")
	})
}

// settingsShortcutsCard is the window's reference for the chords the header
// and sidebar register. Keys are listed once, on the page a reader visits to
// learn them.
func (a *App) settingsShortcutsCard(c *ui.Context) {
	formCard(c, "Shortcuts", "Navigation and window commands; Reload applies to pages that read stored data.", func() {
		ui.Column(c).FillWidth().Gap(space1).Children(func() {
			for i, shortcut := range []struct {
				label string
				chord string
			}{
				{"Pages", shortcutChord("1…8")},
				{"Reload (data pages)", shortcutChord("R")},
				{"Back up database", shortcutChord("B")},
				{"Export lotteries", shortcutChord("E")},
				{"Import lotteries", shortcutChord("I")},
				{"Settings", shortcutChord(",")},
				{"Command palette", shortcutChord("K")},
			} {
				if i > 0 {
					ui.Row(c).FillWidth().Height(1).Background(c.Theme().Border)
				}
				ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).
					Padding(space1+1, 1).Children(func() {
					ui.Text(c, tr(shortcut.label)).FontSize(textSize(c, 12)).
						TextColor(c.Theme().TextMuted).Grow(1).Shrink(1).MinWidth(0).SingleLine()
					shortcutCap(c, shortcut.chord)
				})
			}
		})
	})
}

// settingsSessionKeysCard controls the session-only key memory. Turning the
// policy off asks first when keys are still held, because those keys may exist
// nowhere else.
func (a *App) settingsSessionKeysCard(c *ui.Context) {
	formCard(c, "Session keys", "Private keys can be kept in memory for this session only.", func() {
		remember := a.rememberKeys
		checkbox := ui.Checkbox(c, &remember, tr("Remember private keys for this session")).
			Description(tr("Nothing is written to disk; quitting Aurora clears them.")).
			Bind(&a.rememberKeysControl)
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
					Description(tr("Remove every private key kept in memory for this session.")).
					Bind(&a.forgetKeysButton)
				if forget.Clicked() {
					a.confirmForgetKeys = true
					a.disableKeyMemoryOnForget = false
				}
			})
		}
	})
}

// settingsLiveUpdatesCard is the opt-in timer. Its failure line belongs in the
// card, beside the control that caused it.
func (a *App) settingsLiveUpdatesCard(c *ui.Context) {
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
		if renderStatus(c, pageSettings, a.settingsStatus, func() { a.settingsStatus = statusLine{} }) {
			a.navigationControl.Focus(c)
		}
	})
}

// settingsAppearanceCard lets the operator choose the app's own appearance,
// then reports the OS accessibility settings that modify it.
func (a *App) settingsAppearanceCard(c *ui.Context) {
	formCard(c, "Appearance", "Choose the window palette; text size, contrast and motion follow the operating system.", func() {
		mode := normalizeThemeMode(a.themeMode)
		ui.RadioGroup(c, func() {
			ui.Radio(c, &mode, ThemeModeSystem, tr("Follow System"))
			ui.Radio(c, &mode, ThemeModeLight, tr("Light"))
			ui.Radio(c, &mode, ThemeModeDark, tr("Dark"))
		}).Row().Wrap().Gap(space4).Label(tr("Theme mode"))
		if mode != a.themeMode {
			a.settingsStatus = statusLine{}
			a.themeMode = mode
			if a.actions.SetThemeMode != nil {
				if err := a.actions.SetThemeMode(mode); err != nil {
					a.settingsStatus = errorStatusText(
						tr("Couldn't save settings") + ": " + localizeServiceError(err))
				}
			}
		}
		preferences := c.Preferences()
		readout(c, []readoutRow{
			{"Theme", themeModeLabel(mode)},
			{"Text size", textScaleLabel(preferences.TextScale)},
			{"Contrast", preferenceStateLabel(preferences.HighContrast, "Enhanced", "Standard")},
			{"Motion", preferenceStateLabel(preferences.ReduceMotion, "Reduced", "Enabled")},
		})
		hint(c, "Follow System uses the desktop appearance; Light and Dark keep Aurora's palette regardless of it.")
		if renderStatus(c, pageSettings, a.settingsStatus, func() { a.settingsStatus = statusLine{} }) {
			a.navigationControl.Focus(c)
		}
	})
}

func themeModeLabel(mode ThemeMode) string {
	switch normalizeThemeMode(mode) {
	case ThemeModeSystem:
		return tr("Follow System")
	case ThemeModeLight:
		return tr("Light")
	default:
		return tr("Dark")
	}
}

// settingsAboutCard is the build identity and the one dialog that has no other
// home on this page.
func (a *App) settingsAboutCard(c *ui.Context) {
	formCard(c, "About Aurora", "", func() {
		rows := []readoutRow{
			{"Version", valueOr(a.actions.Version)},
			{"Build time", valueOr(a.actions.BuildTime)},
			{"Database", valueOr(a.actions.DBPath)},
		}
		// The size is measured while the card draws, so it matches the file the
		// path names; a window built without the callback simply omits it.
		if a.actions.DatabaseSize != nil {
			if size := a.actions.DatabaseSize(); size != "" {
				rows = append(rows, readoutRow{"Database size", size})
			}
		}
		readout(c, rows)
		// Support asks for the build facts; copying the card's own rows keeps the
		// reply exact instead of a typed paraphrase.
		if quietButton(c, tr("Copy")+" "+tr("Diagnostics")).Clicked() {
			a.copyDiagnostics(c)
		}
		a.settingsButton(c, "Open About Dialog", "", a.actions.ShowAbout)
	})
}

// diagnosticsText builds the About card's copied text: the build facts, the
// database and the ledger, in the language the window shows. The palette calls
// it too, so both paths copy the same words.
func (a *App) diagnosticsText() string {
	rows := []readoutRow{
		{"Version", valueOr(a.actions.Version)},
		{"Build time", valueOr(a.actions.BuildTime)},
		{"Database", valueOr(a.actions.DBPath)},
	}
	if a.actions.DatabaseSize != nil {
		if size := a.actions.DatabaseSize(); size != "" {
			rows = append(rows, readoutRow{"Database size", size})
		}
	}
	lines := make([]string, 0, len(rows)+1)
	for _, row := range rows {
		lines = append(lines, tr(row.label)+": "+row.value)
	}
	if a.actions.LedgerFacts != nil {
		if blocks, head, valid, ok := a.actions.LedgerFacts(); ok {
			verdict := tr("chain valid")
			if !valid {
				verdict = tr("chain broken")
			}
			lines = append(lines, fmt.Sprintf(tr("Ledger: %d blocks, head %s — %s"), blocks, head, verdict))
		}
	}
	if a.actions.OutboxPending != nil {
		if pending, ok := a.actions.OutboxPending(); ok {
			lines = append(lines, fmt.Sprintf(tr("Audit outbox: %d pending events"), pending))
		}
	}
	return strings.Join(lines, "\n")
}

// copyDiagnostics puts diagnosticsText on the clipboard and confirms where the
// pointer is.
func (a *App) copyDiagnostics(c *ui.Context) {
	c.Services().WriteClipboard(a.diagnosticsText())
	c.Toast(tr("Diagnostics are on the clipboard"))
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
	cancelFocus := a.forgetKeysButton
	if disableMemory {
		cancelFocus = a.rememberKeysControl
	}
	result := destructiveConfirm(c, &a.confirmForgetKeys, title,
		tr("All private keys kept only in this session will be removed. If they are not saved elsewhere, they cannot be recovered."),
		tr("Forget keys"), cancelFocus, a.rememberKeysControl)
	if !a.confirmForgetKeys {
		a.disableKeyMemoryOnForget = false
	}
	if result != destructiveConfirmed {
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
