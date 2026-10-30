package nativeui

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// navigateByKeyboard changes the page from a window-wide shortcut and gives
// the new context a real focus target. The old page's focused control is
// disposed by the switch, so leaving focus at the window root would make the
// next Tab start from an unpredictable place.
func (a *App) navigateByKeyboard(c *ui.Context, p page) {
	if a.page == p {
		return
	}
	a.setPage(p)
	a.navigationControl.Focus(c)
}

// Reload re-reads the current page, for a change made outside the view (an
// imported export, a forwarded launch). A caller that owns a window must also
// invalidate it: changing this state does not schedule a frame on its own.
func (a *App) Reload() { a.refresh() }

// ShowPage switches the window to the page a deep link names. An unknown or
// empty name leaves the current page alone: a malformed link must not blank
// the window.
func (a *App) ShowPage(name string) {
	for p, id := range pageIDs {
		// The ids are the stable contract (sidebar choices, deep links); the
		// titles are shown text and may be translated.
		if strings.EqualFold(id, name) {
			a.setPage(p)
			return
		}
	}
	for p, title := range pageTitles {
		if strings.EqualFold(title, name) {
			a.setPage(p)
			return
		}
	}
}

// setPage switches the window to p and loads its data, which is what every
// entry point (a sidebar click, a deep link) has to do: a page that only drew
// what was already loaded showed an empty list until the operator pressed
// Refresh.
func (a *App) setPage(p page) {
	if a.page == p {
		return
	}
	a.page = p
	a.scroll.X, a.scroll.Y = 0, 0
	a.sidebarReveal = true
	a.pageAnnouncement = a.PageTitle()
	if a.actions.SetPageTitle != nil {
		a.actions.SetPageTitle(a.PageTitle())
	}
	a.refresh()
}

// ShowIntent applies a deep link: it selects the page the link names and, when
// the link also names a draw, verifies that draw. An empty page leaves the
// current page alone, so a malformed link cannot blank the window.
//
// Both the operating system's requests and the launch arguments go through
// this one method, so a link behaves the same whether it started the process
// or reached a running one.
func (a *App) ShowIntent(page, lotteryID string) {
	if page != "" {
		a.ShowPage(page)
	}
	if lotteryID != "" {
		a.ShowLotteryDraw(lotteryID)
	}
}

// ShowLotteryDraw opens the Lottery page focused on one draw and verifies it
// immediately, which is what `aurora://lottery?id=<draw>` promises.
func (a *App) ShowLotteryDraw(id string) {
	if id == "" {
		return
	}
	a.setPage(pageLottery)
	a.lottery.verifyID = id
	a.verifyLottery()
}

// shortcutChord renders a platform shortcut for the Settings reference. The
// keys are the ones header/actionShortcuts register.
func shortcutChord(key string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}
