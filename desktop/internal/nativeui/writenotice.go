package nativeui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// A background write disables the page body but not the sidebar, so an operator
// waiting for a slow fetch can move to another page — which is the point of
// running the write off the frame thread. The outcome, though, was reported only
// on the page that started it: switch away and the busy bar disappears, the
// result lands in a status line nobody is looking at, and the write looks as if
// it were dropped mid-flight.
//
// When that happens the outcome is repeated as a toast. On the same page it is
// not: the status strip is the right place there — it persists until dismissed,
// sits where the operator was working, and does not slide a message over the
// form being read.

// settleWrite reports a write's outcome when the reader left the page it started
// on. It returns whether anything was reported, which is what the caller uses to
// decide whether the outcome still needs the page's own status line.
func (a *App) settleWrite(c *ui.Context, origin page, s statusLine) bool {
	if s.empty() || a.page == origin {
		return false
	}
	toast := ui.Toast{
		// One toast per origin page: two writes from the same form replace the
		// message instead of stacking two nearly identical sentences.
		ID:    "write-" + pageIDs[origin],
		Title: s.spoken(),
		// The action is what makes the message useful rather than merely
		// polite. The page is named so the click is predictable: a bare "Go
		// back" would not say where.
		Action:   a.returnLabel(origin),
		OnAction: func() { a.goToOrigin(origin) },
	}
	toast.Type = "success"
	if s.isErr {
		toast.Type = "error"
	}
	c.AddToast(toast)
	c.Invalidate()
	return true
}

// returnLabel names the action button after the page the write happened on, in
// the window's language. A translated title is used because the button reads as
// part of the sentence above it, not as a page identifier.
func (a *App) returnLabel(p page) string {
	return fmt.Sprintf(tr("Open %s"), tr(pageTitles[p]))
}

// goToOrigin is the toast action's body. It goes through setPage rather than
// setting the page directly, so the destination loads its data and announces
// itself exactly as a sidebar click does.
func (a *App) goToOrigin(p page) {
	a.setPage(p)
	if a.invalidate != nil {
		a.invalidate()
	}
}
