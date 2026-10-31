package blockchain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/pplmx/aurora/internal/infra/store"
	"github.com/pplmx/aurora/internal/logger"
)

var (
	instance *Chain
	once     sync.Once
	initErr  error
)

// ErrLedgerCorruption marks a persisted ledger that cannot be decoded or no
// longer forms the verified chain loaded at startup. A transient SQLite read
// failure may still fall back to the in-memory chain, but corruption must stop
// an append: writing on top of it would make a bad ledger look usable again.
var ErrLedgerCorruption = errors.New("blockchain: persisted ledger is corrupt")

// InitWithError initializes the process-wide chain and reports any database
// failure, unreadable row, or integrity violation. Returning a chain after an
// open failure would label writes as durable while they existed only in memory.
func InitWithError() (*Chain, error) {
	once.Do(func() {
		chain := &Chain{Blocks: []*Block{Genesis()}}

		db, err := store.OpenShared()
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

// ResetForTest forgets the chain and the shared handle so a test can point
// db.path at a fresh database and start over.
func ResetForTest() {
	store.ResetForTest()
	instance = nil
	once = sync.Once{}
	initErr = nil
}

// Close closes the shared handle, if one was opened.
func Close() error { return store.Close() }
