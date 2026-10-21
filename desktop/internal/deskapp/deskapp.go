// Package deskapp hosts Aurora in a MyGo desktop window.
//
// The window does not talk to a listening HTTP server: the app's own router is
// served over the "aurora" URL scheme, so the page and the REST API share one
// origin. That keeps the API key off the network entirely — the same
// injectAPIKey gateway embeds it into the served HTML, and every fetch stays
// inside the process.
package deskapp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	aurora "github.com/pplmx/aurora"
	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/viper"
)

// Scheme is the custom URL scheme the window loads the UI from. MyGo resolves
// aurora://localhost/... through the handler registered with Protocol.Handle.
const Scheme = "aurora"

// AppName is the name the app shows and the identity MyGo derives its
// per-user directory and single-instance lock from, so it must match `name`
// in desktop/mygo.config.ts.
const AppName = "Aurora"

// Options carries the desktop-only overrides. Empty fields fall back to the
// same precedence the CLI and API server use.
type Options struct {
	// WebRoot is the directory holding index.html and the assets. When set it
	// wins over the embedded UI, for developing the pages without a rebuild.
	// When empty the shell serves the UI embedded in the binary.
	WebRoot string
	// DataDir holds the SQLite database. When empty the app uses the platform
	// user-data directory so a packaged app never writes into its install dir.
	DataDir string
	// DataDirDefault, when set, resolves that user-data directory. MyGo's
	// App.Path(PathUserData) is the default so a `mygo build` bundle keeps
	// the database, the logs and MyGo's own window-state.json in one
	// application directory; nil keeps platformUserDataDir, which is what
	// tests and `go run` use.
	DataDirDefault func() (string, error)
}

// Runtime is a started in-process Aurora server plus stop hooks for its
// background loops. Close releases the SQLite handles.
type Runtime struct {
	server        *api.Server
	webRoot       string
	webFS         fs.FS
	dbPath        string
	stopScheduler func()
	stopOutbox    func()
}

// Handler returns the router the window's scheme serves.
//
// The router keeps its own CSP (the HTTP gateway's strict same-origin policy),
// so this wrapper strips the header for the custom-scheme transport.
//
// Why stripping is necessary, and why the policy is not simply widened: on
// aurora://localhost/ WebKitGTK refuses to run the page's own scripts under any
// CSP source expression. Measured with the shipped UI, every one of
// `'self' aurora:`, `'self' aurora://localhost`, and even `*` left the window
// inert — no x-show evaluation and zero /api/v1 requests — while removing the
// header made all six dashboard calls fire. The policy's job here is done by
// the transport instead: the scheme handler serves only this app's embedded
// files, there is no network listener, and no remote origin is reachable, so
// there is no cross-origin script or exfiltration channel for a CSP to close.
// The HTTP gateway keeps its strict policy untouched.
func (r *Runtime) Handler() http.Handler {
	webFS := r.webFS
	if webFS == nil {
		webFS = os.DirFS(r.webRoot)
	}
	return stripCSP(r.server.RouterWithWebFS(webFS))
}

// stripCSP removes Content-Security-Policy from every response. See Handler.
func stripCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&cspDroppingWriter{ResponseWriter: w}, r)
	})
}

// cspDroppingWriter drops CSP from both Header() writes and the WriteHeader
// call, because injectAPIKey sets it through Header() after the downstream
// handler has already written its own headers.
//
// The optional interfaces are forwarded explicitly. This wrapper is the
// outermost one around every desktop response, and Go's net/http adapters do
// not see through a wrapper that lacks them: without Flush a streaming
// handler (the API's own recovery middleware promises to preserve
// http.Flusher) would have its flushes silently dropped on the custom scheme,
// and without Unwrap an http.ResponseController would report "feature not
// supported" instead of reaching the writer underneath.
type cspDroppingWriter struct {
	http.ResponseWriter
}

func (w *cspDroppingWriter) WriteHeader(status int) {
	w.ResponseWriter.Header().Del("Content-Security-Policy")
	w.ResponseWriter.WriteHeader(status)
}

func (w *cspDroppingWriter) Write(p []byte) (int, error) {
	// A handler that writes without an explicit WriteHeader triggers an
	// implicit 200; drop the header then too.
	w.ResponseWriter.Header().Del("Content-Security-Policy")
	return w.ResponseWriter.Write(p)
}

// Unwrap lets http.ResponseController find the writer underneath for the
// features this wrapper does not implement itself (deadlines, full duplex).
func (w *cspDroppingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Flush implements http.Flusher so a streamed response is streamed.
func (w *cspDroppingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		w.ResponseWriter.Header().Del("Content-Security-Policy")
		f.Flush()
	}
}

// Hijack implements http.Hijacker for handlers that take the connection.
func (w *cspDroppingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("deskapp: underlying writer is not an http.Hijacker")
	}
	return h.Hijack()
}

// WebRoot reports the directory the UI is served from, or "" when the UI is
// the one embedded in the binary.
func (r *Runtime) WebRoot() string { return r.webRoot }

// DBPath reports the SQLite file the app is using.
func (r *Runtime) DBPath() string { return r.dbPath }

// Close stops the background loops and releases every SQLite handle.
func (r *Runtime) Close() error {
	if r.stopScheduler != nil {
		r.stopScheduler()
	}
	if r.stopOutbox != nil {
		r.stopOutbox()
	}
	return r.server.Close()
}

// Start loads the configuration, points it at the desktop paths and boots the
// API server with the oracle scheduler and audit-outbox drainer running, the
// same background work cmd/api starts.
func Start(ctx context.Context, opts Options) (*Runtime, error) {
	if _, err := config.Load(); err != nil {
		return nil, err
	}

	// The default is the UI embedded in the binary: a double-clicked bundle
	// has no checkout, so resolving web/ from disk would fail for every user
	// who did not happen to launch from the repository. An explicit
	// --web-root (or a configured server.webRoot) still serves the pages from
	// disk, which is what developing the UI needs.
	var webFS fs.FS
	var webRoot string
	if opts.WebRoot != "" {
		root, err := ResolveWebRoot(opts.WebRoot, config.WebRoot())
		if err != nil {
			return nil, err
		}
		webRoot = root
		viper.Set("server.webRoot", root)
	} else {
		embedded, err := aurora.WebFS()
		if err != nil {
			return nil, fmt.Errorf(T("desktop.error.embedded_assets"), err)
		}
		webFS = embedded
	}

	dbPath, err := prepareDataDir(opts.DataDir, opts.DataDirDefault)
	if err != nil {
		return nil, err
	}
	viper.Set("db.path", dbPath)

	// logger.Init reads viper's log.path, whose default is the CWD-relative
	// ./logs. A packaged app must not write into its install directory - nor
	// into whatever directory a launcher happened to start it from - so while
	// the value is still that default, keep logs beside the database.
	//
	// The comparison is on the cleaned path because the shipped
	// config/aurora.toml spells it "./logs/", which is the same directory as
	// "./logs" but a different string: an equality test on the raw value let a
	// packaged app create ./logs relative to the launch directory (measured,
	// while verifying the logger fix). Any other value wins, so an explicit
	// log.path in the config still decides.
	if path := viper.GetString("log.path"); path == "" || filepath.Clean(path) == "logs" {
		viper.Set("log.path", filepath.Join(filepath.Dir(dbPath), "logs"))
	}
	logger.Init()

	srv, err := api.NewServer()
	if err != nil {
		return nil, err
	}

	rt := &Runtime{server: srv, webRoot: webRoot, webFS: webFS, dbPath: dbPath}
	rt.stopScheduler = srv.StartOracleScheduler(ctx, config.OracleSchedulerCheckInterval())
	rt.stopOutbox = srv.StartAuditOutboxDrainer(ctx)
	return rt, nil
}

// prepareDataDir resolves the directory holding aurora.db and creates it. A
// packaged desktop app must not depend on its working directory, so the
// default is the platform user-data directory rather than ./data.
func prepareDataDir(dir string, defaultDir func() (string, error)) (string, error) {
	if dir == "" {
		resolve := defaultDir
		if resolve == nil {
			resolve = userDataDir
		}
		base, err := resolve()
		if err != nil {
			return "", err
		}
		dir = base
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf(T("desktop.error.data_dir"), dir, err)
	}
	return filepath.Join(dir, "aurora.db"), nil
}

// userDataDir is overridable in tests.
var userDataDir = platformUserDataDir

// platformUserDataDir follows the OS convention (XDG on Linux,
// ~/Library/Application Support on macOS, %AppData% on Windows).
func platformUserDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf(T("desktop.error.user_config_dir"), err)
	}
	return filepath.Join(base, "aurora"), nil
}

// ResolveWebRoot finds the shipped web/ directory. explicit is authoritative:
// when it is given it must be a web root, and nothing else is tried, because
// `--web-root DIR` is how a developer says "these are the pages I am editing".
// Falling through to another copy made a typo look like a working flag: the
// window rendered some other directory's UI, so edits never appeared and
// nothing said why.
//
// Without an explicit value the candidates are tried in order and the first
// one holding index.html is used. Resolving next to the executable first means
// a staged bundle works regardless of the directory it is launched from.
func ResolveWebRoot(explicit, configured string) (string, error) {
	var candidates []string
	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		for _, c := range candidates {
			if c == abs {
				return
			}
		}
		candidates = append(candidates, abs)
	}

	add(explicit)
	if explicit != "" {
		if len(candidates) == 0 {
			return "", fmt.Errorf(T("desktop.error.web_root_unresolvable"), explicit)
		}
		if dir := candidates[0]; isWebRoot(dir) {
			return dir, nil
		}
		return "", fmt.Errorf(T("desktop.error.web_root_no_index"), explicit)
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(exe), "web"))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "web"))
	}
	add(configured)

	for _, dir := range candidates {
		if isWebRoot(dir) {
			return dir, nil
		}
	}
	return "", errors.New(T("desktop.error.web_root_not_found"))
}

func isWebRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && info.Mode().IsRegular()
}

// URL is the page the main window loads.
func URL() string { return Scheme + "://localhost/" }

// BackupStamp is the directory-name suffix for a menu-triggered backup.
func BackupStamp(now time.Time) string {
	return now.Format("20060102-150405")
}
