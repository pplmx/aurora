package blockchain

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/viper"
)

var (
	instance   *BlockChain
	dbInstance *sql.DB
	dbInitOnce sync.Once
	dbInitErr  error
	once       sync.Once
)

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

func DBPath() string {
	path := resolvedDBPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	return path
}

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
		path := resolvedDBPath()
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			dbInitErr = err
			return
		}

		// _txlock=immediate + _busy_timeout: this singleton DB backs voting
		// writes through a TxManager over a real pool, so a deferred
		// read-then-write transaction racing a concurrent writer hits the
		// un-waitable SQLITE_BUSY_SNAPSHOT — serialize writers at BEGIN and
		// wait up to 5s for the lock instead (v1.70, ISS-076; same values as
		// internal/infra/sqlite/dsn.go).
		db, err := sql.Open("sqlite3", path+"?_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000")
		if err != nil {
			dbInitErr = err
			return
		}

		if err := db.Ping(); err != nil {
			_ = db.Close()
			dbInitErr = err
			return
		}

		dbInstance = db
	})
	return dbInstance, dbInitErr
}

func InitBlockChain() *BlockChain {
	once.Do(func() {
		chain := &BlockChain{Blocks: []*Block{Genesis()}}

		if db, err := InitDB(); err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize database; blockchain will operate in non-persistent mode")
		} else {
			if _, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS blocks (
					height INTEGER PRIMARY KEY,
					hash TEXT NOT NULL,
					previous_hash TEXT NOT NULL,
					data TEXT NOT NULL,
					nonce INTEGER NOT NULL,
					timestamp INTEGER NOT NULL,
					created_at INTEGER NOT NULL
				)
			`); err != nil {
				logger.Error().Err(err).Msg("Failed to create blocks table")
			}

			rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce FROM blocks ORDER BY height")
			if err != nil {
				logger.Error().Err(err).Msg("Failed to query blocks from database")
			} else {
				for rows.Next() {
					var block Block
					var hash, prevHash, data string
					if err := rows.Scan(&block.Height, &hash, &prevHash, &data, &block.Nonce); err != nil {
						logger.Warn().Err(err).Msg("Failed to scan block row, skipping")
						continue
					}
					// Genesis is always seeded in memory above; the persisted
					// genesis row (height 0) is the same block, so skipping it
					// avoids doubling it on reload.
					if block.Height == 0 {
						continue
					}
					block.Hash = []byte(hash)
					block.PrevHash = []byte(prevHash)
					block.Data = []byte(data)
					chain.Blocks = append(chain.Blocks, &block)
				}
				if err := rows.Close(); err != nil {
					logger.Warn().Err(err).Msg("Failed to close rows")
				}
			}

			if len(chain.Blocks) <= 1 {
				stmt, err := db.Prepare("INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
				if err != nil {
					logger.Error().Err(err).Msg("Failed to prepare block insert statement")
				} else {
					block := chain.Blocks[0]
					if _, err := stmt.Exec(block.Height, string(block.Hash), string(block.PrevHash), string(block.Data), block.Nonce, block.Timestamp, block.Timestamp); err != nil {
						logger.Error().Err(err).Msg("Failed to insert genesis block")
					}
				}
			}

			// Wire persistence: every block appended by AddBlock is written to
			// the blocks table so heights remain monotonic across restarts.
			// The previous code created+loaded the table but never saved new
			// blocks, so the chain silently reset to genesis on every restart.
			chain.persist = func(b *Block) error {
				_, err := db.Exec(`
					INSERT OR REPLACE INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, b.Height, string(b.Hash), string(b.PrevHash), string(b.Data), b.Nonce, b.Timestamp, b.Timestamp)
				return err
			}
		}

		instance = chain
	})

	return instance
}

func GetBlockChain() *BlockChain {
	if instance == nil {
		return InitBlockChain()
	}
	return instance
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
	_ = os.RemoveAll("./data")
}

func (c *BlockChain) AddLotteryRecord(data string) (int64, error) {
	return c.AddBlock(data)
}

func Close() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}
