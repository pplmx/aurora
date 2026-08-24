package events

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type ReplayProtection interface {
	GetLastNonce(tokenID string, owner []byte) (uint64, error)
	SaveNonce(tokenID string, owner []byte, nonce uint64) error
	// ClaimNextNonce atomically returns a nonce strictly greater than
	// the previously-stored nonce for (tokenID, owner). Implementations
	// MUST be concurrency-safe (e.g. via a single conditional UPDATE)
	// so that concurrent callers each receive a unique nonce. This is
	// the primitive that closes the TOCTOU race in Transfer/TransferFrom
	// where GetLastNonce+increment+SaveNonce allowed two concurrent
	// requests to sign with the same nonce.
	ClaimNextNonce(tokenID string, owner []byte) (uint64, error)
}

type SQLiteReplayProtection struct {
	db *sql.DB
}

func NewSQLiteReplayProtection(dbPath string) (*SQLiteReplayProtection, error) {
	// _busy_timeout: when several connections contend for the write lock (a
	// real pool, i.e. SetMaxOpenConns > 1), a claimant whose atomic UPSERT
	// finds the lock momentarily held waits up to 5s instead of erroring out
	// with SQLITE_BUSY. Without it the concurrent nonce claim would
	// intermittently fail under load (ISS-075).
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON&_busy_timeout=5000", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	rp := &SQLiteReplayProtection{db: database}

	if err := rp.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return rp, nil
}

func (r *SQLiteReplayProtection) createTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := r.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	query := `CREATE TABLE IF NOT EXISTS nonces (
		token_id TEXT NOT NULL,
		owner TEXT NOT NULL,
		nonce INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (token_id, owner)
	)`

	if _, err := r.db.Exec(query); err != nil {
		return err
	}
	return nil
}

func (r *SQLiteReplayProtection) GetLastNonce(tokenID string, owner []byte) (uint64, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	row := r.db.QueryRow("SELECT nonce FROM nonces WHERE token_id = ? AND owner = ?", tokenID, ownerB64)

	var nonce uint64
	err := row.Scan(&nonce)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (r *SQLiteReplayProtection) SaveNonce(tokenID string, owner []byte, nonce uint64) error {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	_, err := r.db.Exec(`
		INSERT INTO nonces (token_id, owner, nonce)
		VALUES (?, ?, ?)
		ON CONFLICT(token_id, owner) DO UPDATE SET nonce = excluded.nonce
	`, tokenID, ownerB64, nonce)
	return err
}

// ClaimNextNonce atomically increments the nonce for (tokenID, owner)
// and returns the new value. The increment+read happens in a single
// UPSERT...RETURNING statement so concurrent callers each observe a
// unique monotonic sequence, closing the TOCTOU window that the
// GetLastNonce/SaveNonce pair left open.
//
// Concurrency: SQLite executes each standalone statement atomically, and a
// single write statement (whose result it reads back) cannot interleave, so
// this is safe across connections without any application-level locking.
// We deliberately avoid the read-modify-write pattern — the original
// BEGIN + SELECT + INSERT ON CONFLICT run as a *deferred* transaction that
// takes its read snapshot before it holds the write lock, so two callers on
// a real connection pool both read nonce=N and the loser hits SQLITE_BUSY
// (SQLITE_BUSY_SNAPSHOT in WAL mode) — or, without WAL, silently writes a
// duplicate nonce and signs two transfers with it (ISS-075). Evaluating
// `nonces.nonce + 1` against the stored row inside DO UPDATE makes every
// caller increment the current high-water mark exactly once.
func (r *SQLiteReplayProtection) ClaimNextNonce(tokenID string, owner []byte) (uint64, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	var next uint64
	err := r.db.QueryRow(`
		INSERT INTO nonces (token_id, owner, nonce)
		VALUES (?, ?, 1)
		ON CONFLICT(token_id, owner) DO UPDATE SET nonce = nonces.nonce + 1
		RETURNING nonce
	`, tokenID, ownerB64).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("claim next nonce: %w", err)
	}

	return next, nil
}

func (r *SQLiteReplayProtection) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
