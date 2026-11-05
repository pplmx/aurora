// The token repository: the store's schema, the token rows themselves, and the
// construction and transaction plumbing every method shares. The allowance and
// balance primitives that the token service also needs live in
// token_approvals.go and token_balances.go, so each file can be read as one
// question rather than as a table-by-table dump.
package sqlite

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/utils"
	_ "modernc.org/sqlite"
)

// tokenExec abstracts *sql.DB and *sql.Tx so TokenRepository can run either
// against the pool or inside an explicit transaction.
type tokenExec interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type TokenRepository struct {
	db     *sql.DB
	dbPath string
	exec   tokenExec // nil => use db
}

// q returns the executor this repository reads/writes through: the pooled DB
// by default, or the active *sql.Tx when WithTx was used.
func (r *TokenRepository) q() tokenExec {
	if r != nil && r.exec != nil {
		return r.exec
	}
	return r.db
}

// WithTx returns a TokenRepository whose operations run inside the given
// transaction. All reads and writes through the returned repository observe
// the transaction's uncommitted state and participate in its commit/rollback.
func (r *TokenRepository) WithTx(tx *sql.Tx) token.Repository {
	return &TokenRepository{db: r.db, dbPath: r.dbPath, exec: tx}
}

func NewTokenRepository(dbPath string) (*TokenRepository, error) {
	if err := utils.EnsureSQLiteDirectory(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite", DSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &TokenRepository{
		db:     database,
		dbPath: dbPath,
	}

	if err := repo.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	if err := utils.RestrictSQLiteFiles(dbPath); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to secure database files: %w", err)
	}

	return repo, nil
}

func (r *TokenRepository) createTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := r.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			symbol TEXT NOT NULL,
			total_supply TEXT NOT NULL,
			decimals INTEGER DEFAULT 8,
			owner TEXT NOT NULL,
			is_mintable INTEGER DEFAULT 1,
			is_burnable INTEGER DEFAULT 1,
			created_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			owner TEXT NOT NULL,
			balance TEXT NOT NULL DEFAULT '0',
			updated_at INTEGER,
			UNIQUE(token_id, owner)
		)`,
		`CREATE TABLE IF NOT EXISTS allowances (
			id TEXT PRIMARY KEY,
			token_id TEXT NOT NULL,
			owner TEXT NOT NULL,
			spender TEXT NOT NULL,
			amount TEXT NOT NULL DEFAULT '0',
			expires_at INTEGER,
			updated_at INTEGER,
			UNIQUE(token_id, owner, spender)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_token_owner ON accounts(token_id, owner)`,
		`CREATE INDEX IF NOT EXISTS idx_allowances_owner ON allowances(token_id, owner, spender)`,
		// The collection listing reads one token page newest first. Without
		// this composite index SQLite materialises and sorts the whole table
		// before LIMIT/OFFSET can apply (measured: "USE TEMP B-TREE FOR ORDER
		// BY"). Migration 000006 creates the same index, so a fresh database
		// and a migrated one agree.
		`CREATE INDEX IF NOT EXISTS idx_tokens_created ON tokens(created_at DESC, id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

// SaveToken insert-only: a token's ID is its symbol, so a create racing a
// concurrent create of the same symbol must resolve atomically, not via the
// pre-check in the service (which is a read-then-write TOCTOU window; two
// racers both read nil and the second INSERT OR REPLACE would silently wipe
// the first token's row and owner balance — ISS-098). ON CONFLICT DO NOTHING
// makes the DB the arbiter: RowsAffected == 0 means the ID already exists,
// surfaced to the caller as ErrTokenExists so the whole create transaction
// rolls back cleanly.
func (r *TokenRepository) SaveToken(t *token.Token) error {
	ownerB64 := base64.StdEncoding.EncodeToString(t.Owner())
	totalSupplyJSON := t.TotalSupply().String()

	res, err := r.q().Exec(`
		INSERT INTO tokens (id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, t.ID(), t.Name(), t.Symbol(), totalSupplyJSON, t.Decimals(), ownerB64, boolToInt(t.IsMintable()), boolToInt(t.IsBurnable()), t.CreatedAt().Unix())
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return token.ErrTokenExists
	}
	return nil
}

func (r *TokenRepository) GetToken(id token.TokenID) (*token.Token, error) {
	row := r.q().QueryRow("SELECT id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at FROM tokens WHERE id = ?", id)

	var idStr, name, symbol, totalSupplyStr, ownerB64 string
	var decimals int8
	var isMintable, isBurnable int
	var createdAt int64

	err := row.Scan(&idStr, &name, &symbol, &totalSupplyStr, &decimals, &ownerB64, &isMintable, &isBurnable, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, token.ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	owner, err := base64.StdEncoding.DecodeString(ownerB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode owner: %w", err)
	}
	amount, err := token.NewAmountFromString(totalSupplyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode amount: %w", err)
	}

	// Reconstruct with the persisted attributes. Using NewToken here would
	// silently reset is_mintable/is_burnable to true and created_at to now,
	// which could authorize mint/burn on tokens that were created as
	// non-mintable/non-burnable and corrupt audit timestamps.
	return token.NewTokenFromRecord(
		token.TokenID(idStr),
		name,
		symbol,
		amount,
		owner,
		decimals,
		isMintable != 0,
		isBurnable != 0,
		time.Unix(createdAt, 0),
	), nil
}

// ListTokens returns one page of tokens, newest first. The scan mirrors
// GetToken so a listed token carries the same attributes a fetched one does.
// limit <= 0 means "all rows from offset", matching the NFT repository.
func (r *TokenRepository) ListTokens(limit, offset int) ([]*token.Token, error) {
	return r.listTokens("", nil, limit, offset)
}

// ListMatchingTokens searches the stored tokens by id, name and symbol, so a
// search finds a token whose row is not on the loaded page. The escaping and the
// case-folding mirror the lottery and audit searches: a symbol containing % or _
// must still match literally.
func (r *TokenRepository) ListMatchingTokens(term string, limit, offset int) ([]*token.Token, error) {
	like, ok := likeContains(term)
	if !ok {
		return r.ListTokens(limit, offset)
	}
	where := `WHERE lower(id) LIKE ? ESCAPE '\' OR lower(name) LIKE ? ESCAPE '\'
		OR lower(symbol) LIKE ? ESCAPE '\'`
	return r.listTokens(where, []any{like, like, like}, limit, offset)
}

// listTokens is the one paged read: ListTokens and ListMatchingTokens differ only
// in the WHERE clause, and both keep the newest-first order.
func (r *TokenRepository) listTokens(where string, args []any, limit, offset int) ([]*token.Token, error) {
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.q().Query(`
		SELECT id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at
		FROM tokens `+where+` ORDER BY created_at DESC, id
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*token.Token
	for rows.Next() {
		var idStr, name, symbol, totalSupplyStr, ownerB64 string
		var decimals int8
		var isMintable, isBurnable int
		var createdAt int64
		if err := rows.Scan(&idStr, &name, &symbol, &totalSupplyStr, &decimals, &ownerB64, &isMintable, &isBurnable, &createdAt); err != nil {
			return nil, err
		}
		owner, err := base64.StdEncoding.DecodeString(ownerB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode owner: %w", err)
		}
		amount, err := token.NewAmountFromString(totalSupplyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode amount: %w", err)
		}
		out = append(out, token.NewTokenFromRecord(
			token.TokenID(idStr), name, symbol, amount, owner, decimals,
			isMintable != 0, isBurnable != 0, time.Unix(createdAt, 0),
		))
	}
	return out, rows.Err()
}

func (r *TokenRepository) TrySubtractFromSupply(tokenID token.TokenID, amount *token.Amount) (*token.Amount, error) {
	amountStr := amount.String()
	id := string(tokenID)

	res, err := r.q().Exec(`
		UPDATE tokens
		SET total_supply = CAST(total_supply AS INTEGER) - ?
		WHERE id = ? AND CAST(total_supply AS INTEGER) >= ?
	`, amountStr, id, amountStr)
	if err != nil {
		return nil, fmt.Errorf("try subtract supply: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try subtract supply rows: %w", err)
	}
	if affected == 0 {
		// Supply is already below the amount being burned — a ledger
		// invariant violation rather than a client input error, so it
		// must surface as a server-side failure, not a 4xx.
		return nil, fmt.Errorf("try subtract supply: total supply below burn amount")
	}
	t, err := r.GetToken(tokenID)
	if err != nil {
		return nil, err
	}
	return t.TotalSupply(), nil
}

func (r *TokenRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *TokenRepository) GetDB() *sql.DB {
	return r.db
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
