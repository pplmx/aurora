// Package deskapp hosts Aurora in a MyGo desktop window.
//
// The window does not talk to a listening HTTP server: the app's own router is
// served over the "aurora" URL scheme, so the page and the REST API share one
// origin. That keeps the API key off the network entirely — the same
// injectAPIKey gateway embeds it into the served HTML, and every fetch stays
// inside the process.
package deskapp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/app/container"
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
	dbPath        string
	stopScheduler func()
	stopOutbox    func()

	// app is the window's state and view. The desktop shell renders it with
	// MyGo's own widgets; there is no web page involved.
	app *nativeui.App
}

// App returns the window's state. The shell passes it to mygo.WindowOptions
// Content; a nil App means Start has not completed.
func (r *Runtime) App() *nativeui.App { return r.app }

// Router returns the gateway's API surface. The desktop window does not serve
// it, but a test (and any future in-process client) can drive the same
// handlers the REST binary exposes.
func (r *Runtime) Router() http.Handler { return r.server.Router() }

// DBPath reports the SQLite file the app is using.
func (r *Runtime) DBPath() string { return r.dbPath }

// Services exposes the container the runtime assembled, so the shell can share
// its repositories instead of opening a second handle for a one-shot action.
func (r *Runtime) Services() container.Services { return r.server.Services() }

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
	if err := keepApplicationFilesPrivate(dbPath, viper.GetString("log.path")); err != nil {
		_ = srv.Close()
		logger.Close()
		return nil, fmt.Errorf(T("desktop.error.data_dir"), filepath.Dir(dbPath), err)
	}

	svc := srv.Services()
	rt := &Runtime{
		server: srv,
		dbPath: dbPath,
	}
	// Start the background work before the window state exists so a page that
	// reads scheduler health (the Oracle page's Feed Health card) sees the same
	// live counters the REST endpoint publishes, rather than a nil scheduler
	// that would make the card permanently empty.
	rt.stopScheduler = srv.StartOracleScheduler(ctx, config.OracleSchedulerCheckInterval())
	rt.stopOutbox = srv.StartAuditOutboxDrainer(ctx)
	rt.app = nativeui.New(newNativeServices(
		svc.LotteryRepository(), svc.VotingRepository(), svc.NFTRepository(),
		svc.OracleRepository(), svc.Token(), svc.TokenTransactions(), svc.NFTTransactions(), svc.Ledger(),
		svc.Fetcher(), srv.OracleHealthStats,
	))
	return rt, nil
}

// prepareDataDir resolves the directory holding aurora.db and creates it. A
// packaged desktop app must not depend on its working directory, so the
// default is the platform user-data directory rather than ./data. The result
// is absolute even when the caller spelled a relative --data-dir: the shell
// compares it with paths forwarded by later launches, which MyGo resolves
// against the launcher's directory.
func prepareDataDir(dir string, defaultDir func() (string, error)) (string, error) {
	privateByDefault := dir == ""
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
	// The database stores operator-supplied Oracle headers, which can include
	// bearer credentials, and the adjacent log can contain request details.
	// Keep a newly created application tree private to this OS user. An
	// explicit --data-dir may intentionally point at a managed or shared
	// directory, so only tighten the automatically selected app directory;
	// changing permissions on an operator's existing path would be surprising
	// and could break other consumers. MkdirAll's mode does not tighten a
	// directory left by an older release, so the default needs an explicit
	// Chmod before SQLite or the logger create files inside it.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf(T("desktop.error.data_dir"), dir, err)
	}
	if privateByDefault && runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", fmt.Errorf(T("desktop.error.data_dir"), dir, err)
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf(T("desktop.error.data_dir"), dir, err)
	}
	return filepath.Join(abs, "aurora.db"), nil
}

// keepApplicationFilesPrivate applies owner-only modes to the state SQLite
// and the logger create before they are exposed to the shell. The directory's
// 0700 mode is the primary boundary, but these files can contain Oracle
// bearer headers and request details; keeping the files themselves at 0600
// also protects them if that directory is later copied or its mode is changed.
// SQLite manages the WAL and shared-memory sidecars alongside the database.
func keepApplicationFilesPrivate(dbPath, logPath string) error {
	if runtime.GOOS == "windows" {
		// AppData inherits a per-user ACL; Unix permission bits do not apply.
		return nil
	}

	paths := []string{dbPath, dbPath + "-wal", dbPath + "-shm"}
	if logPath != "" {
		paths = append(paths, filepath.Join(logPath, "aurora.log"))
	}
	for _, path := range paths {
		if err := os.Chmod(path, 0o600); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// SQLite need not have created a sidecar yet, and an explicit
				// console log path has no file to protect.
				continue
			}
			return fmt.Errorf("restrict %s to owner: %w", path, err)
		}
	}
	return nil
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

// URL is the page the main window loads.
func URL() string { return Scheme + "://localhost/" }

// BackupStamp is the directory-name suffix for a menu-triggered backup.
func BackupStamp(now time.Time) string {
	return now.Format("20060102-150405")
}
