// Command aurora-desktop runs Aurora in a desktop window.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/desktop/internal/deskapp"
)

// Injected at build time by the justfile `desktop` recipe.
var (
	Version   = "0.0.1"
	BuildTime = "unknown"
)

func main() {
	mode, opts, err := deskapp.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, `Run "aurora-desktop --help" for usage.`)
		os.Exit(1)
	}
	switch mode {
	case deskapp.ModeHelp:
		deskapp.Usage(os.Stdout, Version, BuildTime)
		return
	case deskapp.ModeVersion:
		fmt.Print(deskapp.VersionText(Version, BuildTime))
		return
	}

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

	rt, err := deskapp.Start(ctx, deskapp.Options{WebRoot: opts.WebRoot, DataDir: opts.DataDir})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = rt.Close() }()

	// Serve the very same router cmd/api serves, but over the app's own
	// scheme instead of a TCP listener.
	if err := mygo.Protocol.Handle(deskapp.Scheme, rt.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}

	mygo.App.SetName("Aurora")
	mygo.App.SetVersion(Version)
	mygo.App.SetMenu(deskapp.NewMenu(deskapp.About{
		Version:   Version,
		BuildTime: BuildTime,
		WebRoot:   rt.WebRoot(),
		DBPath:    rt.DBPath(),
	}))

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:           "Aurora",
			URL:             deskapp.URL(),
			Width:           1280,
			Height:          860,
			MinWidth:        900,
			MinHeight:       600,
			BackgroundColor: "#1a1a1a",
			StateKey:        "main",
		})
		win.Focus()
	})

	if err := mygo.App.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "aurora-desktop: %v\n", err)
		os.Exit(1)
	}
}
