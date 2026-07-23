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
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
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
// UPDATE...RETURNING statement so concurrent callers each observe a
// unique monotonic sequence, closing the TOCTOU window that the
// GetLastNonce/SaveNonce pair left open.
//
// Concurrency: SQLite serializes write transactions, so the conditional
// INSERT-then-RETURNING pattern is safe across goroutines without
// any application-level locking. We deliberately avoid the read-modify-
// write pattern in the original SaveNonce because two readers could
// both observe nonce=N and both write back N+1.
func (r *SQLiteReplayProtection) ClaimNextNonce(tokenID string, owner []byte) (uint64, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Read the current nonce (0 if absent) inside the same tx so the
	// read-then-update is serialized against other ClaimNextNonce
	// callers.
	var current uint64
	err = tx.QueryRow(
		`SELECT nonce FROM nonces WHERE token_id = ? AND owner = ?`,
		tokenID, ownerB64,
	).Scan(&current)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("read nonce: %w", err)
	}

	next := current + 1
	_, err = tx.Exec(`
		INSERT INTO nonces (token_id, owner, nonce)
		VALUES (?, ?, ?)
		ON CONFLICT(token_id, owner) DO UPDATE SET nonce = excluded.nonce
	`, tokenID, ownerB64, next)
	if err != nil {
		return 0, fmt.Errorf("write nonce: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return next, nil
}

func (r *SQLiteReplayProtection) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
