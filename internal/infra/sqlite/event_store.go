package sqlite

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/token"
)

type TokenEventStore struct {
	db *sql.DB
}

func NewTokenEventStore(dbPath string) (*TokenEventStore, error) {
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	es := &TokenEventStore{db: database}

	if err := es.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return es, nil
}

func (e *TokenEventStore) createTables() error {
	if _, err := e.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := e.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS transfer_events (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			from_owner TEXT NOT NULL,
			to_owner TEXT NOT NULL,
			amount TEXT NOT NULL,
			nonce INTEGER NOT NULL,
			signature TEXT,
			block_height INTEGER,
			timestamp INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS mint_events (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			to_owner TEXT NOT NULL,
			amount TEXT NOT NULL,
			block_height INTEGER,
			timestamp INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS burn_events (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			from_owner TEXT NOT NULL,
			amount TEXT NOT NULL,
			block_height INTEGER,
			timestamp INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS approve_events (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			owner TEXT NOT NULL,
			spender TEXT NOT NULL,
			amount TEXT NOT NULL,
			timestamp INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_transfers_from ON transfer_events(token_id, from_owner)`,
		`CREATE INDEX IF NOT EXISTS idx_transfers_to ON transfer_events(token_id, to_owner)`,
		`CREATE INDEX IF NOT EXISTS idx_transfers_nonce ON transfer_events(token_id, from_owner, nonce)`,
	}

	for _, query := range queries {
		if _, err := e.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (e *TokenEventStore) SaveTransferEvent(event *token.TransferEvent) error {
	fromB64 := base64.StdEncoding.EncodeToString(event.From())
	toB64 := base64.StdEncoding.EncodeToString(event.To())
	sigB64 := base64.StdEncoding.EncodeToString(event.Signature())

	_, err := e.db.Exec(`
		INSERT INTO transfer_events (id, token_id, from_owner, to_owner, amount, nonce, signature, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID(), event.TokenID(), fromB64, toB64, event.Amount().String(), event.Nonce(), sigB64, event.BlockHeight(), event.Timestamp().Unix())
	return err
}

func (e *TokenEventStore) SaveMintEvent(event *token.MintEvent) error {
	toB64 := base64.StdEncoding.EncodeToString(event.To())

	_, err := e.db.Exec(`
		INSERT INTO mint_events (id, token_id, to_owner, amount, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID(), event.TokenID(), toB64, event.Amount().String(), event.Timestamp().Unix())
	return err
}

func (e *TokenEventStore) SaveBurnEvent(event *token.BurnEvent) error {
	fromB64 := base64.StdEncoding.EncodeToString(event.From())

	_, err := e.db.Exec(`
		INSERT INTO burn_events (id, token_id, from_owner, amount, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID(), event.TokenID(), fromB64, event.Amount().String(), event.Timestamp().Unix())
	return err
}

func (e *TokenEventStore) SaveApproveEvent(event *token.ApproveEvent) error {
	ownerB64 := base64.StdEncoding.EncodeToString(event.Owner())
	spenderB64 := base64.StdEncoding.EncodeToString(event.Spender())

	_, err := e.db.Exec(`
		INSERT INTO approve_events (id, token_id, owner, spender, amount, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID(), event.TokenID(), ownerB64, spenderB64, event.Amount().String(), event.Timestamp().Unix())
	return err
}

func (e *TokenEventStore) GetLastNonce(tokenID token.TokenID, owner token.PublicKey) (uint64, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	row := e.db.QueryRow("SELECT MAX(nonce) FROM transfer_events WHERE token_id = ? AND from_owner = ?", tokenID, ownerB64)

	var nonce sql.NullInt64
	err := row.Scan(&nonce)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !nonce.Valid {
		return 0, nil
	}
	return uint64(nonce.Int64), nil
}

func (e *TokenEventStore) SaveNonce(tokenID token.TokenID, owner token.PublicKey, nonce uint64) error {
	return nil
}

func (e *TokenEventStore) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.TransferEvent, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	rows, err := e.db.Query(`
		SELECT id, token_id, from_owner, to_owner, amount, nonce, signature, block_height, timestamp
		FROM transfer_events
		WHERE token_id = ? AND (from_owner = ? OR to_owner = ?)
		ORDER BY timestamp DESC
	`, tokenID, ownerB64, ownerB64)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*token.TransferEvent
	for rows.Next() {
		var id, tkID, fromB64, toB64, amountStr string
		var nonce uint64
		var sigB64 string
		var blockHeight, ts int64

		if err := rows.Scan(&id, &tkID, &fromB64, &toB64, &amountStr, &nonce, &sigB64, &blockHeight, &ts); err != nil {
			return nil, err
		}

		from, _ := base64.StdEncoding.DecodeString(fromB64)
		to, _ := base64.StdEncoding.DecodeString(toB64)
		sig, _ := base64.StdEncoding.DecodeString(sigB64)
		amount, _ := token.NewAmountFromString(amountStr)

		event := token.NewTransferEvent(token.TokenID(tkID), from, to, amount, nonce, sig)
		events = append(events, event)
	}

	return events, rows.Err()
}

func (e *TokenEventStore) GetTransferEventsByToken(tokenID token.TokenID) ([]*token.TransferEvent, error) {
	rows, err := e.db.Query(`
		SELECT id, token_id, from_owner, to_owner, amount, nonce, signature, block_height, timestamp
		FROM transfer_events
		WHERE token_id = ?
		ORDER BY timestamp DESC
	`, tokenID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*token.TransferEvent
	for rows.Next() {
		var id, tkID, fromB64, toB64, amountStr string
		var nonce uint64
		var sigB64 string
		var blockHeight, ts int64

		if err := rows.Scan(&id, &tkID, &fromB64, &toB64, &amountStr, &nonce, &sigB64, &blockHeight, &ts); err != nil {
			return nil, err
		}

		from, _ := base64.StdEncoding.DecodeString(fromB64)
		to, _ := base64.StdEncoding.DecodeString(toB64)
		sig, _ := base64.StdEncoding.DecodeString(sigB64)
		amount, _ := token.NewAmountFromString(amountStr)

		event := token.NewTransferEvent(token.TokenID(tkID), from, to, amount, nonce, sig)
		events = append(events, event)
	}

	return events, rows.Err()
}

func (e *TokenEventStore) GetMintEventsByToken(tokenID token.TokenID) ([]*token.MintEvent, error) {
	rows, err := e.db.Query(`
		SELECT id, token_id, to_owner, amount, block_height, timestamp
		FROM mint_events
		WHERE token_id = ?
		ORDER BY timestamp DESC
	`, tokenID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*token.MintEvent
	for rows.Next() {
		var id, tkID, toB64, amountStr string
		var blockHeight, ts int64

		if err := rows.Scan(&id, &tkID, &toB64, &amountStr, &blockHeight, &ts); err != nil {
			return nil, err
		}

		to, _ := base64.StdEncoding.DecodeString(toB64)
		amount, _ := token.NewAmountFromString(amountStr)

		event := token.NewMintEvent(token.TokenID(tkID), to, amount)
		events = append(events, event)
	}

	return events, rows.Err()
}

func (e *TokenEventStore) GetBurnEventsByToken(tokenID token.TokenID) ([]*token.BurnEvent, error) {
	rows, err := e.db.Query(`
		SELECT id, token_id, from_owner, amount, block_height, timestamp
		FROM burn_events
		WHERE token_id = ?
		ORDER BY timestamp DESC
	`, tokenID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*token.BurnEvent
	for rows.Next() {
		var id, tkID, fromB64, amountStr string
		var blockHeight, ts int64

		if err := rows.Scan(&id, &tkID, &fromB64, &amountStr, &blockHeight, &ts); err != nil {
			return nil, err
		}

		from, _ := base64.StdEncoding.DecodeString(fromB64)
		amount, _ := token.NewAmountFromString(amountStr)

		event := token.NewBurnEvent(token.TokenID(tkID), from, amount)
		events = append(events, event)
	}

	return events, rows.Err()
}

func (e *TokenEventStore) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}
