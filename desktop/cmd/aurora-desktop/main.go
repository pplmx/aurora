// Command aurora-desktop runs Aurora in a desktop window.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	// MyGo's own strings (menu roles, native dialog buttons) in Chinese. The
	// app declares the languages below with SetLanguages, and without the
	// catalog imported MyGo would fall back to English and log a warning.
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
	"github.com/egoist/mygo/ui"
	"github.com/pplmx/aurora/desktop/appicon"
	"github.com/pplmx/aurora/desktop/internal/deskapp"
	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// generationRequested reports whether MyGo compiled this binary only to
// extract a TypeScript client. Such a run must not start the shell: Aurora
// binds no MyGo services, so there is no client to write, and booting the
// database and HTTP router would make a package build depend on a writable
// user-data directory.
func generationRequested() bool { return os.Getenv("MYGO_GENERATE") != "" }

// Injected at build time by the justfile `desktop` recipe. A `mygo build`
// bundle links its own packageVersion instead, so version() prefers that and
// falls back to these for `go build`/`just desktop`.
var (
	Version   = "0.0.1"
	BuildTime = unknownBuildTime
)

const unknownBuildTime = "unknown"

// version is the version a user sees. MyGo knows the version linked into a
// packaged app (mygo.config.ts), which the justfile's -X main.Version cannot
// reach: `mygo build` supplies its own -ldflags. Without this fallback a
// .deb built from the same source reported 0.0.1 (measured), so
// `aurora-desktop --version` disagreed with the installer's package version.
func version() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return Version
}

// buildTime is the build timestamp a user sees. `mygo build` supplies its own
// -ldflags, so the justfile's -X main.BuildTime never lands in a packaged
// binary. mygo.config.ts runs scripts/stamp-build-time.mjs before MyGo copies
// resources/, so a package carries the time it was built next to the
// executable. The executable's timestamp remains a last fallback for a plain
// `go build` without ldflags.
func buildTime() string {
	return resolveBuildTime(BuildTime, stampedBuildTime, os.Executable, os.Stat)
}

// resolveBuildTime is buildTime with its inputs as parameters, so tests can
// exercise the fallback without replacing the test binary.
func resolveBuildTime(
	linked string,
	stamped func() (string, error),
	executable func() (string, error),
	stat func(string) (fs.FileInfo, error),
) string {
	if linked != "" && linked != unknownBuildTime {
		return linked
	}
	if stamped != nil {
		if value, err := stamped(); err == nil && value != "" {
			return value
		}
	}
	exe, err := executable()
	if err != nil {
		return unknownBuildTime
	}
	info, err := stat(exe)
	if err != nil || info.ModTime().IsZero() {
		return unknownBuildTime
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}

// stampedBuildTime reads the timestamp written by
// scripts/stamp-build-time.mjs into the app's runtime resources.
func stampedBuildTime() (string, error) {
	dir, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "build-time.txt"))
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	built, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", err
	}
	return built.UTC().Format(time.RFC3339), nil
}

// userDataDir is the default data directory for a packaged app: MyGo's
// per-user application directory, next to its window state. Outside a package
// (go run, go test, `just desktop`) MyGo still answers with the same XDG /
// AppData location, so the database does not move between development and a
// bundle.
func userDataDir() (string, error) {
	return mygo.App.Path(mygo.PathUserData)
}

// startDesktop performs the checks that must happen before any durable state
// is created. The webview check is first: a packaged app launched without
// WebKitGTK must report that fact without creating a database or log file.
func startDesktop(webViewAvailable func() bool, start func() error) error {
	if !webViewAvailable() {
		return errors.New(deskapp.T("desktop.error.no_webview"))
	}
	return start()
}

func main() {
	// `mygo build` runs the compiled app once with MYGO_GENERATE to extract a
	// TypeScript client. Aurora binds no MyGo services, so there is nothing to
	// generate, and starting the whole shell first would create a database and
	// logs merely to write an empty client. App.Run already returns without
	// starting the event loop in this mode.
	if generationRequested() {
		if err := mygo.App.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Aurora's own strings (menus, dialogs, --help) go through internal/i18n,
	// which has to be initialized before the first one is read: MyGo picks its
	// own language from the user's, and FollowAppLocale below pins Aurora's
	// translator to it so one window does not mix two languages.
	deskapp.InitLocale()

	mode, opts, err := deskapp.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, deskapp.T("desktop.help.run_help"))
		os.Exit(1)
	}
	switch mode {
	case deskapp.ModeHelp:
		deskapp.Usage(os.Stdout, version(), buildTime())
		return
	case deskapp.ModeVersion:
		fmt.Print(deskapp.VersionText(version(), buildTime()))
		return
	}
	// Declare the languages this build ships - MyGo's own strings and Aurora's
	// both - before anything can show one. MyGo translates its menu roles and
	// the native dialog buttons (OK/Cancel) in the language the app shows, and
	// can only pick Chinese if it knows the app is written in it. Setting this
	// after Start left the startup-failure dialog - the one dialog a broken
	// start shows - with an English "OK" and English menu roles, because a
	// failed Start returns before the call was reached.
	//
	// Both Chinese catalogs are declared so a zh-TW/zh-HK reader keeps
	// Chinese: MyGo answers "zh-Hant" for those and says its own roles in
	// Traditional, while Aurora's strings come from internal/i18n, which has
	// one Chinese table (Simplified). Declaring Simplified only made MyGo
	// match nothing and answer "en", so the whole window went English.
	mygo.App.SetLanguages("en", "zh-Hans", "zh-Hant")

	// Start with the previous dark default until the saved preference is read
	// below. This avoids a white native frame flashing around a dark window
	// before the first frame. The view sets its own palette on top of this.
	//
	// This asks GTK for its dark variant. A theme without one — plain Adwaita,
	// which is what a bare X server gets — has nothing to switch to, so the
	// menu bar there stays light however this is set; measured by forcing
	// gtk-application-prefer-dark-theme with and without a themed variant.
	mygo.Theme.SetSource(mygo.ThemeDark)

	// Name the app before ANYTHING asks MyGo about identity. Two things depend
	// on it and would otherwise disagree between runs:
	//   - App.Path(PathUserData) joins the application name, so resolving the
	//     data directory before SetName created ~/.config/MyGo on one run and
	//     ~/.config/Aurora on the next (measured);
	//   - the single-instance lock is keyed by the name, so two launches that
	//     named themselves differently would both believe they were first.
	mygo.App.SetName(deskapp.AppName)

	// One desktop instance per user: the database is single-writer (SQLite)
	// and the shell owns the window, so a second launch focuses the first.
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Let MyGo own the per-user directory when the app is packaged: the
	// database, the logs and MyGo's window-state.json then live under one
	// application directory (and the name MyGo already resolved), instead of
	// our own guess next to it.
	var rt *deskapp.Runtime
	err = startDesktop(
		mygo.WebViewAvailable,
		func() (err error) {
			rt, err = deskapp.Start(ctx, deskapp.Options{
				DataDir:        opts.DataDir,
				DataDirDefault: userDataDir,
			})
			return err
		},
	)
	if err != nil {
		failStartup(err)
	}
	defer func() { _ = rt.Close() }()
	// Start loaded the config; a configured [i18n] locale now applies to the
	// window too, not only to the CLI of the same build.
	deskapp.ApplyConfiguredLocale()

	// Only override the version `mygo build` linked in; SetVersion("") would
	// erase it (App.Version falls back to the packaged identity only when no
	// override is set).
	if mygo.App.Version() == "" {
		mygo.App.SetVersion(Version)
	}
	meta := deskapp.About{
		Version:   version(),
		BuildTime: buildTime(),
		DBPath:    rt.DBPath(),
	}

	// Operating-system integration. On macOS the scheme and file association
	// come from mygo.config.ts at packaging time; on Linux and Windows the
	// app registers the scheme on every start, which is cheap and works
	// without an installer.
	shell := deskapp.NewShell(meta)
	shell.ApplyTheme()
	// Window writes leave the frame thread so the busy banner is painted and
	// the window stays responsive during SQLite/file work. Tests keep the
	// synchronous path for deterministic assertions.
	rt.App().SetBackgroundWrites(true)
	// File import/export is a one-shot action, but it should use the runtime's
	// repository rather than open a second SQLite writer beside it.
	shell.SetLotteryRepository(rt.Services().LotteryRepository())
	shell.SetBackup(rt.Services().Backup)
	// The window draws native widgets from this state; a link or a second
	// launch steers it, so the shell needs it before any intent arrives.
	shell.SetApp(rt.App())
	// The menu bar is hidden by default now, so the application work that is
	// not page work lives in the sidebar's File menu and Settings page: same shell
	// calls as the native File/Help items, with the live window as dialog parent.
	rt.App().SetActions(nativeui.Actions{
		BackUpDatabase:  shell.BackUpDatabase,
		ExportLotteries: shell.ExportLotteries,
		ImportLotteries: shell.ImportLotteries,
		ShowDataFolder:  shell.ShowDataFolder,
		ShowAbout:       shell.ShowAbout,
		RecentDocuments: shell.RecentDocuments,
		OpenRecent:      shell.OpenRecent,
		SetLanguage:     shell.SetLanguage,
		Language:        shell.Language,
		LiveUpdates:     shell.LiveUpdates,
		SetLiveUpdates:  shell.SetLiveUpdates,
		ThemeMode:       shell.ThemeMode,
		SetThemeMode:    shell.SetThemeMode,
		SystemDark:      shell.SystemDark,
		SetPageTitle:    shell.SetPageTitle,
		Version:         meta.Version,
		BuildTime:       meta.BuildTime,
		DBPath:          meta.DBPath,
		DatabaseSize:    func() string { return deskapp.DatabaseSizeText(meta.DBPath) },
	})
	// Keep the listener for the whole process, not only for WhenReady: a
	// deferred unregister inside that callback would make the next OS
	// appearance change invisible to the native window.
	offTheme := mygo.Theme.OnUpdated(func() {
		if app := shell.App(); app != nil {
			app.SetSystemDark(mygo.Theme.IsDark())
			if win := shell.Window(); win != nil && !win.IsDestroyed() {
				win.Invalidate()
			}
		}
	})
	defer offTheme()
	// macOS owns a global menu bar; give it only the platform's roles (app,
	// File/Close, Edit, Window) because Aurora's own actions live in the
	// window's File menu and Settings page on every platform. Linux and Windows
	// install no application menu at all: the sidebar owns navigation and the
	// window owns file/settings work, so an empty bar would have nothing useful.
	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(deskapp.NewPlatformMenu())
	}
	if runtime.GOOS != "darwin" {
		if err := mygo.App.RegisterURLScheme(deskapp.DeepLinkScheme); err != nil {
			fmt.Fprintf(os.Stderr, "aurora-desktop: register %s://: %v\n", deskapp.DeepLinkScheme, err)
		}
	}
	// A launch argument is acted on twice on a cold start: ParseFlags read it
	// from our own command line, and MyGo re-delivers it through
	// OnOpenURL/OnOpenFile once the app is ready. A link is consumed here
	// because this process applies it itself in WhenReady, below; MyGo's echo
	// of the same link is then dropped by Shell.take instead of navigating
	// twice.
	//
	// A file is deliberately NOT consumed: MyGo never echoes one (Aurora
	// declares no file association), and ImportRecords already refuses the
	// duplicate record, so the guard would only block the operator's next
	// genuine import of that file.
	if opts.Intent.File == "" && opts.Intent.Raw != "" {
		shell.Consume(opts.Intent.Raw)
	}
	mygo.App.OnOpenURL(shell.OpenURL)
	mygo.App.OnOpenFile(shell.OpenFile)
	// A second launch with another link or file must reach this instance
	// rather than start a rival database writer. MyGo hands us the raw
	// arguments, and only turns them into OnOpenURL/OnOpenFile for schemes and
	// file types it is configured to handle; an export is not one of those
	// (aurora claims no .json association on purpose), so the shell parses the
	// arguments itself.
	mygo.App.OnSecondInstance(func(args []string, workDir string) {
		shell.HandleLaunchArgs(args, workDir)
	})

	// A macOS app outlives its window: Cmd+W closes the window, the app stays
	// in the Dock, and clicking it there (OnActivate with no visible windows)
	// must bring the window back. Without these two listeners MyGo's defaults
	// apply - closing the last window quits everywhere - so on macOS a closed
	// window was the end of the app, and a Dock click had no way back.
	//
	// Elsewhere the default is what users expect: closing the window quits,
	// and the single-instance lock exits the process with it.
	mygo.App.OnWindowAllClosed(func() {
		if quitsOnLastWindowClosed(runtime.GOOS) {
			mygo.App.Quit()
		}
	})
	shell.SetWindowFactory(func() *mygo.Window { return openMainWindow(shell, opts) })
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			shell.EnsureWindow()
		}
	})

	mygo.App.WhenReady(func() {
		// SetActions ran before App.Run and could only use the explicit source
		// or the previous dark default for Follow System. Resolve the real
		// desktop appearance now, before the first window draws.
		rt.App().SetSystemDark(mygo.Theme.IsDark())
		deskapp.FollowAppLocale(func() {
			// MyGo relabels the platform roles itself; Aurora text in the
			// window reads the translator on the next frame. The native title
			// is outside that frame, so refresh it explicitly.
			shell.SetPageTitle(rt.App().PageTitle())
			if win := shell.Window(); win != nil && !win.IsDestroyed() {
				win.Invalidate()
			}
		})
		openMainWindow(shell, opts)
		// The launch's own request is applied now that there is a window: a
		// deep link selects its page (and verifies the draw it names), and an
		// export is imported. Both are already consumed, so MyGo's own
		// delivery of the same argument is a no-op and each happens once.
		//
		// Applying the link only through MyGo's delivery is not enough: that
		// path exists on Linux and Windows, and a macOS launch that names a
		// draw reached an app that never opened the page.
		if opts.Intent.Raw != "" && opts.Intent.File == "" {
			shell.Navigate(opts.Intent)
		}
		if opts.Intent.File != "" {
			shell.OpenFile(opts.Intent.File)
		}
	})

	if err := mygo.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}
}

// quitsOnLastWindowClosed reports whether closing the last window should end
// the app.
//
// On macOS it should not: the app stays in the Dock and OnActivate brings the
// window back, which is what the platform's users expect from Cmd+W. Every
// other platform quits with its window. MyGo applies the same split to its own
// default menu, so an app that does nothing here would quit on macOS too -
// exactly the behavior this function exists to prevent.
func quitsOnLastWindowClosed(goos string) bool {
	return goos != "darwin"
}

// mainWindowHooks are the per-window bindings that must survive recreation.
// Keeping them behind an interface lets the tests exercise the wiring without
// a native MyGo window.
type mainWindowHooks interface {
	SetIcon([]byte) error
	OnFileDrop(func(*mygo.FileDropEvent)) func()
	Focus()
}

// mainWindowOptions is the one specification for the main window. Startup and
// macOS window recreation both go through it, so a reopened window keeps the
// same state key and appearance as the first one.
//
// Content replaces URL: the window draws MyGo's own widgets instead of hosting
// the REST gateway's HTML. There is no web page, no browser engine and no
// Content-Security-Policy to get wrong.
func mainWindowOptions(app *nativeui.App, opts deskapp.Flags) mygo.WindowOptions {
	return mygo.WindowOptions{
		Title:           deskapp.AppName,
		Content:         ui.View(app.View),
		Width:           1280,
		Height:          860,
		MinWidth:        900,
		MinHeight:       600,
		BackgroundColor: "light-dark(#F4F0E6, #0A1211)",
		StateKey:        "main",
		// Aurora installs no application menu on Linux and Windows: pages live
		// in the sidebar, file work in the File menu and configuration in
		// Settings. AutoHideMenuBar is the fallback for a menu a host or toolkit
		// installs anyway, so it cannot reappear as a permanent strip; macOS
		// keeps its application menu because the platform owns that bar.
		AutoHideMenuBar: true,
	}
}

// wireMainWindow installs the bindings a window needs regardless of whether
// it is the first window or a replacement. The application menu is not
// per-window on macOS; MyGo installs the existing application menu on each
// new window on Linux and Windows, so recreating a window must not replace or
// clear it.
func wireMainWindow(shell *deskapp.Shell, win *mygo.Window, hooks mainWindowHooks) {
	// A packaged build takes the icon from appicon/icon.png at build time;
	// an unpackaged one would fall back to the toolkit's placeholder in its
	// title bar and taskbar, so set the very same image here.
	if err := hooks.SetIcon(appicon.IconPNG); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: set window icon: %v\n", err)
	}
	shell.Attach(win)
	if app := shell.App(); app != nil {
		shell.SetPageTitle(app.PageTitle())
	}
	// Dropping a lottery export on the window imports it, the same way opening
	// one from the file manager does. Anything else is reported rather than
	// silently ignored.
	hooks.OnFileDrop(func(e *mygo.FileDropEvent) { shell.ImportDropped(e.Paths) })
	hooks.Focus()
}

type mainWindowConstructor func(mygo.WindowOptions) (*mygo.Window, mainWindowHooks)

// buildMainWindow applies the shared specification and wiring through an
// injectable constructor, so a test can prove that a replacement window gets
// the same state and handlers as the first.
func buildMainWindow(shell *deskapp.Shell, opts deskapp.Flags, create mainWindowConstructor) *mygo.Window {
	win, hooks := create(mainWindowOptions(shell.App(), opts))
	wireMainWindow(shell, win, hooks)
	return win
}

// openMainWindow creates the app's window and wires it to the shell. It runs
// both at startup and when a macOS Dock click reopens a closed window, so the
// two paths cannot produce differently-behaved windows (a missing file-drop
// handler on the reopened one, for instance).
func openMainWindow(shell *deskapp.Shell, opts deskapp.Flags) *mygo.Window {
	return buildMainWindow(shell, opts, func(options mygo.WindowOptions) (*mygo.Window, mainWindowHooks) {
		win := mygo.NewWindow(options)
		return win, win
	})
}

// failStartup reports a failure to start and exits 1.
//
// Printing to stderr is not enough: a packaged app is launched from a desktop
// entry or a file manager, which gives it no console at all, so a bad
// --data-dir or an unusable data directory looked exactly like a double-click
// that did nothing. The app is run just long enough to show the reason in a
// native dialog. A dialog needs the running app (MyGo panics on one before
// Run), which is why this starts the event loop instead of calling
// Dialog.Message directly.
func failStartup(err error) {
	fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
	mygo.App.WhenReady(func() {
		_, _ = mygo.Dialog.Message(startupFailureDialog(err))
		mygo.App.Quit()
	})
	if runErr := mygo.App.Run(); runErr != nil {
		// No display, no webview, or the loop died before the dialog: the
		// stderr line above is then the only report there can be.
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", runErr)
	}
	os.Exit(1)
}

// startupFailureDialog is what a failed start tells the user. The reason is in
// Detail so it is readable in full (a data-directory path can be long) and the
// message states the consequence, because Linux shows the body without the
// title.
func startupFailureDialog(err error) mygo.MessageOptions {
	return mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   deskapp.T("desktop.startup.title"),
		Message: deskapp.T("desktop.startup.message"),
		Detail:  err.Error(),
	}
}
