package sqlite

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/token"
)

type TokenRepository struct {
	db     *sql.DB
	dbPath string
}

func NewTokenRepository(dbPath string) (*TokenRepository, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
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
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (r *TokenRepository) SaveToken(t *token.Token) error {
	ownerB64 := base64.StdEncoding.EncodeToString(t.Owner())
	totalSupplyJSON := t.TotalSupply().String()

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO tokens (id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID(), t.Name(), t.Symbol(), totalSupplyJSON, t.Decimals(), ownerB64, boolToInt(t.IsMintable()), boolToInt(t.IsBurnable()), t.CreatedAt().Unix())
	return err
}

func (r *TokenRepository) GetToken(id token.TokenID) (*token.Token, error) {
	row := r.db.QueryRow("SELECT id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at FROM tokens WHERE id = ?", id)

	var idStr, name, symbol, totalSupplyStr, ownerB64 string
	var decimals int8
	var isMintable, isBurnable int
	var createdAt int64

	err := row.Scan(&idStr, &name, &symbol, &totalSupplyStr, &decimals, &ownerB64, &isMintable, &isBurnable, &createdAt)
	if err == sql.ErrNoRows {
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

	return token.NewToken(
		token.TokenID(idStr),
		name,
		symbol,
		amount,
		owner,
	), nil
}

func (r *TokenRepository) SaveApproval(approval *token.Approval) error {
	ownerB64 := base64.StdEncoding.EncodeToString(approval.Owner())
	spenderB64 := base64.StdEncoding.EncodeToString(approval.Spender())
	amountStr := approval.Amount().String()
	id := fmt.Sprintf("%s-%s-%s", approval.TokenID(), ownerB64, spenderB64)

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO allowances (id, token_id, owner, spender, amount, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, approval.TokenID(), ownerB64, spenderB64, amountStr, approval.ExpiresAt().Unix(), time.Now().Unix())
	return err
}

func (r *TokenRepository) GetApproval(tokenID token.TokenID, owner, spender token.PublicKey) (*token.Approval, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	spenderB64 := base64.StdEncoding.EncodeToString(spender)

	row := r.db.QueryRow("SELECT token_id, owner, spender, amount, expires_at FROM allowances WHERE token_id = ? AND owner = ? AND spender = ?",
		tokenID, ownerB64, spenderB64)

	var tkID, ownB64, spendB64, amountStr string
	var expiresAt int64

	err := row.Scan(&tkID, &ownB64, &spendB64, &amountStr, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	own, err := base64.StdEncoding.DecodeString(ownB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode owner: %w", err)
	}
	spend, err := base64.StdEncoding.DecodeString(spendB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode spender: %w", err)
	}
	amount, err := token.NewAmountFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode amount: %w", err)
	}

	return token.NewApproval(token.TokenID(tkID), own, spend, amount), nil
}

func (r *TokenRepository) GetAccountBalance(tokenID token.TokenID, owner token.PublicKey) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	row := r.db.QueryRow("SELECT balance FROM accounts WHERE token_id = ? AND owner = ?", tokenID, ownerB64)

	var balanceStr string
	err := row.Scan(&balanceStr)
	if err == sql.ErrNoRows {
		return token.NewAmount(0), nil
	}
	if err != nil {
		return nil, err
	}

	amount, err := token.NewAmountFromString(balanceStr)
	if err != nil {
		return token.NewAmount(0), err
	}
	return amount, nil
}

func (r *TokenRepository) UpdateBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) error {
	return r.SetAccountBalance(tokenID, owner, amount)
}

func (r *TokenRepository) SetAccountBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) error {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	id := fmt.Sprintf("%s-%s", tokenID, ownerB64)

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO accounts (id, token_id, owner, balance, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, tokenID, ownerB64, amount.String(), time.Now().Unix())
	return err
}

func (r *TokenRepository) GetApprovalsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.Approval, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	rows, err := r.db.Query("SELECT token_id, owner, spender, amount, expires_at FROM allowances WHERE token_id = ? AND owner = ?",
		tokenID, ownerB64)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var approvals []*token.Approval
	for rows.Next() {
		var tkID, ownB64, spendB64, amountStr string
		var expiresAt int64

		if err := rows.Scan(&tkID, &ownB64, &spendB64, &amountStr, &expiresAt); err != nil {
			return nil, err
		}

		own, err := base64.StdEncoding.DecodeString(ownB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode owner: %w", err)
		}
		spend, err := base64.StdEncoding.DecodeString(spendB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode spender: %w", err)
		}
		amount, err := token.NewAmountFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode amount: %w", err)
		}

		approvals = append(approvals, token.NewApproval(token.TokenID(tkID), own, spend, amount))
	}

	return approvals, rows.Err()
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
