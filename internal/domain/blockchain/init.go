package blockchain

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	sqlite3 "github.com/mattn/go-sqlite3"
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
	// If the table itself cannot be created, the chain runs in-memory only: the
	// persist/syncFromDB hooks are left nil so appends don't churn failing DB
	// round-trips after a boot that already logged the failure (the package's
	// documented "operate in non-persistent mode" fallback, TASK-244 review LOW).
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
		logger.Error().Err(err).Msg("Failed to create blocks table; blockchain will operate in non-persistent mode")
		return
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
// Two failure modes are reconciled against the ledger:
//
//  1. Peer is ahead: rows at height >= len(chain.Blocks) were committed by a
//     concurrent process and this process has not loaded them yet. They are
//     appended in order (only at-or-past the tail, so the common single-writer
//     case is a no-op).
//  2. Phantom tail (the dangerous one): a prior persist returned a NON-conflict
//     error (SQLITE_BUSY after the busy timeout, disk full...). Per the
//     best-effort contract that block stays in memory, so the in-memory chain
//     is ONE taller than the DB at the seam. If a peer then commits ITS block
//     at that same seam height, len == DB height+1 while the DB block at
//     height len-1 is a DIFFERENT block. The windowed SELECT above would miss
//     that divergence (the row is below the window) and the next append would
//     mine on top of a phantom PrevHash — a broken chain that only surfaces as
//     VerifyIntegrity corruption on the next boot. So before appending, the
//     seam is checked: the DB block at height len-1 must be exactly the
//     in-memory tail, else the chain is rebuilt from the authoritative ledger
//     and the append retries at the true next height.
func appendBlocksFromDB(chain *BlockChain, db *sql.DB) error {
	tail := int64(len(chain.Blocks))
	if tail > 0 {
		var seamHeight int64
		var seamHash string
		// The DB block at or directly below the seam.
		err := db.QueryRow("SELECT height, hash FROM blocks WHERE height <= ? ORDER BY height DESC LIMIT 1", tail-1).
			Scan(&seamHeight, &seamHash)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// DB has nothing below the tail (fresh ledger / reset): in-memory
			// boot view stands; nothing to fold in.
			return nil
		case err != nil:
			return err
		}
		if seamHeight == tail-1 && seamHash != string(chain.Blocks[tail-1].Hash) {
			// Divergent seam: a peer owns height tail-1 with a different block
			// than the phantom one this process kept after a persist failure.
			// Rebuild from the authoritative ledger (drop the phantom tail),
			// then let appendBlock retry at the true next height.
			return rebuildChainFromDB(chain, db)
		}
		if seamHeight < tail-1 {
			// DB is genuinely shorter than memory below the seam (persist
			// failure left the tail in memory only). Safe to widen only if the
			// DB has NOT climbed past the seam via a different peer chain; any
			// peer row at or above the seam means the phantom tail no longer
			// links to the ledger → rebuild.
			var peerAbove int
			if err := db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height >= ?", tail).Scan(&peerAbove); err != nil {
				return err
			}
			if peerAbove > 0 {
				return rebuildChainFromDB(chain, db)
			}
			// DB strictly shorter, no peers above: keep the in-memory tail
			// (non-persistent fallback) and proceed below (no-op SELECT).
		}
	}

	rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks WHERE height >= ? ORDER BY height", tail)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	// Appended blocks are buffered and committed to the chain at the end, so a
	// mid-iteration scan/rows.Err() failure leaves the in-memory chain
	// untouched (atomic sync) instead of half-folded.
	var fresh []*Block
	for rows.Next() {
		block, err := scanBlockRow(rows)
		if err != nil {
			return err
		}
		if block.Height == 0 {
			continue
		}
		fresh = append(fresh, block)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	chain.Blocks = append(chain.Blocks, fresh...)
	return nil
}

// rebuildChainFromDB resets the in-memory chain to the authoritative persisted
// ledger (genesis + every non-genesis block row, in height order). Any phantom
// tail blocks a persist failure left in memory are dropped: they were never
// committed, so the shared ledger does not know them. Used by appendBlocksFromDB
// when the in-memory tail diverges from the DB seam (TASK-244 review, ISS-242).
func rebuildChainFromDB(chain *BlockChain, db *sql.DB) error {
	// Keep genesis as the anchor (a bound chain always has one); a nil/empty
	// chain (constructed bare, e.g. in tests) is re-seeded rather than panicked
	// on.
	if len(chain.Blocks) > 0 {
		chain.Blocks = chain.Blocks[:1]
	} else {
		chain.Blocks = []*Block{Genesis()}
	}
	_, err := loadBlocksFromDB(chain, db)
	return err
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

// sqliteIsUniqueViolation reports whether err is a SQLite constraint violation
// of the blocks table's height PRIMARY KEY — i.e. the INSERT this process
// attempted collided with a row a concurrent process committed at the same
// height. It matches on the extended error code rather than the message text:
// mattn/go-sqlite3 returns the error by value (so errors.As into its exported
// value type), and an INTEGER PRIMARY KEY duplicate surfaces as
// ErrConstraintPrimaryKey (extended 1555) even though SQLite prints "UNIQUE
// constraint failed: blocks.height". Matching text would both break on driver
// message-format changes and misclassify a future unrelated UNIQUE constraint
// as a height collision (TASK-244 review, MEDIUM).
func sqliteIsUniqueViolation(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.ExtendedCode {
		case sqlite3.ErrConstraintPrimaryKey, sqlite3.ErrConstraintUnique:
			return true
		}
	}
	return false
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
