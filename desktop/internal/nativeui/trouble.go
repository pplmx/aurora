package nativeui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// Reporting which pages could not be read, from anywhere in the window.
//
// Every page already warns inside its own cards when a read fails, and that is
// right where the reader is looking. What it does not do is tell a reader who is
// somewhere else: an operator sitting on Dashboard cannot see that the token
// page failed to load, and a page that is failing quietly while its own card is
// off screen looks exactly like a page with nothing to report.
//
// The reference apps built on MyGo treat "which one needs me" as the window's
// job rather than each pane's: Kero's sidebar answers it for a dozen sessions at
// once, and one shortcut takes you to the one waiting. The same rule applied to a
// handful of pages: one line in the sidebar, and a click that opens the first
// page that is failing.

// pageLoadFailure is one page that could not be read, in sidebar order.
type pageLoadFailure struct {
	Page page
	Err  error
}

// loadFailures collects the pages whose last read failed, in sidebar order. The
// order matters: the click opens the first entry, so the line always leads to the
// same page for the same state rather than to whichever field was checked first.
func (a *App) loadFailures() []pageLoadFailure {
	var out []pageLoadFailure
	add := func(p page, err error) {
		if err != nil {
			out = append(out, pageLoadFailure{Page: p, Err: err})
		}
	}
	// Sidebar order, so the first entry is the topmost failing page.
	add(pageDashboard, a.dashboard.loadErr)
	add(pageLottery, a.lottery.loadErr)
	add(pageVoting, a.voting.loadErr)
	add(pageToken, a.token.loadErr)
	add(pageOracle, a.oracle.loadErr)
	add(pageNFT, a.nft.loadErr)
	add(pageBlockchain, a.blockchain.loadErr)
	return out
}

// troubleNoticeAfter is how long a page must stay unreadable before its failure is
// worth a system notification. A page that recovers on the next read — a locked
// database, a transient write — heals itself, and a notification for it would be
// noise; the sidebar line already reports it for anyone at the window.
const troubleNoticeAfter = 30 * time.Second

// troubleNotice tracks one page's failure across frames, so a notice is raised
// when it has lasted rather than when it first appeared.
type troubleNotice struct {
	page  page
	since time.Time
	// notified records that the notice for this failure has been raised, so a
	// long outage notifies once instead of every frame past the threshold.
	notified bool
}

// maybeNotifyTrouble raises a notice for a failure that has outlasted the
// threshold while the window is in the background. Without live updates a failed
// page does not retry itself, so it stays failed until the reader comes back and
// presses Refresh — which is exactly the case worth a notification.
func (a *App) maybeNotifyTrouble(now time.Time) {
	failures := a.loadFailures()
	if len(failures) == 0 {
		a.trouble = troubleNotice{}
		return
	}
	first := failures[0]
	if a.trouble.page != first.Page || a.trouble.since.IsZero() {
		a.trouble = troubleNotice{page: first.Page, since: now}
		return
	}
	if a.trouble.notified || now.Sub(a.trouble.since) < troubleNoticeAfter {
		return
	}
	if a.actions.Notify == nil || a.actions.Active == nil || a.actions.Active() {
		// Not a notification opportunity: the window is in front, where the
		// sidebar line and the page's own warning already say this.
		return
	}
	a.trouble.notified = true
	a.actions.Notify(NotifyOptions{
		ID:    "page-failed:" + pageIDs[first.Page],
		Page:  pageIDs[first.Page],
		Title: tr("Aurora couldn't read a page"),
		Body:  fmt.Sprintf(tr("%s has not loaded. Open it to retry."), tr(pageTitles[first.Page])),
	})
}

// troubleLine is the sidebar sentence, or "" when nothing is failing. One page
// is named; several are counted, because a list of names in a status line wraps
// and stops being scannable.
func troubleLine(failures []pageLoadFailure) string {
	switch len(failures) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(tr("Couldn't read %s"), tr(pageTitles[failures[0].Page]))
	default:
		return fmt.Sprintf(tr("%d pages couldn't be read"), len(failures))
	}
}

// openFirstFailure moves to the topmost page that is failing. The reader gets the
// page with its own warning and its own Refresh, which is where they can act.
func (a *App) openFirstFailure() {
	failures := a.loadFailures()
	if len(failures) == 0 {
		return
	}
	a.ShowPage(pageTitles[failures[0].Page])
}

// syncWindowTitle pushes the native title when it changes, and only then. The
// title carries the failure state so a reader in another window sees that
// something needs them — the reference apps built on MyGo make the window answer
// "which one needs me" rather than leaving it to the pane.
//
// It compares against the last value pushed because the frame runs on every
// repaint: setting the title each time would be a platform call per frame, and
// a task switcher does not need the same string re-set sixty times a second.
func (a *App) syncWindowTitle() {
	if a.actions.SetPageTitle == nil {
		return
	}
	title := a.PageTitle()
	if len(a.loadFailures()) > 0 {
		title = tr("Needs attention") + " — " + title
	}
	if title == a.windowTitle {
		return
	}
	a.windowTitle = title
	a.actions.SetPageTitle(title)
}

// syncBadgeCount mirrors the number of failing pages onto the application icon,
// so a reader whose window is completely hidden still sees that something needs
// them. Zero clears it, which is how the platform spells "nothing to report".
func (a *App) syncBadgeCount() {
	if a.actions.SetBadgeCount == nil {
		return
	}
	count := len(a.loadFailures())
	if count == a.badgeCount && a.badgePushed {
		return
	}
	a.badgeCount = count
	a.badgePushed = true
	a.actions.SetBadgeCount(count)
}

// waitForWrite draws the line a page shows while it has read nothing: the busy
// bar names the write that owns the database, and this says the page is waiting
// for it rather than reporting on a store it has not looked at. It reports
// whether it drew, so a caller returns without repeating the check.
//
// It is one function rather than a string each page keeps because the condition
// and the sentence belong together: five lists across four pages show it, and a
// page that got the wording without the condition — or the reverse — would claim
// something about a store nothing has read.
func (a *App) waitForWrite(c *ui.Context, rows int, loaded bool) bool {
	if rows > 0 || loaded {
		return false
	}
	hint(c, "Waiting for the running write to finish before reading this page.")
	return true
}
