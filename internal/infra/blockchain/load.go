package blockchain

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// bindChainToDB wires a chain to a blocks table in the given DB: creates the
// table if needed, reloads persisted blocks, seeds genesis when the table is
// empty, and installs the conflict-detecting persist + DB-authoritative height
// resync hooks. It is shared by Init (via the process singleton) and
// the multi-process collision tests, which build two independent chains over
// one DB file to exercise TASK-244 / ISS-242.
func bindChainToDB(chain *Chain, db *sql.DB) error {
	// No persistence or sync hook is installed unless the whole reload and
	// integrity check succeeds. This keeps a partially readable or invalid
	// ledger from accepting writes through a false-success path.
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
		return fmt.Errorf("create blocks table: %w", err)
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
		return fmt.Errorf("load persisted blocks: %w", err)
	}

	if !persisted {
		if err := seedGenesisIfEmpty(db, chain); err != nil {
			return fmt.Errorf("seed genesis block: %w", err)
		}
	}

	// The reload is the only point where production sees the persisted chain
	// before accepting writes. Verify heights, links, and proofs now rather
	// than starting a service on a chain that can only fail later during an
	// append or an on-demand integrity query.
	if report := chain.VerifyIntegrity(); !report.Valid {
		return fmt.Errorf(
			"%w: persisted blockchain is invalid at index %d: %s",
			ErrLedgerCorruption,
			report.FirstBrokenIndex,
			report.FirstBrokenReason,
		)
	}

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
		return insertBlock(db, b)
	}
	chain.removePersist = func(b *Block) error {
		return removeBlock(db, b)
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

	return nil
}

// loadBlocksFromDB reads every row of the blocks table into the chain's
// in-memory Blocks slice. It returns (true, nil) when at least one row existed
// (the genesis row counts), mirroring the pre-TASK-238 semantics that drove
// genesis seeding; a nil error means every row was decoded successfully. The
// height-0 row is validated against the in-memory genesis and then skipped
// rather than appended twice.
func loadBlocksFromDB(chain *Chain, db *sql.DB) (bool, error) {
	persisted := false
	rows, err := db.Query("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks ORDER BY height")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()

	// Buffer the reload so a corrupt row cannot leave the caller with a chain
	// that silently omits it while the row remains in SQLite.
	var loaded []*Block
	rowNumber := 0
	genesisSeen := false
	for rows.Next() {
		persisted = true
		rowNumber++
		block, err := scanBlockRow(rows)
		if err != nil {
			return persisted, fmt.Errorf("%w: scan block row %d: %w", ErrLedgerCorruption, rowNumber, err)
		}
		if block.Height == 0 {
			if len(chain.Blocks) == 0 {
				return persisted, fmt.Errorf("%w: in-memory chain has no genesis block", ErrLedgerCorruption)
			}
			if err := validatePersistedGenesis(chain.Blocks[0], block); err != nil {
				return persisted, fmt.Errorf("%w: %w", ErrLedgerCorruption, err)
			}
			genesisSeen = true
			continue
		}
		loaded = append(loaded, block)
	}
	if err := rows.Err(); err != nil {
		return persisted, err
	}
	if persisted && !genesisSeen {
		return persisted, fmt.Errorf("%w: persisted ledger has no genesis block at height 0", ErrLedgerCorruption)
	}
	chain.Blocks = append(chain.Blocks, loaded...)
	return persisted, nil
}

func validatePersistedGenesis(expected, persisted *Block) error {
	switch {
	case !bytes.Equal(expected.Hash, persisted.Hash):
		return errors.New("persisted genesis hash does not match")
	case !bytes.Equal(expected.PrevHash, persisted.PrevHash):
		return errors.New("persisted genesis previous hash does not match")
	case !bytes.Equal(expected.Data, persisted.Data):
		return errors.New("persisted genesis data does not match")
	case expected.Nonce != persisted.Nonce:
		return errors.New("persisted genesis nonce does not match")
	default:
		return nil
	}
}

// appendBlocksFromDB folds blocks a concurrent process committed to the shared
// DB into the in-memory chain, so the chain's length is the DB-authoritative
// next height. It is wired as Chain.syncFromDB (TASK-244, ISS-242).
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
func appendBlocksFromDB(chain *Chain, db *sql.DB) error {
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
			return fmt.Errorf("%w: scan appended block: %w", ErrLedgerCorruption, err)
		}
		if block.Height == 0 {
			continue
		}
		fresh = append(fresh, block)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(fresh) == 0 {
		return nil
	}

	// Verify the whole candidate before publishing any of it. A row can scan
	// successfully yet have a height gap, broken link, or invalid proof; adding
	// it first and checking later would expose a partial chain to nextBlock.
	candidate := &Chain{Blocks: make([]*Block, 0, len(chain.Blocks)+len(fresh))}
	candidate.Blocks = append(candidate.Blocks, chain.Blocks...)
	candidate.Blocks = append(candidate.Blocks, fresh...)
	if report := candidate.VerifyIntegrity(); !report.Valid {
		return fmt.Errorf(
			"%w: appended blocks failed verification at index %d: %s",
			ErrLedgerCorruption,
			report.FirstBrokenIndex,
			report.FirstBrokenReason,
		)
	}
	chain.Blocks = candidate.Blocks
	return nil
}

// rebuildChainFromDB resets the in-memory chain to the authoritative persisted
// ledger (genesis + every non-genesis block row, in height order). Any phantom
// tail blocks a persist failure left in memory are dropped: they were never
// committed, so the shared ledger does not know them. Used by appendBlocksFromDB
// when the in-memory tail diverges from the DB seam (TASK-244 review, ISS-242).
func rebuildChainFromDB(chain *Chain, db *sql.DB) error {
	// Build and verify a complete replacement before touching the live chain.
	// The old implementation truncated first, so a corrupt or unreadable row
	// left the caller with a partial ledger that appendBlock then treated as
	// best-effort-readable state.
	candidate := &Chain{}
	if len(chain.Blocks) > 0 {
		candidate.Blocks = []*Block{chain.Blocks[0]}
	} else {
		candidate.Blocks = []*Block{Genesis()}
	}
	if _, err := loadBlocksFromDB(candidate, db); err != nil {
		return err
	}
	if report := candidate.VerifyIntegrity(); !report.Valid {
		return fmt.Errorf(
			"%w: rebuilt ledger failed verification at index %d: %s",
			ErrLedgerCorruption,
			report.FirstBrokenIndex,
			report.FirstBrokenReason,
		)
	}
	chain.Blocks = candidate.Blocks
	return nil
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
// modernc.org/sqlite exposes the SQLite result code through *sqlite.Error,
// and an INTEGER PRIMARY KEY duplicate surfaces as
// SQLITE_CONSTRAINT_PRIMARYKEY (extended 1555) even though SQLite prints "UNIQUE
// constraint failed: blocks.height". Matching text would both break on driver
// message-format changes and misclassify a future unrelated UNIQUE constraint
// as a height collision (TASK-244 review, MEDIUM).
func sqliteIsUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return true
		}
	}
	return false
}

// seedGenesisIfEmpty writes the in-memory genesis block (Blocks[0]) to an
// empty blocks table. bindChainToDB calls it only when the reload observed no
// rows, including the height-0 genesis row. The old `len(Blocks) <= 1` guard
// stayed true after a reload and re-inserted height 0 on every boot, tripping
// the PRIMARY KEY and logging a false error (TASK-238, ISS-236).
func seedGenesisIfEmpty(db *sql.DB, chain *Chain) error {
	stmt, err := db.Prepare("INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare genesis insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	block := chain.Blocks[0]
	if _, err := stmt.Exec(block.Height, string(block.Hash), string(block.PrevHash), string(block.Data), block.Nonce, block.Timestamp, block.Timestamp); err != nil {
		return fmt.Errorf("insert genesis block: %w", err)
	}
	return nil
}
