package blockchain

import (
	"database/sql"
	"encoding/json"
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

			rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks ORDER BY height")
			if err != nil {
				logger.Error().Err(err).Msg("Failed to query blocks from database")
			} else {
				for rows.Next() {
					var block Block
					var hash, prevHash, data string
					if err := rows.Scan(&block.Height, &hash, &prevHash, &data, &block.Nonce, &block.Timestamp); err != nil {
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
				// A mid-iteration driver error would otherwise be silently
				// dropped with the rows closed; surface it so a failed reload
				// (which desyncs the in-memory chain from the persisted table
				// and can let the next append overwrite a stored block via
				// INSERT OR REPLACE) is never quiet.
				if err := rows.Err(); err != nil {
					logger.Error().Err(err).Msg("Failed while iterating persisted blocks")
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

// AddLotteryRecord appends a lottery or oracle audit record to the chain and
// returns its height. Unlike AddBlock, it stamps the true height into the
// record's JSON `block_height` field immediately before the block is mined,
// so the immutable on-chain payload is self-describing.
//
// The app-layer callers (lottery create, oracle fetch, the scheduler, the
// TUI) serialized their record while its height was still 0, called
// AddLotteryRecord, and only then set BlockHeight from the return value — so
// the permanent chain copy claimed "block_height":0 and a consumer replaying
// the chain could never correlate a block to its record (ISS-097). Stamping
// here, under the same lock that assigns the height, fixes every caller at
// the one point they all funnel through.
func (c *BlockChain) AddLotteryRecord(data string) (int64, error) {
	return c.appendBlock(data, func(height int64) string {
		stamped, err := stampRecordHeight(data, height)
		if err != nil {
			return "" // not JSON (or unreadable) — store unchanged
		}
		return stamped
	})
}

// stampRecordHeight rewrites the block_height field of a JSON audit record to
// the given height. Non-JSON payloads (e.g. tests passing bare strings) error
// and are stored unchanged. Re-marshalling via a map drops the original key
// order but preserves every value; the chain payload is opaque text so
// ordering is irrelevant.
func stampRecordHeight(data string, height int64) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return "", err
	}
	m["block_height"] = height
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Close() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}
