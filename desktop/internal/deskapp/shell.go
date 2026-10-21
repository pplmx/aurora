package deskapp

import (
	"fmt"
	"path/filepath"
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
type Shell struct {
	mu      sync.Mutex
	meta    About
	win     *mygo.Window
	handled map[string]bool
	// report shows a message to the operator. A shell launched from a file
	// manager has no terminal, so failures must be visible in a dialog.
	report func(title, message, detail string)
	// navigate loads an intent in the window. Tests replace it to observe
	// exactly which links reach the page; the default needs a live window.
	navigate func(Intent)
}

// NewShell builds a shell for meta. The window is attached later, once MyGo
// has created it.
func NewShell(meta About) *Shell {
	return &Shell{meta: meta, handled: map[string]bool{}, report: showMessage}
}

// Attach points the shell at the main window. Until then, navigation requests
// are parsed and recorded but not loaded, which is what a test without a
// window needs.
func (s *Shell) Attach(win *mygo.Window) {
	s.mu.Lock()
	s.win = win
	s.mu.Unlock()
}

// Consume marks launch arguments as already handled, so the delivery MyGo
// makes after it is ready does not repeat them. Call it once, before Run.
func (s *Shell) Consume(args ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, arg := range args {
		s.handled[arg] = true
	}
}

// take reports whether arg is new, marking it handled.
func (s *Shell) take(arg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handled[arg] {
		return false
	}
	s.handled[arg] = true
	return true
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
func (s *Shell) OpenFile(path string) {
	if !s.take(path) {
		return
	}
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
func (s *Shell) HandleLaunchArgs(args []string) {
	for _, arg := range args {
		in, ok := ParseIntent(arg)
		if !ok {
			// A malformed aurora:// link is still ours to report; flags and
			// unrelated positionals are not.
			if IsDeepLink(arg) {
				s.OpenURL(arg)
			}
			continue
		}
		if in.File != "" {
			s.OpenFile(in.File)
			continue
		}
		s.OpenURL(in.Raw)
	}
	s.Reveal()
}

// Reveal brings the main window back to the front: a link opened while Aurora
// was minimized must show the page, not load it behind the user's other
// windows.
func (s *Shell) Reveal() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
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
	s.mu.Unlock()
	if win == nil || win.IsDestroyed() || win.Page() == nil {
		return
	}
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
	s.report(title, err.Error(), "")
}

// reportImport shows the same accounting the File > Import action shows, so a
// file opened from the file manager and one chosen in the dialog report
// identically.
func (s *Shell) reportImport(path string, result lotteryapp.ImportResult) {
	if len(result.Rejected) > 0 {
		s.report("Import partially failed",
			fmt.Sprintf("Imported %d of %d lottery records (failed: %d).", result.Imported, result.Total, len(result.Rejected)),
			fmt.Sprintf("Rejected record indices (0-based): %v\n\n%s", result.Rejected, path))
		return
	}
	s.report("Lotteries imported", fmt.Sprintf("%d draws imported.", result.Imported), path)
}

func showMessage(title, message, detail string) {
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageInfo,
		Title:   title,
		Message: message,
		Detail:  detail,
	})
}
