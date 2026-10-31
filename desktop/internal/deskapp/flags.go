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
	DataDir string
	// Intent is the page (or lottery export) the launch argument asked for:
	// an aurora:// deep link, or the path of a file the app opens. The zero
	// value means "open the dashboard".
	Intent Intent
}

// ParseFlags parses the desktop binary's flags and its one optional launch
// intent. Startup flags may appear before or after the link or export; any
// other positional is rejected, and silently ignoring one would hide a
// misspelled flag.
func ParseFlags(args []string) (Mode, Flags, error) {
	fs := flag.NewFlagSet("aurora-desktop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts Flags
	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "show version information and exit")
	fs.BoolVar(&showVersion, "v", false, "alias for --version")
	fs.StringVar(&opts.DataDir, "data-dir", "", "directory holding aurora.db (default: per-user data directory)")

	// flag.Parse stops at the first positional argument, but a launcher may
	// put startup flags on either side of the deep link or export. That is
	// also the grammar ForwardedPaths handles for a second launch, so both
	// paths must accept the same command lines. Collect the one allowed
	// positional and parse the flags around it. A flag that takes a value
	// consumes the following argument even when it is spelled after the
	// intent, so a missing value still reports the same error as flag.Parse.
	// "--" keeps its meaning: it ends flags, and everything after it is a
	// positional intent rather than a startup option.
	var intentArg string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			// Keep flag.Parse's explicit end-of-flags behavior. Any
			// positional after it is still an intent, but a literal flag
			// spelling after it must be rejected as a positional, not
			// reinterpreted as a startup option.
			flagArgs = append(flagArgs, arg)
			for _, rest := range args[i+1:] {
				if intentArg != "" {
					return ModeRun, Flags{}, fmt.Errorf(T("desktop.error.extra_arg"), rest)
				}
				intentArg = rest
			}
			break
		}
		if arg == "-" {
			return ModeRun, Flags{}, fmt.Errorf(T("desktop.error.unexpected_arg"), arg)
		}
		if !strings.HasPrefix(arg, "-") {
			if intentArg != "" {
				return ModeRun, Flags{}, fmt.Errorf(T("desktop.error.extra_arg"), arg)
			}
			intentArg = arg
			continue
		}
		flagArgs = append(flagArgs, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if isValueFlag(name) && !hasValue && i+1 < len(args) {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ModeHelp, Flags{}, nil
		}
		return ModeRun, Flags{}, err
	}
	if intentArg != "" {
		// A positional argument is either a deep link or a lottery export the
		// operating system handed us (file association), or a mistake. MyGo
		// delivers the same arguments through OnOpenURL/OnOpenFile; parsing
		// them here too means `aurora-desktop aurora://lottery?id=x` and a
		// file-manager double-click both work without a running app.
		in, ok := ParseIntent(intentArg)
		if !ok {
			return ModeRun, Flags{}, fmt.Errorf(T("desktop.error.unexpected_arg"), intentArg)
		}
		opts.Intent = in
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
	fmt.Fprintln(w, "  aurora-desktop [flags] [aurora://link | lottery-export.json] [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, T("desktop.help.examples"))
	fmt.Fprintln(w, T("desktop.help.example_dashboard"))
	fmt.Fprintln(w, T("desktop.help.example_link"))
	fmt.Fprintln(w, T("desktop.help.example_file"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, T("desktop.help.flags"))
	fmt.Fprintln(w, T("desktop.help.help_flag"))
	fmt.Fprintln(w, T("desktop.help.data_dir_flag"))
	fmt.Fprintln(w, T("desktop.help.version_flag"))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s: %s\n%s: %s\n%s: %s\n",
		T("app.version"), version, T("app.build_time"), buildTime, T("app.go_version"), runtime.Version())
}

// VersionText renders the version block for --version.
func VersionText(version, buildTime string) string {
	text := fmt.Sprintf("Aurora Desktop\n%s: %s\n%s: %s\n%s: %s\n",
		T("app.version"), version, T("app.build_time"), buildTime, T("app.go_version"), runtime.Version())
	// A `go run` build carries no module list, so the MyGo half is omitted rather
	// than printed empty.
	if mygo := mygoVersion(); mygo != "" {
		text += fmt.Sprintf("%s: %s\n", T("app.mygo_version"), mygo)
	}
	return text
}

// ForwardedPaths returns the --data-dir value of a launch that was forwarded to
// an already running instance (App.OnSecondInstance), or "" for one that was
// not given.
//
// The flags are scanned here instead of being handed to flag.Parse because a
// launch line may put them after the link ("aurora://voting --data-dir X") and
// flag.Parse stops at the first positional argument. The values are always
// well formed: the second process parsed and validated its own command line
// before it forwarded it.
func ForwardedPaths(args []string) (dataDir string) {
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || !isValueFlag(name) {
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
		}
	}
	return dataDir
}

// isValueFlag reports whether name takes a separate argument. The parser
// needs this when flags surround the intent: it must keep a value attached to
// its flag instead of mistaking that value for the intent.
func isValueFlag(name string) bool {
	return name == "data-dir"
}
