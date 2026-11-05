package nativeui

// One answer to "may this window read the database right now".
//
// A background write owns the SQLite connection, so every read taken while one
// runs lands on a locked database. What that looked like depended on the path: a
// list read recorded the page as failed and then showed it in the sidebar, the
// window title and the application icon; a search marked the page as read when it
// had not been; a detail button recorded an error for a card nobody had asked
// for.
//
// The window draws its page body disabled while a write runs, and that is not
// enough on its own: `ui.Disabled` is a paint-time state, so a programmatic
// caller — or a test's click — reaches the handler anyway. Each read therefore
// has to ask, and they all have to ask the same question, or the next path added
// is the one that forgets. Three rounds were spent finding paths that had been
// missed; readBlocked is the one place the answer lives now.

// readBlocked reports whether a read must wait for the running write.
func (a *App) readBlocked() bool {
	return a.busy != ""
}
