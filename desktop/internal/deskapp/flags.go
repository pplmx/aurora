package deskapp

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"strings"
)

// Mode classifies what the process should do after parsing.
type Mode int

const (
	// ModeRun starts the desktop window.
	ModeRun Mode = iota
	// ModeHelp prints usage and exits 0.
	ModeHelp
	// ModeVersion prints the build identity and exits 0.
	ModeVersion
)

// Flags carries the desktop command line.
type Flags struct {
	WebRoot string
	DataDir string
	// Intent is the page (or lottery export) the launch argument asked for:
	// an aurora:// deep link, or the path of a file the app opens. The zero
	// value means "open the dashboard".
	Intent Intent
}

// ParseFlags parses the desktop binary's flags. Positional arguments are
// rejected: the shell takes none, and silently ignoring them would hide a
// misspelled flag.
func ParseFlags(args []string) (Mode, Flags, error) {
	fs := flag.NewFlagSet("aurora-desktop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts Flags
	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "show version information and exit")
	fs.BoolVar(&showVersion, "v", false, "alias for --version")
	fs.StringVar(&opts.WebRoot, "web-root", "", "serve the web UI from this directory instead of the embedded copy")
	fs.StringVar(&opts.DataDir, "data-dir", "", "directory holding aurora.db (default: per-user data directory)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ModeHelp, Flags{}, nil
		}
		return ModeRun, Flags{}, err
	}
	if fs.NArg() > 0 {
		// A positional argument is either a deep link or a lottery export the
		// operating system handed us (file association), or a mistake. MyGo
		// delivers the same arguments through OnOpenURL/OnOpenFile; parsing
		// them here too means `aurora-desktop aurora://lottery?id=x` and a
		// file-manager double-click both work without a running app.
		if in, ok := ParseIntent(fs.Arg(0)); ok {
			opts.Intent = in
		} else {
			return ModeRun, Flags{}, fmt.Errorf("unexpected argument %q (aurora-desktop takes flags, aurora:// links, or .json lottery exports)", fs.Arg(0))
		}
		if fs.NArg() > 1 {
			return ModeRun, Flags{}, fmt.Errorf("unexpected argument %q (only one link or file can be opened at a time)", fs.Arg(1))
		}
	}
	if showVersion {
		return ModeVersion, opts, nil
	}
	return ModeRun, opts, nil
}

// Usage writes the flag surface.
func Usage(w io.Writer, version, buildTime string) {
	fmt.Fprintln(w, "Aurora Desktop")
	fmt.Fprintln(w, "  "+T("desktop.help.tagline"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, T("desktop.help.usage"))
	fmt.Fprintln(w, "  aurora-desktop [flags] [aurora://link | lottery-export.json]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, T("desktop.help.examples"))
	fmt.Fprintln(w, T("desktop.help.example_dashboard"))
	fmt.Fprintln(w, T("desktop.help.example_link"))
	fmt.Fprintln(w, T("desktop.help.example_file"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, T("desktop.help.flags"))
	fmt.Fprintln(w, T("desktop.help.help_flag"))
	fmt.Fprintln(w, T("desktop.help.web_root_flag"))
	fmt.Fprintln(w, T("desktop.help.web_root_cont"))
	fmt.Fprintln(w, T("desktop.help.data_dir_flag"))
	fmt.Fprintln(w, T("desktop.help.version_flag"))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s: %s\n%s: %s\n%s: %s\n",
		T("app.version"), version, T("app.build_time"), buildTime, T("app.go_version"), runtime.Version())
}

// VersionText renders the version block for --version.
func VersionText(version, buildTime string) string {
	return fmt.Sprintf("Aurora Desktop\n%s: %s\n%s: %s\n%s: %s\n",
		T("app.version"), version, T("app.build_time"), buildTime, T("app.go_version"), runtime.Version())
}

// ForwardedPaths returns the --data-dir and --web-root values of a launch that
// was forwarded to an already running instance (App.OnSecondInstance), or ""
// for one that was not given.
//
// The flags are scanned here instead of being handed to flag.Parse because a
// launch line may put them after the link ("aurora://voting --data-dir X") and
// flag.Parse stops at the first positional argument. The values are always
// well formed: the second process parsed and validated its own command line
// before it forwarded it.
func ForwardedPaths(args []string) (dataDir, webRoot string) {
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || (name != "data-dir" && name != "web-root") {
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				continue
			}
			i++
			value = args[i]
		}
		if name == "data-dir" {
			dataDir = value
		} else {
			webRoot = value
		}
	}
	return dataDir, webRoot
}
