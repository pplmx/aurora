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
	EventBus     *infraevents.CompositeEventBus
	TokenService token.Service
}

func Wire(dataDir string) (*App, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	eventStore, err := infraevents.NewSQLiteEventStore(filepath.Join(dataDir, "events.db"))
	if err != nil {
		return nil, err
	}

	replay, err := infraevents.NewSQLiteReplayProtection(filepath.Join(dataDir, "nonces.db"))
	if err != nil {
		return nil, err
	}

	bus := infraevents.NewCompositeEventBus()
	bus.SyncBus.SubscribeAll(infraevents.NewAuditHandler(eventStore).Handle)
	bus.SyncBus.SubscribeAll(infraevents.NewStatsHandler().Handle)

	chain := blockchain.NewBlockChain()

	tokenRepo, err := sqlite.NewTokenRepository(filepath.Join(dataDir, "tokens.db"))
	if err != nil {
		return nil, err
	}

	eventReader := sqlite.NewTokenEventReader(eventStore)

	tokenService := token.NewService(tokenRepo, bus, eventReader, replay, chain)

	return &App{
		EventBus:     bus,
		TokenService: tokenService,
	}, nil
}
