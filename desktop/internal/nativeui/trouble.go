package nativeui

import "fmt"

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
