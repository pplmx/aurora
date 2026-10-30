package blockchain

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/viper"
)

var (
	instance   *Chain
	dbInstance *sql.DB
	dbInitOnce sync.Once
	dbInitErr  error
	once       sync.Once
	initErr    error
)

// ErrLedgerCorruption marks a persisted ledger that cannot be decoded or no
// longer forms the verified chain loaded at startup. A transient SQLite read
// failure may still fall back to the in-memory chain, but corruption must stop
// an append: writing on top of it would make a bad ledger look usable again.
var ErrLedgerCorruption = errors.New("blockchain: persisted ledger is corrupt")

const defaultDBPath = "./data/aurora.db"

// resolvedDBPath returns the effective SQLite database location: the
// configured db.path (aurora.toml / env / viper, set by config.Load and the
// CLI's initConfig) when non-empty, else the default. Every store, the API
// server, and both migrate paths resolve through this single function, so a
// configured db.path can no longer silently split-brain with the hardcoded
// default (TASK-102, ISS-094).
func resolvedDBPath() string {
	if p := viper.GetString("db.path"); p != "" {
		return p
	}
	return defaultDBPath
}

// DBPath returns the configured database path without touching the filesystem.
// Opening the database (or constructing a repository) owns directory creation
// and must surface its error; returning an empty path here would make SQLite
// silently open a temporary database and report writes that vanish at exit.
func DBPath() string { return resolvedDBPath() }

// InitDB returns the process-wide singleton *sql.DB, opening it on
// the first call. Concurrent callers all observe the same pointer
// (the sync.Once guarantees sql.Open runs exactly once).
//
// The previous implementation read dbInstance, called sql.Open, and
// assigned dbInstance without any synchronization — two concurrent
// callers could both see nil, both call sql.Open, and leak the first
// connection (it would never get closed).
func InitDB() (*sql.DB, error) {
	dbInitOnce.Do(func() {
		dbInstance, dbInitErr = OpenDB(resolvedDBPath())
	})
	return dbInstance, dbInitErr
}

// InitWithError initializes the process-wide chain and reports any database
// failure, unreadable row, or integrity violation. Returning a chain after an
// open failure would label writes as durable while they existed only in memory.
func InitWithError() (*Chain, error) {
	once.Do(func() {
		chain := &Chain{Blocks: []*Block{Genesis()}}

		db, err := InitDB()
		if err != nil {
			initErr = fmt.Errorf("open blockchain database: %w", err)
			return
		}
		initErr = bindChainToDB(chain, db)

		if initErr == nil {
			instance = chain
		}
	})

	if initErr != nil {
		return nil, initErr
	}
	return instance, nil
}

// Init preserves the historical process-wide constructor for callers that
// cannot return an error. New production write paths use InitWithError so
// ledger corruption is surfaced instead of degrading to a partial chain.
func Init() *Chain {
	chain, err := InitWithError()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load a trustworthy blockchain")
	}
	return chain
}

func Get() *Chain {
	chain, err := InitWithError()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load a trustworthy blockchain")
	}
	return chain
}

func ResetForTest() {
	if dbInstance != nil {
		_ = dbInstance.Close()
		dbInstance = nil
	}
	dbInitOnce = sync.Once{}
	dbInitErr = nil
	instance = nil
	once = sync.Once{}
	initErr = nil
	_ = os.RemoveAll("./data")
}

func Close() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}
