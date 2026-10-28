// Package container is the process-level composition root for Aurora's
// application services. It owns the SQLite handles, repositories, ledger
// singleton and event bus; presentation layers receive this container instead
// of constructing storage themselves.
package container

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/backup"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/http"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// Container holds the assembled application services. Its fields are
// read-only to callers: Close owns their lifetime.
type Container struct {
	DB     *sql.DB
	DBPath string

	LotteryRepo *sqlite.LotteryRepository
	VotingRepo  *sqlite.VotingRepository
	NFTRepo     *sqlite.NFTRepository
	TokenRepo   *sqlite.TokenRepository
	OracleRepo  *sqlite.OracleRepository

	TokenService  token.Service
	TxManager     *sqlite.TxManager
	VotingTx      voting.TransactionManager
	NFTTx         nft.TransactionManager
	Chain         *blockchain.Chain
	OracleFetcher oracleapp.FetcherInterface

	EventStore *infraevents.SQLiteEventStore

	closers   []func() error
	closeOnce sync.Once
	closeErr  error
}

func (c *Container) addCloser(v interface{ Close() error }) {
	c.closers = append(c.closers, v.Close)
}

// Close releases every handle Open created. It matches the previous
// api.Server.Close order and is safe to call concurrently: the first caller
// owns the close sequence, while later callers wait for it and receive the
// same result.
func (c *Container) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		for _, closeFn := range c.closers {
			if err := closeFn(); err != nil && c.closeErr == nil {
				c.closeErr = err
			}
		}
		c.closers = nil
	})
	return c.closeErr
}

// Open wires the storage-facing application layer. It is all-or-nothing: a
// failure closes every handle opened so far, so callers never inherit a
// half-constructed container or a leaked SQLite handle.
func Open() (*Container, error) {
	dbPath := blockchain.DBPath()
	db, err := blockchain.OpenDB(dbPath)
	if err != nil {
		return nil, err
	}
	c := &Container{DB: db, DBPath: dbPath}
	c.addCloser(db)
	complete := false
	defer func() {
		if !complete {
			_ = c.Close()
		}
	}()

	lotteryRepo, err := sqlite.NewLotteryRepository(dbPath)
	if err != nil {
		return nil, err
	}
	c.LotteryRepo = lotteryRepo
	c.addCloser(lotteryRepo)

	votingRepo, err := sqlite.NewVotingRepository(db)
	if err != nil {
		return nil, err
	}
	c.VotingRepo = votingRepo
	c.VotingTx = sqlite.NewTxManager(db)

	nftRepo, err := sqlite.NewNFTRepository(dbPath)
	if err != nil {
		return nil, err
	}
	c.NFTRepo = nftRepo
	c.addCloser(nftRepo)
	c.NFTTx = sqlite.NewTxManager(nftRepo.GetDB())

	tokenRepo, err := sqlite.NewTokenRepository(dbPath)
	if err != nil {
		return nil, err
	}
	c.TokenRepo = tokenRepo
	c.addCloser(tokenRepo)

	eventStore, err := infraevents.NewSQLiteEventStore(dbPath)
	if err != nil {
		return nil, err
	}
	c.EventStore = eventStore
	c.addCloser(eventStore)

	eventReader := sqlite.NewTokenEventReader(eventStore)
	eventBus := infraevents.NewSyncEventBus()
	// Token audit events must be persisted on every production path. The
	// outbox-backed handler parks transient delivery failures for the drainer
	// started by cmd/api or the desktop runtime.
	eventBus.SubscribeAll(infraevents.NewAuditHandlerWithOutbox(eventStore, eventStore).Handle)

	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	if err != nil {
		return nil, err
	}
	c.addCloser(replay)

	chain, err := blockchain.NewWithDB(db)
	if err != nil {
		return nil, fmt.Errorf("initialize blockchain: %w", err)
	}
	txManager := sqlite.NewTxManager(tokenRepo.GetDB())
	c.Chain = chain
	c.TxManager = txManager
	c.TokenService = token.NewService(tokenRepo, txManager, eventBus, eventReader, replay, chain)

	oracleRepo, err := sqlite.NewOracleRepository(dbPath)
	if err != nil {
		return nil, err
	}
	c.OracleRepo = oracleRepo
	c.addCloser(oracleRepo)
	c.OracleFetcher = http.NewFetcher()

	complete = true
	return c, nil
}

// StartOutboxDrainer retries parked token-audit deliveries until ctx is
// cancelled. The returned stop function waits for an in-flight pass before
// returning, so Close cannot race a drain against a closed database. It is a
// container lifecycle concern, like the SQLite handles it uses, so the HTTP
// layer does not import the events implementation.
func (c *Container) StartOutboxDrainer(ctx context.Context) (stop func()) {
	if c == nil || c.EventStore == nil {
		return func() {}
	}
	drainer := infraevents.NewOutboxDrainer(c.EventStore, nil)
	return drainer.Start(ctx)
}

// Backup creates the WAL-complete, checksummed snapshot the CLI writes. The
// desktop calls it through the narrow Services port instead of importing the
// backup implementation.
func (c *Container) Backup(ctx context.Context, dest string) (string, error) {
	if c == nil || c.DBPath == "" {
		return "", errors.New("backup database path is empty")
	}
	res, err := backup.NewBackupService(map[string]string{"aurora": c.DBPath}).Create(ctx, dest)
	if err != nil {
		return "", err
	}
	return res.File, nil
}
