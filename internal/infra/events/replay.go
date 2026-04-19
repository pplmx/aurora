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

func (r *SQLiteReplayProtection) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
