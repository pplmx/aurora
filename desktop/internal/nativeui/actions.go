package nativeui

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"

	"github.com/pplmx/aurora/internal/i18n"
)

// Actions is the application work a page cannot do itself: file operations,
// the About dialog and the language choice. The shell owns those because they
// need the database path and native dialogs, so it hands them to the window as
// callbacks instead of the view importing the shell.
//
// Every field may be nil (tests, or a window built before the shell is ready):
// the menu disables what it cannot do rather than showing a dead item.
type Actions struct {
	BackUpDatabase  func()
	ExportLotteries func()
	ImportLotteries func()
	ShowDataFolder  func()
	ShowAbout       func()
	// RecentDocuments lists the lottery exports the user opened or saved;
	// OpenRecent loads one through the same import path as a drop.
	RecentDocuments func() []string
	OpenRecent      func(string)
	// SetLanguage takes "", "en", "zh-Hans" or "zh-Hant"; "" follows the
	// system again. Language reports the language currently in effect, for the
	// check marks.
	SetLanguage func(string)
	Language    func() string
}

// SetActions gives the window its application-level callbacks. It is called
// once, after Start built the App and the shell has a database path.
func (a *App) SetActions(actions Actions) {
	a.actions = actions
}

// sidebarActions draws the application menu at the bottom of the sidebar. It
// replaces a permanent window menu bar: the pages already have their own
// navigation, and the actions that are not page work (back up, export, import,
// data folder, About, language) stay one click away instead of hiding behind
// Alt.
func (a *App) sidebarActions(c *ui.Context) {
	if a.actions.empty() {
		return
	}
	ui.Column(c).FillWidth().Padding(12, 8, 16, 14).Children(func() {
		label := i18n.GetText("desktop.menu.actions")
		button := ui.MenuButton(c, label, a.actionsMenu).
			FillWidth().Height(34).PaddingX(12).Radius(buttonRadius).
			Background(ui.Hex(surface)).BorderWidth(1).BorderColor(ui.Hex(border))
		button.Label(label)
	})
}

func (a Actions) empty() bool {
	return a.BackUpDatabase == nil && a.ExportLotteries == nil && a.ImportLotteries == nil &&
		a.ShowDataFolder == nil && a.ShowAbout == nil && a.RecentDocuments == nil &&
		a.SetLanguage == nil
}

func (a *App) actionsMenu(m *ui.Menu) {
	menuAction(m, i18n.GetText("desktop.menu.backup"), a.actions.BackUpDatabase)
	menuAction(m, i18n.GetText("desktop.menu.export"), a.actions.ExportLotteries)
	menuAction(m, i18n.GetText("desktop.menu.import"), a.actions.ImportLotteries)
	menuAction(m, i18n.GetText("desktop.menu.show_data_folder"), a.actions.ShowDataFolder)
	if a.actions.RecentDocuments != nil && a.actions.OpenRecent != nil {
		m.Submenu(i18n.GetText("desktop.menu.open_recent"), func(sub *ui.Menu) {
			docs := a.actions.RecentDocuments()
			if len(docs) == 0 {
				sub.Item(i18n.GetText("desktop.menu.open_recent_empty")).Disabled(true)
				return
			}
			for _, path := range docs {
				path := path
				if sub.Item(filepath.Base(path)).Chosen() {
					a.actions.OpenRecent(path)
				}
			}
		})
	}
	m.Separator()
	menuAction(m, i18n.GetText("desktop.menu.about"), a.actions.ShowAbout)
	if a.actions.SetLanguage != nil {
		m.Submenu(i18n.GetText("desktop.menu.language"), func(sub *ui.Menu) {
			languageAction(sub, i18n.GetText("desktop.menu.language_system"), "", a)
			sub.Separator()
			languageAction(sub, i18n.GetText("desktop.menu.language_en"), "en", a)
			languageAction(sub, i18n.GetText("desktop.menu.language_zh"), "zh-Hans", a)
			languageAction(sub, i18n.GetText("desktop.menu.language_zh_hant"), "zh-Hant", a)
		})
	}
}

// menuAction adds one enabled or disabled action item. Chosen reports the
// click on the frame after the menu closed, so the callback runs exactly once.
func menuAction(m *ui.Menu, label string, fn func()) {
	item := m.Item(label)
	if fn == nil {
		item.Disabled(true)
		return
	}
	if item.Chosen() {
		fn()
	}
}

func languageAction(m *ui.Menu, label, tag string, app *App) {
	item := m.Item(label)
	// The system item has no resolved tag of its own; ticking it when an
	// override is active would claim a choice the app cannot observe.
	if tag != "" && app.actions.Language != nil {
		item.Checked(app.actions.Language() == tag)
	}
	if item.Chosen() {
		app.actions.SetLanguage(tag)
	}
}
