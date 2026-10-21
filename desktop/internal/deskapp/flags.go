package deskapp

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
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
		return ModeRun, Flags{}, fmt.Errorf("unexpected argument %q (aurora-desktop takes no positional arguments)", fs.Arg(0))
	}
	if showVersion {
		return ModeVersion, opts, nil
	}
	return ModeRun, opts, nil
}

// Usage writes the flag surface.
func Usage(w io.Writer, version, buildTime string) {
	fmt.Fprintln(w, "Aurora Desktop")
	fmt.Fprintln(w, "  Blockchain toolkit desktop shell")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  aurora-desktop [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -h, --help          Show this help and exit")
	fmt.Fprintln(w, "      --web-root DIR  Serve the web UI from DIR instead of the copy")
	fmt.Fprintln(w, "                      embedded in this binary (for UI development)")
	fmt.Fprintln(w, "      --data-dir DIR  Directory holding aurora.db (default: per-user data dir)")
	fmt.Fprintln(w, "  -v, --version       Show version information and exit")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Version: %s\nBuild time: %s\nGo: %s\n", version, buildTime, runtime.Version())
}

// VersionText renders the version block for --version.
func VersionText(version, buildTime string) string {
	return fmt.Sprintf("Aurora Desktop\nVersion: %s\nBuild time: %s\nGo: %s\n",
		version, buildTime, runtime.Version())
}
