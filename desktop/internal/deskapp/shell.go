package deskapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
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
	// reload re-reads the current page after an import. Tests replace it; the
	// default reaches the App when one is attached.
	reload func()
	// createWindow recreates the main window after it was closed. It is set by
	// main and nil in tests or before the app can create windows.
	createWindow func() *mygo.Window
	// pending is the last page an aurora:// link asked for while there was no
	// live window. A second launch can arrive before WhenReady creates one, or
	// after macOS kept the app alive with its last window closed. Attach loads
	// the newest request once there is a window for it.
	pending *Intent
	// app is the window's state. The shell sets its current page when a link
	// or a second launch names one, and the window's Content closure reads it.
	app *nativeui.App
	// lotteryRepo is the shared repository owned by the application container.
	// File import/export uses it instead of opening a second SQLite handle.
	lotteryRepo lottery.Repository
	// backup is the container's backup use case; nil in a shell built before
	// the runtime and reported by BackUpDatabase.
	backup func(ctx context.Context, dest string) (string, error)
	// prefs are the desktop-only choices that survive a launch. Session keys
	// are not here by design.
	prefs     Preferences
	prefsPath string
}

// NewShell builds a shell for meta. The window is attached later, once MyGo
// has created it.
func NewShell(meta About) *Shell {
	s := &Shell{meta: meta, expectEcho: map[string]int{}, prefsPath: preferencesPath(meta.DBPath)}
	s.prefs = loadPreferences(s.prefsPath)
	// The shell's own messages belong to the shell's window, so a dialog about
	// a dropped file is a sheet on that window rather than a floating box.
	// Before Attach (tests, an early failure) there is no window and the
	// dialog stands alone, which is what MyGo does with a nil Parent.
	s.report = func(typ mygo.MessageType, title, message, detail string) {
		showMessage(s.Window(), typ, title, message, detail)
	}
	return s
}

// SetLotteryRepository injects the container-owned lottery repository. File
// import/export is a one-shot action, but reopening the database for it left a
// second SQLite handle next to the app's writer for no benefit.
func (s *Shell) SetBackup(backup func(ctx context.Context, dest string) (string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backup = backup
}

func (s *Shell) SetLotteryRepository(repo lottery.Repository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lotteryRepo = repo
}

// lotteryRepository returns the container-owned repository. A shell built
// before the runtime is wired reports the missing dependency instead of opening
// a second SQLite writer.
func (s *Shell) lotteryRepository() (lottery.Repository, func(), error) {
	s.mu.Lock()
	repo := s.lotteryRepo
	s.mu.Unlock()
	if repo == nil {
		return nil, nil, errors.New(T("desktop.error.lottery_repo_unavailable"))
	}
	return repo, func() {}, nil
}

// SetApp gives the shell the window state a link can steer. It must be called
// before any link is handled, so Navigate has something to point at.
func (s *Shell) SetApp(app *nativeui.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = app
}

// App returns the window state, or nil before SetApp.
func (s *Shell) App() *nativeui.App {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.app
}

// Window returns the attached window, or nil before Attach.
func (s *Shell) Window() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.win
}

// SetWindowFactory gives the shell a way to recreate the main window after it
// was closed. On macOS the process deliberately outlives its last window, so
// an OS request that arrives later must not be handled against a destroyed
// window. The factory is nil-able for tests and for callers without a window.
func (s *Shell) SetWindowFactory(create func() *mygo.Window) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createWindow = create
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
		s.reportError(T("desktop.error.open_link_title"),
			fmt.Errorf(T("desktop.error.open_link_message"), raw))
		return
	}
	s.EnsureWindow()
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
	// Reopen before importing: both the result and a failure are shown in a
	// dialog, and on macOS the process may still be alive with no window.
	s.EnsureWindow()
	result, err := importLotteryFile(s.lotteryRepository, path)
	if err != nil {
		s.reportError(T("desktop.error.import_failed"), err)
		return
	}
	rememberDocument(path)
	if result.Imported > 0 {
		// The import changed the lottery store this window is showing; re-read
		// the page before the dialog covers it, so dismissing the report does
		// not reveal the history from before the import.
		s.reloadWindow()
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
// A launch that asks for a different --data-dir is refused
// rather than half-applied: this process cannot move its database or swap its
// UI, and importing an export into the running instance's database when the
// user named another one would write to the wrong place without saying so.
// Flags that match what is already running are a no-op, like any other flag.
//
// workDir is the directory the launching process ran in. A relative export
// path is relative to THAT, not to this process (see ParseIntentIn), which is
// the same rule MyGo applies to the files it delivers itself.
func (s *Shell) HandleLaunchArgs(args []string, workDir string) {
	// A launch is also a request to see the app. On macOS the process can
	// still be running with its last window closed, so importing or navigating
	// before recreating it would apply the request invisibly.
	s.EnsureWindow()

	var echoed []string
	if conflict := s.pathConflict(args, workDir); conflict != "" {
		// The message stands on its own: Linux shows the message box without
		// the title, so "quit it before ..." alone would not say what "it" is.
		s.reportError(T("desktop.error.already_running_title"),
			errors.New(TFormat("desktop.error.already_running_message", conflict)))
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
	dataDir := ForwardedPaths(args)
	if dataDir == "" {
		return ""
	}
	want := filepath.Join(resolveAgainst(dataDir, workDir), "aurora.db")
	if s.meta.DBPath == "" || !samePath(want, s.meta.DBPath) {
		return "--data-dir " + dataDir
	}
	return ""
}

// samePath reports whether two spellings name the same filesystem object.
// Cleaning catches trailing separators and redundant components. When both
// paths exist, os.SameFile also catches a symlinked directory alias: that is
// still the database and UI the running instance uses, not a different one.
// A missing path falls back to the lexical comparison so it remains a
// conflict instead of matching some other nonexistent spelling by accident.
func samePath(a, b string) bool {
	if aInfo, aErr := os.Stat(a); aErr == nil {
		if bInfo, bErr := os.Stat(b); bErr == nil {
			return os.SameFile(aInfo, bInfo)
		}
	}
	return filepath.Clean(a) == filepath.Clean(b)
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

// MainWindow is the window a macOS Dock click or a second launch asks for,
// creating it again when it was closed. Closing the last window on macOS
// leaves the app running (see main's OnWindowAllClosed), so "open the app"
// must mean "show the window again" rather than "do nothing".
//
// create runs only when there is no live window; the caller wires the new
// window exactly like the first one. It is a nil-able hook so a shell without
// a window factory (tests) still answers.
func (s *Shell) MainWindow(create func() *mygo.Window) *mygo.Window {
	if win := s.Window(); win != nil && !win.IsDestroyed() {
		s.Reveal()
		return win
	}
	if create == nil {
		return nil
	}
	win := create()
	if win == nil {
		return nil
	}
	s.Attach(win)
	s.Reveal()
	return win
}

// EnsureWindow returns the live main window, recreating it when the app is
// still running without one. It is the path every OS-delivered intent uses:
// deep links and files must land in a visible window instead of being applied
// to a destroyed one.
func (s *Shell) EnsureWindow() *mygo.Window {
	// Before WhenReady, NewWindow panics. A request that arrives that early is
	// not lost: Navigate records it and the startup Attach loads it.
	if !appReady() {
		return nil
	}
	s.mu.Lock()
	create := s.createWindow
	s.mu.Unlock()
	return s.MainWindow(create)
}

// Navigate loads the page an intent names in the window, when there is one.
func (s *Shell) Navigate(in Intent) {
	if s.navigate != nil {
		s.navigate(in)
		return
	}
	s.mu.Lock()
	win := s.win
	app := s.app
	if win == nil || win.IsDestroyed() || app == nil {
		// No live window is either a startup race or a macOS process whose
		// last window was closed. Keep the newest request for the next Attach;
		// EnsureWindow is what turns that into a visible window.
		s.pending = &in
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	// A native window has no page to load: the link selects which view the
	// App renders, and changing that state does not schedule a frame on its own
	// (a second launch would switch the state but leave the old page on screen
	// until the next click). Invalidate draws the new page now, then Reveal
	// brings the window forward.
	app.ShowIntent(in.Page, in.LotteryID)
	win.Invalidate()
	s.Reveal()
}

// ImportDropped handles files dropped onto the window: a lottery export is
// imported, anything else is reported as unsupported instead of being
// silently ignored.
func (s *Shell) ImportDropped(paths []string) {
	for _, path := range paths {
		if !isLotteryDocument(path) {
			s.reportError(T("desktop.error.unsupported_file_title"),
				fmt.Errorf(T("desktop.error.unsupported_file_message"), filepath.Base(path)))
			continue
		}
		s.OpenFile(path)
	}
}

// reloadWindow re-reads the current page and redraws the window after a change
// made outside the view (an import), when there is a live window.
func (s *Shell) reloadWindow() {
	s.mu.Lock()
	win, app := s.win, s.app
	s.mu.Unlock()
	switch {
	case s.reload != nil:
		s.reload()
	case app != nil:
		app.Reload()
	}
	if win != nil && !win.IsDestroyed() {
		win.Invalidate()
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

// The methods below are the application actions the File menu and Settings page
// run. They use the shell's current window as the dialog parent, so every entry
// point shares the same dialog behavior.

func (s *Shell) BackUpDatabase() {
	backUpDatabase(s.backup, s.meta.DBPath, s.Window())
}

func (s *Shell) ExportLotteries() {
	exportLotteriesWith(s.lotteryRepository, s.Window())
}

func (s *Shell) ImportLotteries() {
	var reload func()
	if app := s.App(); app != nil {
		reload = app.Reload
	}
	importLotteriesWith(s.lotteryRepository, s.Window(), reload)
}

func (s *Shell) ShowDataFolder() {
	if s.meta.DBPath != "" {
		mygo.Shell.ShowItemInFolder(s.meta.DBPath)
	}
}

// RecentDocuments is the list MyGo keeps for File → Open Recent, now also
// available to the in-window File menu.
func (s *Shell) RecentDocuments() []string {
	if !appReady() {
		return nil
	}
	return mygo.App.RecentDocuments()
}

// OpenRecent imports a document from that list through the same path as a
// dropped file, so an old export and a new one behave identically.
func (s *Shell) OpenRecent(path string) {
	s.OpenFile(path)
}

func (s *Shell) ShowAbout() {
	showAbout(s.meta, s.Window())
}

// SetPageTitle names the visible page in the native window title. The app name
// stays in front so task switchers can tell Aurora windows apart.
func (s *Shell) SetPageTitle(title string) {
	if win := s.Window(); win != nil && title != "" {
		win.SetTitle(AppName + " — " + title)
	}
}

// SetLanguage changes the app's language. MyGo reports the change through
// OnLocaleChanged, which relabels the native menu and redraws the window.
func (s *Shell) SetLanguage(tag string) {
	mygo.App.SetLocale(tag)
}

// Language reports the language in effect, for Settings' current-language row.
func (s *Shell) Language() string {
	return appLocale()
}

// LiveUpdates reports the persisted live-update choice. It is read when the
// window receives its Actions, so Settings opens on the saved value.
func (s *Shell) LiveUpdates() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prefs.LiveUpdates
}

// SetLiveUpdates persists the choice before reporting success: the window
// applies it immediately, and a failed write must be visible in Settings.
func (s *Shell) SetLiveUpdates(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.prefs
	next.LiveUpdates = on
	if err := savePreferences(s.prefsPath, next); err != nil {
		return err
	}
	s.prefs = next
	return nil
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
