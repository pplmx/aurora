package app

import (
	"os"
	"path/filepath"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type App struct {
	EventBus     infraevents.EventBus
	TokenService token.Service

	// closers releases every SQLite handle Wire opened (event store, replay
	// protection, token repo). Tests call Close() before t.TempDir() cleanup so
	// Windows can delete the temp data dir; production gets graceful release.
	closers []func() error
}

// addCloser appends a handle to be released by Close().
func (a *App) addCloser(c interface{ Close() error }) {
	a.closers = append(a.closers, c.Close)
}

// Close releases every SQLite handle the App opened. Safe to call once at
// shutdown; best-effort (returns the first error encountered, if any).
func (a *App) Close() error {
	var firstErr error
	for _, c := range a.closers {
		if err := c(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	a.closers = nil
	return firstErr
}

func Wire(dataDir string) (*App, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	app := &App{}

	eventStore, err := infraevents.NewSQLiteEventStore(filepath.Join(dataDir, "events.db"))
	if err != nil {
		return nil, err
	}
	app.addCloser(eventStore)

	replay, err := infraevents.NewSQLiteReplayProtection(filepath.Join(dataDir, "nonces.db"))
	if err != nil {
		_ = app.Close()
		return nil, err
	}
	app.addCloser(replay)

	// The token service only drives the synchronous bus (audit + stats
	// handlers). The previous CompositeEventBus also spawned an idle
	// AsyncEventBus goroutine and an empty PluginBus, neither subscribed nor
	// closeable from App — one leaked goroutine per Wire() call.
	bus := infraevents.NewSyncEventBus()
	bus.SubscribeAll(infraevents.NewAuditHandler(eventStore).Handle)
	bus.SubscribeAll(infraevents.NewStatsHandler().Handle)

	chain := blockchain.NewBlockChain()

	tokenRepo, err := sqlite.NewTokenRepository(filepath.Join(dataDir, "tokens.db"))
	if err != nil {
		_ = app.Close()
		return nil, err
	}
	app.addCloser(tokenRepo)

	txManager := sqlite.NewTxManager(tokenRepo.GetDB())

	eventReader := sqlite.NewTokenEventReader(eventStore)

	tokenService := token.NewService(tokenRepo, txManager, bus, eventReader, replay, chain)

	app.EventBus = bus
	app.TokenService = tokenService
	return app, nil
}
