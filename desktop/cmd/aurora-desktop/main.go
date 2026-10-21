// Command aurora-desktop runs Aurora in a desktop window.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/desktop/internal/deskapp"
	"github.com/pplmx/aurora/desktop/resources"
)

// Injected at build time by the justfile `desktop` recipe. A `mygo build`
// bundle links its own packageVersion instead, so version() prefers that and
// falls back to these for `go build`/`just desktop`.
var (
	Version   = "0.0.1"
	BuildTime = "unknown"
)

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

// userDataDir is the default data directory for a packaged app: MyGo's
// per-user application directory, next to its window state. Outside a package
// (go run, go test, `just desktop`) MyGo still answers with the same XDG /
// AppData location, so the database does not move between development and a
// bundle.
func userDataDir() (string, error) {
	return mygo.App.Path(mygo.PathUserData)
}

func main() {
	mode, opts, err := deskapp.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, `Run "aurora-desktop --help" for usage.`)
		os.Exit(1)
	}
	switch mode {
	case deskapp.ModeHelp:
		deskapp.Usage(os.Stdout, version(), BuildTime)
		return
	case deskapp.ModeVersion:
		fmt.Print(deskapp.VersionText(version(), BuildTime))
		return
	}

	// Name the app before ANYTHING asks MyGo about identity. Two things depend
	// on it and would otherwise disagree between runs:
	//   - App.Path(PathUserData) joins the application name, so resolving the
	//     data directory before SetName created ~/.config/MyGo on one run and
	//     ~/.config/Aurora on the next (measured);
	//   - the single-instance lock is keyed by the name, so two launches that
	//     named themselves differently would both believe they were first.
	mygo.App.SetName(deskapp.AppName)

	// Refuse to start without a system webview BEFORE booting the window:
	// NewWindow panics when the platform has none, and a raw panic with a Go
	// stack trace is useless to a user who is missing a system package.
	if !mygo.WebViewAvailable() {
		fmt.Fprintln(os.Stderr, "aurora-desktop: no system webview available; "+
			"install WebKitGTK (libwebkit2gtk-4.1-0 on Debian/Ubuntu) to show the UI")
		os.Exit(1)
	}

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
	rt, err := deskapp.Start(ctx, deskapp.Options{
		WebRoot:        opts.WebRoot,
		DataDir:        opts.DataDir,
		DataDirDefault: userDataDir,
	})
	if err != nil {
		failStartup(err)
	}
	defer func() { _ = rt.Close() }()

	// Serve the very same router cmd/api serves, but over the app's own
	// scheme instead of a TCP listener.
	if err := mygo.Protocol.Handle(deskapp.Scheme, rt.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}

	// Only override the version `mygo build` linked in; SetVersion("") would
	// erase it (App.Version falls back to the packaged identity only when no
	// override is set).
	if mygo.App.Version() == "" {
		mygo.App.SetVersion(Version)
	}
	meta := deskapp.About{
		Version:   version(),
		BuildTime: BuildTime,
		WebRoot:   rt.WebRoot(),
		DBPath:    rt.DBPath(),
	}
	mygo.App.SetMenu(deskapp.NewMenu(meta))

	// Operating-system integration. On macOS the scheme and file association
	// come from mygo.config.ts at packaging time; on Linux and Windows the
	// app registers the scheme on every start, which is cheap and works
	// without an installer.
	shell := deskapp.NewShell(meta)
	if runtime.GOOS != "darwin" {
		if err := mygo.App.RegisterURLScheme(deskapp.DeepLinkScheme); err != nil {
			fmt.Fprintf(os.Stderr, "aurora-desktop: register %s://: %v\n", deskapp.DeepLinkScheme, err)
		}
	}
	// A deep link is re-delivered by MyGo once the app is ready, but the
	// window already opens at opts.Intent.URL(); mark it handled so the
	// delivery does not reload the page. A file is NOT marked here: the import
	// needs a running app, so it happens in WhenReady, and Shell.take makes
	// whichever of that call and MyGo's delivery arrives first the only one
	// that imports.
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

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:           deskapp.AppName,
			URL:             opts.Intent.URL(),
			Width:           1280,
			Height:          860,
			MinWidth:        900,
			MinHeight:       600,
			BackgroundColor: "#1a1a1a",
			StateKey:        "main",
		})
		// A packaged build takes the icon from resources/icon.png at build
		// time; an unpackaged one would fall back to the toolkit's placeholder
		// in its title bar and taskbar, so set the very same image here.
		if err := win.SetIcon(resources.IconPNG); err != nil {
			fmt.Fprintf(os.Stderr, "aurora-desktop: set window icon: %v\n", err)
		}
		shell.Attach(win)
		// Dropping a lottery export on the window imports it, the same way
		// opening one from the file manager does. Anything else is reported
		// rather than silently ignored.
		win.OnFileDrop(func(e *mygo.FileDropEvent) { shell.ImportDropped(e.Paths) })
		win.Focus()
		// A lottery export named on the command line is imported once the app
		// is ready: the storage layer is already open, and the result dialog
		// MyGo shows needs the running app (calling it before Run panics).
		// The path is already consumed, so MyGo's own delivery of it is a
		// no-op and the file is imported exactly once.
		if opts.Intent.File != "" {
			shell.OpenFile(opts.Intent.File)
		}
	})

	if err := mygo.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}
}

// failStartup reports a failure to start and exits 1.
//
// Printing to stderr is not enough: a packaged app is launched from a desktop
// entry or a file manager, which gives it no console at all, so a bad
// --web-root or an unusable data directory looked exactly like a double-click
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
// Detail so it is readable in full (a web root path can be long) and the
// message states the consequence, because Linux shows the body without the
// title.
func startupFailureDialog(err error) mygo.MessageOptions {
	return mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   "Aurora could not start",
		Message: "Aurora did not open its window.",
		Detail:  err.Error(),
	}
}
