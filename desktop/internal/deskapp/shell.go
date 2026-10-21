package deskapp

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
)

// Shell routes what the operating system asks the app to open onto the one
// window. It exists because the same request arrives twice on a cold start:
//
//   - the argument is on our own command line, which ParseFlags reads; and
//   - MyGo re-delivers it through App.OnOpenURL / App.OnOpenFile once the app
//     is ready, because the launch arguments are how Linux and Windows hand a
//     deep link or an associated file to an app.
//
// Applying both would import a file twice (the second import refused as a
// duplicate) or reload the page the window just loaded. Shell remembers what
// it has already honored and ignores the repeat, so the two delivery paths
// cannot double-fire an action.
//
// The bookkeeping is an "expect one echo" count, not a permanent set: a link
// the user opens again later must work again. Only the echo that MyGo delivers
// for an argument this shell has just acted on is suppressed, and the entry is
// dropped once that echo has been seen, so the map never grows.
type Shell struct {
	mu   sync.Mutex
	meta About
	win  *mygo.Window
	// expectEcho counts deliveries still owed to MyGo for arguments this shell
	// already acted on: the cold-start arguments ParseFlags read, and the ones
	// a second launch forwarded (MyGo re-delivers both through
	// OnOpenURL/OnOpenFile).
	expectEcho map[string]int
	// report shows a message to the operator. A shell launched from a file
	// manager has no terminal, so failures must be visible in a dialog.
	// It takes the severity too: a failure that arrives by a dropped file must
	// look like a failure, exactly as it does from the File menu.
	report func(typ mygo.MessageType, title, message, detail string)
	// navigate loads an intent in the window. Tests replace it to observe
	// exactly which links reach the page; the default needs a live window.
	navigate func(Intent)
	// pending is the last page an aurora:// link asked for while there was no
	// window yet (a second launch can arrive between RequestSingleInstanceLock
	// and WhenReady). It is loaded by Attach instead of being dropped.
	pending *Intent
}

// NewShell builds a shell for meta. The window is attached later, once MyGo
// has created it.
func NewShell(meta About) *Shell {
	s := &Shell{meta: meta, expectEcho: map[string]int{}}
	// The shell's own messages belong to the shell's window, so a dialog about
	// a dropped file is a sheet on that window rather than a floating box.
	// Before Attach (tests, an early failure) there is no window and the
	// dialog stands alone, which is what MyGo does with a nil Parent.
	s.report = func(typ mygo.MessageType, title, message, detail string) {
		showMessage(s.Window(), typ, title, message, detail)
	}
	return s
}

// Window returns the attached window, or nil before Attach.
func (s *Shell) Window() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.win
}

// Attach points the shell at the main window. Until then, navigation requests
// are parsed and recorded but not loaded, which is what a test without a
// window needs; a page that was requested in that gap is loaded here, once
// there is a window to load it in.
func (s *Shell) Attach(win *mygo.Window) {
	s.mu.Lock()
	s.win = win
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()
	if pending != nil {
		s.Navigate(*pending)
	}
}

// Consume marks launch arguments as already handled, so the delivery MyGo
// makes after it is ready does not repeat them. Call it once, before Run.
func (s *Shell) Consume(args ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, arg := range args {
		s.expectEcho[arg]++
	}
}

// take reports whether arg should be acted on now. The first delivery after
// Consume (or after this shell acted on a forwarded launch argument) is MyGo's
// echo of a request already honored, so it is dropped; anything later is a new
// request and is acted on.
func (s *Shell) take(arg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.expectEcho[arg] > 0 {
		if s.expectEcho[arg]--; s.expectEcho[arg] == 0 {
			delete(s.expectEcho, arg)
		}
		return false
	}
	return true
}

// oweEcho records that MyGo will re-deliver arg once, because this shell just
// acted on it in the OnSecondInstance callback (MyGo runs its own
// deliverArgs immediately after that callback).
func (s *Shell) oweEcho(args []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, arg := range args {
		s.expectEcho[arg]++
	}
}

// OpenURL handles a deep link from MyGo.
func (s *Shell) OpenURL(raw string) {
	if !s.take(raw) {
		return
	}
	in, ok := ParseIntent(raw)
	if !ok || in.File != "" {
		s.reportError("Aurora could not open that link", fmt.Errorf("%s is not an aurora:// link to a page", raw))
		return
	}
	s.Navigate(in)
}

// OpenFile handles a lottery export from MyGo: a file dropped on the window,
// or a path MyGo delivered from the launch arguments.
//
// Unlike links, files are not deduped here. MyGo only fires OnOpenFile for
// extensions named in the packaged fileAssociations, and Aurora deliberately
// declares none - so the launch-argument path never echoes a file back, and
// a permanent "already imported" entry would only ever block the user's next
// genuine import of that file. Importing the same export twice is already
// handled where it belongs: ImportRecords refuses the duplicate record and
// reports it, so the database keeps one copy.
func (s *Shell) OpenFile(path string) {
	result, err := ImportLotteryFile(s.meta, path)
	if err != nil {
		s.reportError("Import failed", err)
		return
	}
	s.reportImport(path, result)
}

// HandleLaunchArgs applies the arguments of a launch that reached this already
// running instance (App.OnSecondInstance): later links and later exports must
// land in the one window instead of starting a rival database writer.
//
// MyGo forwards every argument as it is, flags included, but it only fires
// OnOpenFile for extensions named in the packaged fileAssociations. Aurora
// deliberately declares none (every .json file would be claimed), so a second
// `aurora-desktop draws.json` reaches this handler alone: without the file
// branch the argument was dropped silently, and nothing was imported.
//
// The launch also raises the window, which is what a second double-click
// without arguments asks for.
//
// A launch that asks for a different --data-dir or --web-root is refused
// rather than half-applied: this process cannot move its database or swap its
// UI, and importing an export into the running instance's database when the
// user named another one would write to the wrong place without saying so.
// Flags that match what is already running are a no-op, like any other flag.
//
// workDir is the directory the launching process ran in. A relative export
// path is relative to THAT, not to this process (see ParseIntentIn), which is
// the same rule MyGo applies to the files it delivers itself.
func (s *Shell) HandleLaunchArgs(args []string, workDir string) {
	var echoed []string
	if conflict := s.pathConflict(args, workDir); conflict != "" {
		// The message stands on its own: Linux shows the message box without
		// the title, so "quit it before ..." alone would not say what "it" is.
		s.reportError("This Aurora is already running", errors.New(
			"Aurora is already running with a different "+conflict+
				". This window keeps the data directory and Web UI it started with; "+
				"quit Aurora first to launch with another one."))
		// MyGo runs deliverArgs right after this callback regardless of what
		// we did, so the refused launch's links would navigate on their own.
		// Own those echoes: a refusal that still opened the page would not be
		// a refusal.
		var links []string
		for _, arg := range args {
			if IsDeepLink(arg) {
				links = append(links, arg)
			}
		}
		s.oweEcho(links)
		s.Reveal()
		return
	}
	for _, arg := range args {
		in, ok := ParseIntentIn(arg, workDir)
		if !ok {
			// A malformed aurora:// link is still ours to report; flags and
			// unrelated positionals are not.
			if IsDeepLink(arg) {
				s.OpenURL(arg)
				echoed = append(echoed, arg)
			}
			continue
		}
		if in.File != "" {
			s.OpenFile(in.File)
			continue
		}
		s.OpenURL(in.Raw)
		echoed = append(echoed, in.Raw)
	}
	// MyGo runs its own deliverArgs right after this callback, so each link
	// acted on here arrives once more. Owning that echo keeps the window
	// from reloading the page it just loaded, without the link being dead
	// the next time the user opens it.
	s.oweEcho(echoed)
	s.Reveal()
}

// pathConflict reports the startup flag of a forwarded launch that this
// running instance cannot honor, or "" when there is none. An empty value
// means "not given", and a value naming what is already running is not a
// conflict.
//
// A relative value is joined onto workDir, the way MyGo resolves the files it
// delivers itself: `cd ~/Downloads && aurora-desktop --data-dir db` names
// ~/Downloads/db, not a directory beside this process.
func (s *Shell) pathConflict(args []string, workDir string) string {
	dataDir, webRoot := ForwardedPaths(args)
	var conflicts []string
	if dataDir != "" {
		want := filepath.Join(resolveAgainst(dataDir, workDir), "aurora.db")
		if s.meta.DBPath == "" || filepath.Clean(want) != filepath.Clean(s.meta.DBPath) {
			conflicts = append(conflicts, "--data-dir "+dataDir)
		}
	}
	if webRoot != "" && filepath.Clean(resolveAgainst(webRoot, workDir)) != filepath.Clean(s.meta.WebRoot) {
		conflicts = append(conflicts, "--web-root "+webRoot)
	}
	return strings.Join(conflicts, " and ")
}

// resolveAgainst makes a forwarded path absolute against the launching
// process's working directory, when it is relative and one was given.
func resolveAgainst(path, workDir string) string {
	if filepath.IsAbs(path) || workDir == "" {
		return path
	}
	return filepath.Join(workDir, path)
}

// Reveal brings the main window back to the front: a link opened while Aurora
// was minimized must show the page, not load it behind the user's other
// windows.
func (s *Shell) Reveal() {
	win := s.Window()
	if win == nil || win.IsDestroyed() || win.Page() == nil {
		return
	}
	if win.IsMinimized() {
		win.Restore()
	}
	win.Show()
	win.Focus()
}

// Navigate loads the page an intent names in the window, when there is one.
func (s *Shell) Navigate(in Intent) {
	if s.navigate != nil {
		s.navigate(in)
		return
	}
	s.mu.Lock()
	win := s.win
	if win == nil || win.IsDestroyed() || win.Page() == nil {
		if win == nil {
			s.pending = &in
		}
		// No window yet is a startup race, not a reason to forget the link:
		// keep the newest one for Attach. A window that exists but has been
		// destroyed is a different case - there is nothing to show it in.
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	win.Page().LoadURL(in.URL())
	s.Reveal()
}

// ImportDropped handles files dropped onto the window: a lottery export is
// imported, anything else is reported as unsupported instead of being
// silently ignored.
func (s *Shell) ImportDropped(paths []string) {
	for _, path := range paths {
		if !isLotteryDocument(path) {
			s.reportError("Unsupported file",
				fmt.Errorf("%s is not an Aurora lottery export (.json); nothing was imported", filepath.Base(path)))
			continue
		}
		s.OpenFile(path)
	}
}

func (s *Shell) reportError(title string, err error) {
	s.report(mygo.MessageError, title, err.Error(), "")
}

// reportImport shows the same accounting the File > Import action shows, so a
// file opened from the file manager and one chosen in the dialog report
// identically: one place builds both the wording (importReport) and the
// severity (importReportType), so a partial import is a warning on every path
// and never a plain success.
func (s *Shell) reportImport(path string, result lotteryapp.ImportResult) {
	title, message, detail := importReport(result, path)
	s.report(importReportType(result), title, message, detail)
}

// messageBox puts a message on screen. It is a variable rather than a direct
// call so a test can observe the dialog a shell or menu action would show
// without needing a live window.
var messageBox = func(opts mygo.MessageOptions) {
	_, _ = mygo.Dialog.Message(opts)
}

// showMessage is the one message dialog constructor for the shell and the
// menu: same call, same severity handling, wherever the message comes from.
// parentWin attaches the dialog to that window when there is one, so it is a
// sheet on macOS and a child modal elsewhere instead of a stray window the
// user can leave behind while the app carries on.
func showMessage(parentWin *mygo.Window, typ mygo.MessageType, title, message, detail string) {
	messageBox(mygo.MessageOptions{
		Parent:  parentWin,
		Type:    typ,
		Title:   title,
		Message: message,
		Detail:  detail,
	})
}
