package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
)

const shutdownTimeout = 15 * time.Second

// Version and BuildTime are overridable at link time, e.g.
//
//	go build -ldflags "-X main.Version=v1.2.3 -X main.BuildTime=2026-09-03T00:00:00Z"
//
// (The root package of a binary links as "main", unlike the CLI's
// cmd/aurora/cmd vars which live in an importable package.) They mirror
// cmd/aurora/cmd.Version/BuildTime so `aurora-api --version` reports the
// exact build the binary carries instead of a placeholder; the justfile `api`
// recipe injects them the same way the CLI's are set (TASK-267, ISS-263).
var (
	Version   = "0.0.1"
	BuildTime = "unknown"
)

// runMode classifies what the process should do after flag parsing, so the
// flag surface stays decoupled from (and testable without) the server boot.
type runMode int

const (
	runServer  runMode = iota // start the HTTP server (the only default)
	runHelp                   // print usage and exit 0
	runVersion                // print version and exit 0
)

// serverOptions carries the listen-address overrides a CLI flag may set.
// The *Set bools distinguish "flag absent (fall back to config)" from an
// explicit value, so the precedence chain runs flag > env > config > default
// (env and defaults are applied by config.Load).
type serverOptions struct {
	host    string
	port    int
	hostSet bool
	portSet bool
}

// parseFlags classifies the command line. Before this guard the server binary
// ignored every flag, so `aurora-api --help` (or any misspelled flag) started
// the HTTP server anyway and only died when the bind failed. Unknown flags
// and stray positional arguments are rejected here, before config loading or
// any listener is created. parseFlags writes nothing itself — the caller
// routes the output to stdout for help/version and stderr for errors.
func parseFlags(args []string) (runMode, *serverOptions, error) {
	fs := flag.NewFlagSet("aurora-api", flag.ContinueOnError)
	// Swallow flag's own error/usage echo; we print exactly once ourselves so
	// help lands on stdout and errors on stderr, never both.
	fs.SetOutput(io.Discard)
	opts := &serverOptions{}
	showVersion := fs.Bool("version", false, "show version information and exit")
	fs.BoolVar(showVersion, "v", *showVersion, "alias for --version")
	fs.StringVar(&opts.host, "host", "", "listen host (overrides config [server].host)")
	fs.StringVar(&opts.host, "H", "", "alias for --host")
	fs.IntVar(&opts.port, "port", 0, "listen port (overrides config [server].port; 1-65535)")
	fs.IntVar(&opts.port, "p", 0, "alias for --port")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return runHelp, nil, nil
		}
		return 0, nil, err
	}
	// Reject stray positionals before the version branch so `--version foo`
	// cannot silently drop the argument: a server binary takes no positional
	// args in any mode (iss 263 / review M1). Help still short-circuits in
	// flag.Parse, so `--help anything` prints help as stdlib convention.
	if fs.NArg() > 0 {
		return 0, nil, fmt.Errorf("unexpected argument %q (aurora-api takes no positional arguments)", fs.Arg(0))
	}
	// Visit marks only flags explicitly present on the command line, so a set
	// option is never mistaken for its zero default (explicit `-p 0` errors
	// below, not silently treats 0 as "unset").
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "host", "H":
			opts.hostSet = true
		case "port", "p":
			opts.portSet = true
		}
	})
	// Version/help are one-shot modes: validate the port only on the server
	// path so `aurora-api --version -p 99999` still prints its version.
	if *showVersion {
		return runVersion, nil, nil
	}
	if opts.portSet && (opts.port < 1 || opts.port > 65535) {
		return 0, nil, fmt.Errorf("invalid port %d (must be 1-65535)", opts.port)
	}
	return runServer, opts, nil
}

// resolveAddr applies the listen-address precedence chain: an explicit
// --host/--port flag wins over the loaded config (whose env binding and
// defaults are applied inside config.Load). Extracted as a pure helper so the
// precedence is unit-testable without binding a socket (ISS-065 style).
func resolveAddr(opts *serverOptions, cfg *config.Config) string {
	host, port := cfg.Server.Host, cfg.Server.Port
	if opts != nil {
		if opts.hostSet {
			host = opts.host
		}
		if opts.portSet {
			port = opts.port
		}
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// printUsage writes the command's flag surface. Kept English: the server has
// no interactive surface and the few lines stay stable; the CLI (cobra) owns
// full localization.
func printUsage(w io.Writer) {
	fmt.Fprintf(w, "%s\n", i18n.GetText("app.name"))
	fmt.Fprintln(w, "  REST API and Web server")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  aurora-api [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -h, --help      Show this help and exit")
	fmt.Fprintln(w, "  -H, --host IP   Listen host (default from config [server].host)")
	fmt.Fprintln(w, "  -p, --port N    Listen port, 1-65535 (default from config [server].port)")
	fmt.Fprintln(w, "  -v, --version   Show version information and exit")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Configuration is loaded with flag > environment > config file >")
	fmt.Fprintln(w, "default precedence (AURORA_SERVER_HOST / AURORA_SERVER_PORT).")
}

// printVersion reports the link-time build identity, mirroring the CLI's
// `aurora version` output shape (i18n labels included).
func printVersion(w io.Writer) {
	fmt.Fprintln(w, i18n.GetText("app.name"))
	fmt.Fprintf(w, "%s: %s\n", i18n.GetText("app.version"), Version)
	fmt.Fprintf(w, "%s: %s\n", i18n.GetText("app.build_time"), BuildTime)
	fmt.Fprintf(w, "%s: %s\n", i18n.GetText("app.go_version"), runtime.Version())
}

// HTTP server timeouts (v1.60). net/http defaults leave ReadHeaderTimeout,
// ReadTimeout, WriteTimeout and IdleTimeout at zero, meaning a connection that
// trickles headers or a body (slowloris / slow-body DoS) can hold a handler
// (and its goroutine) open indefinitely, and idle keep-alive connections are
// never reaped. Keeping them bounded bounds the resources any single client
// can consume; most API operations are fast, so even conservative values are
// not a throughput constraint. ReadHeaderTimeout also independently guards
// against slow-header attacks even before a body is read.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

// newServer creates the HTTP server for the API. Extracted as a pure
// constructor so the timeout configuration is unit-testable (ISS-065,
// TASK-073); it does not Listen, so a test can assert the configured
// timeouts without binding a socket.
func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func main() {
	mode, opts, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, `Run "aurora-api --help" for usage.`)
		os.Exit(1)
	}
	switch mode {
	case runHelp:
		printUsage(os.Stdout)
		os.Exit(0)
	case runVersion:
		printVersion(os.Stdout)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init()

	srv, err := api.NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}
	// Release the SQLite/event-store handles NewServer opened when the server
	// exits (graceful or forced shutdown), matching srv.Close() semantics.
	defer func() { _ = srv.Close() }()

	// Start the oracle fetch scheduler so enabled sources are refreshed on
	// their configured interval (v1.15 Oracle Scheduled Fetching). Cancelled
	// at shutdown.
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	stopScheduler := srv.StartOracleScheduler(schedCtx, config.OracleSchedulerCheckInterval())
	if stopScheduler != nil {
		defer stopScheduler()
	}

	// Start the audit outbox drainer: token audit events whose direct publish
	// hit a transient failure were parked in pending_events by the audit
	// handler (TASK-119, ISS-111) — this loop retries them until they land.
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	defer outboxCancel()
	stopOutbox := srv.StartAuditOutboxDrainer(outboxCtx)
	if stopOutbox != nil {
		defer stopOutbox()
	}

	router := srv.Router()

	addr := resolveAddr(opts, cfg)
	server := newServer(addr, router)

	go func() {
		logger.Info().Str("addr", addr).Msg("Starting API server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// logger.Fatal exits without running main's deferred srv.Close(),
			// so release the SQLite/event-store handles here first — otherwise
			// the WAL is left uncleaned and the outbox/scheduler goroutines are
			// torn down over an open DB. Idempotent if srv.Close ran already.
			_ = srv.Close()
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info().Str("signal", sig.String()).Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
		if err := server.Close(); err != nil {
			logger.Error().Err(err).Msg("Server force close error")
		}
	}

	logger.Info().Msg("Server stopped")
}
