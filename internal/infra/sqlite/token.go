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

// TryAddToSupply atomically adds amount to the token's
// total_supply column. The UPDATE is conditional on the token
// existing (WHERE id = ?) but does not check any other state —
// Mint is allowed to grow the supply unboundedly.
//
// This closes the TOCTOU window in TokenService.Mint: the
// pre-fix flow did GetToken → token.AddToSupply(in-memory
// increment) → SaveToken(full-row write). Two concurrent mints
// both read the same total_supply, both added their amount in
// memory, and the last SaveToken clobbered the other mint's
// increment — silently producing less total_supply than the
// sum of all mints.
func (r *TokenRepository) TryAddToSupply(id token.TokenID, amount *token.Amount) (*token.Amount, error) {
	res, err := r.db.Exec(`
		UPDATE tokens SET total_supply = CAST(total_supply AS INTEGER) + ?
		WHERE id = ?
	`, amount.String(), id)
	if err != nil {
		return nil, fmt.Errorf("try add to supply: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try add to supply rows: %w", err)
	}
	if affected == 0 {
		return nil, token.ErrTokenNotFound
	}
	updated, err := r.GetToken(id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, token.ErrTokenNotFound
	}
	return updated.TotalSupply(), nil
}

// TryDeductApproval atomically subtracts amount from the allowance
// (tokenID, owner, spender) and returns the new allowance amount. If
// the allowance is missing or less than amount, no change is made and
// token.ErrInsufficientAllowance is returned.
//
// This is the atomic primitive that closes the TOCTOU window in
// TransferFrom: the previous GetApproval → check → SaveApproval flow
// let two concurrent transfers read the same allowance, both pass the
// check, and both write back allowance - amount — silently allowing
// double-spend of the allowance.
//
// Concurrency: the conditional UPDATE is serialized by SQLite's
// per-connection write locking, so concurrent callers either succeed
// with a strictly-decreasing allowance or get token.ErrInsufficientAllowance.
// amount comparison uses SQLite's numeric ordering via the text
// representation, which is correct because amount is a decimal string.
func (r *TokenRepository) TryDeductApproval(tokenID token.TokenID, owner, spender token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	spenderB64 := base64.StdEncoding.EncodeToString(spender)
	amountStr := amount.String()

	// The conditional UPDATE is the atomicity primitive: SQLite's
	// CAST ensures the comparison is numeric (not lexicographic on
	// text), and only rows whose current amount is >= the requested
	// amount are touched. RowsAffected==0 means insufficient.
	res, err := r.db.Exec(`
		UPDATE allowances
		SET amount = CAST(amount AS INTEGER) - ?, updated_at = ?
		WHERE token_id = ? AND owner = ? AND spender = ?
		  AND CAST(amount AS INTEGER) >= ?
	`, amountStr, time.Now().Unix(), tokenID, ownerB64, spenderB64, amountStr)
	if err != nil {
		return nil, fmt.Errorf("try deduct approval: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try deduct approval rows: %w", err)
	}
	if affected == 0 {
		// Either the allowance doesn't exist or is < amount.
		// Distinguish so the caller can return the right HTTP code.
		if _, err := r.GetApproval(tokenID, owner, spender); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("try deduct approve: %w", token.ErrInsufficientAllowance)
	}

	// Re-read the new value so callers can pass it to SaveApproval
	// for audit. (We already wrote it, so this is informational.)
	updated, err := r.GetApproval(tokenID, owner, spender)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("approval vanished after deduct")
	}
	return updated.Amount(), nil
}

// TryAdjustApproval atomically applies a signed delta to the
// allowance (tokenID, owner, spender):
//
//   - delta > 0: amount becomes amount + delta (creating the row if
//     it does not exist with amount = delta).
//   - delta < 0: amount becomes MAX(0, amount + delta). That is,
//     subtracting more than the current allowance clamps to zero
//     rather than going negative (matches DecreaseAllowance
//     semantics).
//
// The whole operation is one SQLite statement (UPSERT), so
// concurrent callers cannot lose updates the way the
// read-modify-write path in IncreaseAllowance / DecreaseAllowance
// did. Returns the new allowance amount.
//
// Note: the case expression in the INSERT values handles the
// "decrease-only" path so we don't accidentally store a negative
// allowance when the row is being created by a decrement.
// Expiration is set to 0 (no expiry) on insert; the previous
// row's expiry is preserved on update.
func (r *TokenRepository) TryAdjustApproval(tokenID token.TokenID, owner, spender token.PublicKey, delta *token.Amount) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	spenderB64 := base64.StdEncoding.EncodeToString(spender)
	deltaStr := delta.String()
	id := fmt.Sprintf("%s-%s-%s", tokenID, ownerB64, spenderB64)
	now := time.Now().Unix()

	// amount stored on INSERT: clamp delta at 0 so a decrease-only
	// first-touch doesn't create a negative row.
	res, err := r.db.Exec(`
		INSERT INTO allowances (id, token_id, owner, spender, amount, expires_at, updated_at)
		VALUES (?, ?, ?, ?, CASE WHEN ? >= 0 THEN ? ELSE '0' END, 0, ?)
		ON CONFLICT(token_id, owner, spender) DO UPDATE SET
		  amount = MAX(0, CAST(amount AS INTEGER) + ?),
		  updated_at = ?
	`, id, tokenID, ownerB64, spenderB64, deltaStr, deltaStr, now, deltaStr, now)
	if err != nil {
		return nil, fmt.Errorf("try adjust approval: %w", err)
	}
	if _, err := res.RowsAffected(); err != nil {
		return nil, fmt.Errorf("try adjust approval rows: %w", err)
	}

	updated, err := r.GetApproval(tokenID, owner, spender)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("approval vanished after adjust")
	}
	return updated.Amount(), nil
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

func (r *TokenRepository) SetAccountBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) error {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	id := fmt.Sprintf("%s-%s", tokenID, ownerB64)

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO accounts (id, token_id, owner, balance, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, tokenID, ownerB64, amount.String(), time.Now().Unix())
	return err
}

// TrySubtractBalance atomically subtracts amount from the (tokenID,
// owner) account. Returns ErrInsufficientBalance if the current
// balance is less than amount.
//
// This closes the TOCTOU window in Transfer: the previous
// GetAccountBalance → check → SetAccountBalance(new) flow let two
// concurrent transfers both pass the check and both write back
// (balance - amount), silently allowing the same funds to be spent
// twice.
//
// SQLite's CAST(... AS INTEGER) caps comparisons at int64 (≈9.2e18);
// that is many orders of magnitude beyond any realistic token
// supply, so it is safe here. For amounts larger than int64 the
// caller should use a different storage backend.
func (r *TokenRepository) TrySubtractBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	amountStr := amount.String()

	res, err := r.db.Exec(`
		UPDATE accounts
		SET balance = CAST(balance AS INTEGER) - ?, updated_at = ?
		WHERE token_id = ? AND owner = ?
		  AND CAST(balance AS INTEGER) >= ?
	`, amountStr, time.Now().Unix(), tokenID, ownerB64, amountStr)
	if err != nil {
		return nil, fmt.Errorf("try subtract balance: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try subtract balance rows: %w", err)
	}
	if affected == 0 {
		// Either the account doesn't exist or has insufficient
		// funds. Distinguish so the caller can return the right
		// error. We wrap the domain sentinel so callers can use
		// errors.Is(err, token.ErrInsufficientBalance) regardless
		// of which layer the error originated from.
		cur, err := r.GetAccountBalance(tokenID, owner)
		if err != nil {
			return nil, err
		}
		if cur.Int.Cmp(amount.Int) < 0 {
			return nil, fmt.Errorf("try subtract balance: %w", token.ErrInsufficientBalance)
		}
		return nil, fmt.Errorf("try subtract balance: unexpected zero rows affected")
	}

	updated, err := r.GetAccountBalance(tokenID, owner)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// TryAddBalance atomically adds amount to the (tokenID, owner)
// account, creating the account row if it doesn't exist. Returns
// the new balance.
//
// This is the atomic primitive used by Mint and the credit side of
// Transfer. The previous flow read GetAccountBalance, computed
// newBalance = current + amount, then SetAccountBalance(newBalance)
// — two concurrent Mints to the same account could both pass the
// read and one would write (current + amount) while the other
// wrote (current + amount), losing one Mint's worth of credit.
func (r *TokenRepository) TryAddBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)
	id := fmt.Sprintf("%s-%s", tokenID, ownerB64)
	amountStr := amount.String()

	// INSERT ... ON CONFLICT DO UPDATE handles both the "create new
	// account" and "increment existing account" cases in one
	// statement, so there is no read-then-write window for a race.
	_, err := r.db.Exec(`
		INSERT INTO accounts (id, token_id, owner, balance, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			balance = CAST(balance AS INTEGER) + excluded.balance,
			updated_at = excluded.updated_at
	`, id, tokenID, ownerB64, amountStr, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("try add balance: %w", err)
	}

	return r.GetAccountBalance(tokenID, owner)
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
