package nativeui

import (
	"path/filepath"
	"strings"
	"time"

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
	// Runtime is the toolchain line ("go1.27.3 · mygo v0.7.3") shown in the
	// copied diagnostics; empty drops the row.
	Runtime string
	DBPath  string
	// DatabaseSize reports the ledger's current disk usage (the database plus
	// its -wal/-shm siblings). Unlike the values above it is a callback because
	// it grows as the app writes; the About readout asks each time it draws.
	DatabaseSize func() string
	// LedgerFacts reports the chain's block count, short head hash and verdict
	// for the diagnostics copy. It is asked once per copy, not per frame:
	// verifying the chain walks every block's proof of work.
	LedgerFacts func() (blocks int, head string, valid bool, ok bool)
	// OutboxPending reports the audit events parked in the outbox, the number
	// `aurora doctor` prints; the diagnostics copy quotes it the same way.
	OutboxPending func() (pending int, ok bool)
	// OutboxOldest reports how long the oldest parked audit event has waited,
	// so the copied diagnostics can say whether the retry loop is moving.
	OutboxOldest func() (age time.Duration, ok bool)
	// OutboxModules names the modules whose events are parked, so following the
	// backlog can narrow the ledger to them instead of opening the whole chain.
	// It is optional: without it the backlog line simply opens the ledger.
	OutboxModules func() []string
	// RecentDocuments lists the lottery exports the user opened or saved;
	// OpenRecent loads one through the same import path as a drop.
	RecentDocuments func() []string
	OpenRecent      func(string)
	// SetLanguage takes "", "en", "zh-Hans" or "zh-Hant"; "" follows the
	// system again. It reports a failed write so the page can say the choice
	// will not survive a restart — the language is applied either way.
	// Language reports the language currently in effect, for the check marks;
	// FollowsSystem reports whether no language was ever chosen, which the
	// effect alone cannot say — a system locale of English looks the same as an
	// explicit English choice until this answers.
	SetLanguage   func(string) error
	Language      func() string
	FollowsSystem func() bool
	// LiveUpdates/SetLiveUpdates persist the Settings toggle. The getter is
	// read when SetActions runs; the setter reports a failed write so the page
	// can show it instead of silently resetting on the next launch.
	LiveUpdates    func() bool
	SetLiveUpdates func(bool) error
	// ThemeMode/SetThemeMode persist the appearance choice. The shell also
	// applies it to MyGo's native chrome so the title bar and dialogs match
	// the window the view draws.
	ThemeMode    func() ThemeMode
	SetThemeMode func(ThemeMode) error
	SystemDark   func() bool
	// SetPageTitle keeps the native window title in step with the visible
	// page (and its language); optional for view tests.
	SetPageTitle func(string)
	// Zoom/SetContentZoom persist the window's content zoom. The getter is read
	// when SetActions runs; the setter runs on every zoom and reports nothing,
	// because a zoom the operator reaches for while reading must not raise a
	// dialog about a settings file.
	Zoom           func() float32
	SetContentZoom func(float32)
	// Notify raises a system notification, and Active reports whether the
	// window is the front one. Both are optional: the view only decides *when*
	// a notification is worth raising, and the shell owns showing it, because
	// that needs the app handle.
	Notify func(NotifyOptions)
	Active func() bool
	// PageChanged reports a page switch by its stable id (the one the sidebar
	// chooses by, not the title). The shell records it so the next launch can
	// open the same page; optional for view tests.
	PageChanged func(id string)
	// LastPage is the page the previous launch closed on, or empty. SetActions
	// resolves it before the first frame, so the window opens where the
	// operator left it instead of flashing the dashboard first.
	LastPage string
}

// SetActions gives the window its application-level callbacks. It is called
// once, after Start built the App and the shell has a database path.
func (a *App) SetActions(actions Actions) {
	a.actions = actions
	if actions.LiveUpdates != nil {
		a.liveUpdates = actions.LiveUpdates()
	}
	if actions.ThemeMode != nil {
		a.themeMode = normalizeThemeMode(actions.ThemeMode())
	}
	if actions.SystemDark != nil {
		a.systemDark = actions.SystemDark()
	}
	if actions.Zoom != nil {
		a.contentZoom = clampZoom(actions.Zoom())
	}
	// Open where the previous launch left off before the first frame draws, so
	// the window does not paint the dashboard and then jump to the real page.
	// ShowPage ignores an id no page answers to: a first launch, a page a later
	// release removed and a hand-edited settings file all land on the default
	// instead of a blank window.
	if actions.LastPage != "" {
		a.ShowPage(actions.LastPage)
	}
	if actions.SetPageTitle != nil {
		actions.SetPageTitle(a.PageTitle())
	}
}

// SetSystemDark applies the desktop's effective appearance to Follow system
// mode. The shell calls it when the OS appearance changes, then invalidates
// the window so the next frame resolves the other palette.
func (a *App) SetSystemDark(dark bool) {
	a.systemDark = dark
}

// sidebarActions draws the File menu at the bottom of the sidebar. It replaces
// a permanent window menu bar: file operations stay one click away instead of
// hiding behind Alt, while application configuration lives on Settings.
func (a *App) sidebarActions(c *ui.Context) {
	if a.actions.empty() {
		return
	}
	ui.Column(c).FillWidth().Padding(space3, space5, space4, space2).Gap(space2).Children(func() {
		label := i18n.GetText("desktop.menu.file")
		button := ui.MenuButton(c, label, a.actionsMenu).
			FillWidth().Height(max(34, textSize(c, 34))).
			PaddingX(max(12, textSize(c, 12))).Radius(radiusSmall + 2).
			Background(c.Theme().Surface).BorderWidth(1).BorderColor(c.Theme().Border).
			Disabled(a.busy != "")
		button.Label(label)
		if a.actions.Version != "" {
			ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space1, 2).Children(func() {
				statusDot(c, c.Theme().Success, 5)
				ui.Text(c, tr("Local ledger")).FontSize(textSize(c, 10)).
					TextColor(c.Theme().TextMuted).Grow(1).Shrink(1).MinWidth(0).SingleLine()
				ui.Text(c, versionBadge(a.actions.Version)).FontSize(textSize(c, 10)).
					TextColor(c.Theme().TextMuted).SingleLine()
			})
		}
		// A parked audit backlog is a condition, not an event: it can last for
		// minutes, so it belongs where the ledger's own health already sits and
		// must be visible without opening anything. It is shown only when there
		// is one — an empty outbox is healthy and has nothing to say.
		if line, stuck := outboxLine(a.readOutbox()); line != "" {
			tone := c.Theme().Warning
			if stuck {
				tone = c.Theme().Danger
			}
			// The line is also a way in. A reader who sees a backlog wants to know
			// what is parked, and the ledger page is where those events are
			// visible; a status line nobody can act on leaves them to find the
			// page themselves, which is the problem this report was added to fix.
			// It opens the audit page rather than starting a drain: the drainer
			// is a background retry loop owned by the runtime, and a window that
			// started a second pass would race it.
			b := ui.ButtonBase(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space1, 2).
				Role(ui.RoleStatus).Label(line).Description(tr("Open the audit ledger"))
			fill := ui.Transparent
			if b.Hovered() || b.Pressed() {
				fill = c.Theme().SurfaceHover
			}
			b.Background(fill)
			b.Children(func() {
				statusDot(c, tone, 5)
				ui.Text(c, line).FontSize(textSize(c, 10)).TextColor(tone).
					Grow(1).Shrink(1).MinWidth(0).Wrap()
			})
			if b.Clicked() {
				a.followOutbox()
			}
		}
	})
}

// versionBadge is the short version the sidebar footer shows. The value comes
// from the shell, and the two entry points disagree about its shape: a
// development build links `git describe` ("v1.1-1745-gac0f69b", already
// prefixed) while an unstamped `go build` keeps the source default ("0.0.1",
// bare). The footer used to prepend "v" unconditionally, so a tagged build —
// the only kind a user installs — read vv1.1.
func versionBadge(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func (a Actions) empty() bool {
	return a.BackUpDatabase == nil && a.ExportLotteries == nil && a.ImportLotteries == nil &&
		a.ShowDataFolder == nil && a.ShowAbout == nil && a.RecentDocuments == nil &&
		a.SetLanguage == nil && a.SetThemeMode == nil && a.OutboxPending == nil
}

// actionShortcuts keeps the file shortcuts that used to come from the native
// menu. The native bar now carries platform roles only on every OS, so the
// window registers Cmd/Ctrl+E/I/B itself instead of leaving macOS without them.
func (a *App) actionShortcuts(c *ui.Context) {
	if a.busy != "" {
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
