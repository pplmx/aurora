package blockchain

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
			bindChainToDB(chain, db)
		}

		instance = chain
	})

	return instance
}

// bindChainToDB wires a chain to a blocks table in the given DB: creates the
// table if needed, reloads persisted blocks, seeds genesis when the table is
// empty, and installs the conflict-detecting persist + DB-authoritative height
// resync hooks. It is shared by InitBlockChain (via the process singleton) and
// the multi-process collision tests, which build two independent chains over
// one DB file to exercise TASK-244 / ISS-242.
func bindChainToDB(chain *BlockChain, db *sql.DB) {
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

	// persisted tracks whether ANY block row exists in the blocks table
	// (including the height-0 genesis row). It drives the seed-below:
	// once genesis is stored, a later boot that reloads only genesis in
	// memory (len==1) must not try to INSERT OR REPLACE... a plain
	// INSERT of height 0 again — that would hit the PRIMARY KEY every
	// start and log a spurious "Failed to insert genesis block" on an
	// otherwise healthy restart (TASK-238, ISS-236).
	persisted, err := loadBlocksFromDB(chain, db)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to query blocks from database")
	}

	seedGenesisIfEmpty(db, chain, persisted)

	// Wire persistence: every block appended by AddBlock is written to the
	// blocks table so heights remain monotonic across restarts. The previous
	// code created+loaded the table but never saved new blocks, so the chain
	// silently reset to genesis on every restart.
	//
	// Cross-process safety (TASK-244, ISS-242): height is the PRIMARY KEY, but
	// INSERT OR REPLACE would let a second process sharing the DB file silently
	// overwrite a committed block (both boot genesis, both compute height 1,
	// the loser's row is replaced and the winner's payload is lost from the
	// ledger). A plain INSERT is used instead so a duplicate height surfaces as
	// a UNIQUE conflict, which is mapped to ErrHeightConflict for appendBlock's
	// drop-and-retry loop (the block is never overwritten). This also treats an
	// identical re-persist as a conflict — safe because appendBlock drops the
	// in-memory candidate and re-syncs it from the DB, converging to the same
	// chain rather than erroring twice on the same height.
	chain.persist = func(b *Block) error {
		_, err := db.Exec(`
			INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, b.Height, string(b.Hash), string(b.PrevHash), string(b.Data), b.Nonce, b.Timestamp, b.Timestamp)
		if err != nil {
			if sqliteIsUniqueViolation(err) {
				return ErrHeightConflict
			}
			return err
		}
		return nil
	}

	// Wire the DB-authoritative height source: before every append the
	// in-memory chain is reloaded from the shared blocks table so a concurrent
	// process's committed blocks are folded in and the next reserved height is
	// the true next free one (TASK-244, ISS-242). Lazy windowing: only rows at
	// or past the current in-memory tail are loaded, so the overwhelmingly
	// common single-writer case is a no-op.
	chain.syncFromDB = func() error {
		return appendBlocksFromDB(chain, db)
	}
}

func GetBlockChain() *BlockChain {
	if instance == nil {
		return InitBlockChain()
	}
	return instance
}

// loadBlocksFromDB reads every row of the blocks table into the chain's
// in-memory Blocks slice. It returns (true, nil) when at least one row existed
// (the genesis row counts), mirroring the pre-TASK-238 semantics that drove
// genesis seeding; a nil error means the table was queried successfully
// (possibly empty). The height-0 genesis row is skipped because genesis is
// always seeded in memory; skipping it avoids doubling it on reload.
func loadBlocksFromDB(chain *BlockChain, db *sql.DB) (bool, error) {
	persisted := false
	rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks ORDER BY height")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		persisted = true
		block, err := scanBlockRow(rows)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to scan block row, skipping")
			continue
		}
		if block.Height == 0 {
			continue
		}
		chain.Blocks = append(chain.Blocks, block)
	}
	return persisted, rows.Err()
}

// appendBlocksFromDB folds blocks a concurrent process committed to the shared
// DB into the in-memory chain, so the chain's length is the DB-authoritative
// next height. It is wired as BlockChain.syncFromDB (TASK-244, ISS-242).
//
// To keep the common single-writer path cheap it only SELECTs rows at
// height >= len(chain.Blocks) — i.e. blocks this process has not loaded yet.
// Under the write lock held by appendBlock this is safe: a successful persist
// always occurred while this process already had the block in memory, so rows
// this process committed are never taller than the in-memory tail.
func appendBlocksFromDB(chain *BlockChain, db *sql.DB) error {
	tail := int64(len(chain.Blocks))
	rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks WHERE height >= ? ORDER BY height", tail)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	// Unlike boot-time reload (which tolerates a corrupt row by skipping it,
	// yielding a shorter-but-valid chain), the in-place resync must NOT skip a
	// bad row and keep appending: subsequent rows link to the skipped height,
	// so a continuation would wire an invalid PrevHash, and the next append
	// would mine on top of that broken tail. Abort the whole sync instead;
	// appendBlock falls back to the stale in-memory height and the persist
	// conflict check protects the DB.
	for rows.Next() {
		block, err := scanBlockRow(rows)
		if err != nil {
			return err
		}
		if block.Height == 0 {
			continue
		}
		chain.Blocks = append(chain.Blocks, block)
	}
	return rows.Err()
}

// scanBlockRow reads one blocks-table row (skip the created_at column) into a
// *Block. Shared by boot-time reload and append-time DB resync.
func scanBlockRow(rows *sql.Rows) (*Block, error) {
	var block Block
	var hash, prevHash, data string
	if err := rows.Scan(&block.Height, &hash, &prevHash, &data, &block.Nonce, &block.Timestamp); err != nil {
		return nil, err
	}
	block.Hash = []byte(hash)
	block.PrevHash = []byte(prevHash)
	block.Data = []byte(data)
	return &block, nil
}

// sqliteIsUniqueViolation reports whether err is SQLite's UNIQUE constraint
// violation (code 2067, "UNIQUE constraint failed: blocks.height"). SQLite
// surfaces these as errors whose message contains that exact text via
// mattn/go-sqlite3.
func sqliteIsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// seedGenesisIfEmpty writes the in-memory genesis block (Blocks[0]) to the
// blocks table ONLY when no block row exists yet. It is keyed on `persisted`
// (did the reload above read any row, including the height-0 genesis row?)
// rather than chain length: after a reload the h0 row was skipped into memory,
// so the chain is "genesis-only" again — the old `len(Blocks) <= 1` guard
// stayed true and re-inserted height 0 on every boot, tripping the PRIMARY KEY
// and logging a false "Failed to insert genesis block" (TASK-238, ISS-236).
func seedGenesisIfEmpty(db *sql.DB, chain *BlockChain, persisted bool) {
	if persisted {
		return
	}
	stmt, err := db.Prepare("INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to prepare block insert statement")
		return
	}
	defer func() { _ = stmt.Close() }()
	block := chain.Blocks[0]
	if _, err := stmt.Exec(block.Height, string(block.Hash), string(block.PrevHash), string(block.Data), block.Nonce, block.Timestamp, block.Timestamp); err != nil {
		logger.Error().Err(err).Msg("Failed to insert genesis block")
	}
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
