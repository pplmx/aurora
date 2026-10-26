package nativeui

import (
	"path/filepath"
	"runtime"

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
	// Version, BuildTime and DBPath are shown on the Settings page; they are
	// values rather than callbacks because they do not change at run time.
	Version   string
	BuildTime string
	DBPath    string
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

// sidebarActions draws the File menu at the bottom of the sidebar. It replaces
// a permanent window menu bar: file operations stay one click away instead of
// hiding behind Alt, while application configuration lives on Settings.
func (a *App) sidebarActions(c *ui.Context) {
	if a.actions.empty() {
		return
	}
	ui.Column(c).FillWidth().Padding(12, 8, 16, 14).Children(func() {
		label := i18n.GetText("desktop.menu.file")
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

// actionShortcuts keeps the file shortcuts that used to come from the native
// menu. Linux and Windows install no menu now, so the window registers
// Cmd/Ctrl+E/I/B itself; macOS keeps them on the application menu.
func (a *App) actionShortcuts(c *ui.Context) {
	if runtime.GOOS == "darwin" {
		return
	}
	mod := platformRefreshModifier()
	for _, action := range []struct {
		key ui.Key
		fn  func()
	}{
		{ui.KeyE, a.actions.ExportLotteries},
		{ui.KeyI, a.actions.ImportLotteries},
		{ui.KeyB, a.actions.BackUpDatabase},
	} {
		if action.fn != nil && c.Shortcut(mod, action.key) {
			action.fn()
		}
	}
}

func (a *App) actionsMenu(m *ui.Menu) {
	menuAction(m, i18n.GetText("desktop.menu.backup"), a.actions.BackUpDatabase)
	menuAction(m, i18n.GetText("desktop.menu.export"), a.actions.ExportLotteries)
	menuAction(m, i18n.GetText("desktop.menu.import"), a.actions.ImportLotteries)
	if a.actions.RecentDocuments != nil && a.actions.OpenRecent != nil {
		m.Separator()
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
