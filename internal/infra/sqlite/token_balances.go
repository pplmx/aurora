// The account-balance half of the token repository. Its writes are conditional
// UPDATEs for the same reason the supply and allowance writes are: a
// read-check-write pair let two concurrent transfers both observe a balance
// and the second clobber the first. SQLite's numeric CAST keeps the comparison
// arithmetic rather than lexicographic on the text columns.
package sqlite

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/domain/token"
)

func (r *TokenRepository) GetAccountBalance(tokenID token.TokenID, owner token.PublicKey) (*token.Amount, error) {
	ownerB64 := base64.StdEncoding.EncodeToString(owner)

	row := r.q().QueryRow("SELECT balance FROM accounts WHERE token_id = ? AND owner = ?", tokenID, ownerB64)

	var balanceStr string
	err := row.Scan(&balanceStr)
	if errors.Is(err, sql.ErrNoRows) {
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

	_, err := r.q().Exec(`
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

	res, err := r.q().Exec(`
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
	//
	// The WHERE clause on DO UPDATE prevents incrementing an existing
	// balance past MaxInt64: a skipped update reports RowsAffected()==0
	// (verified empirically), and since a fresh insert always reports 1,
	// affected==0 unambiguously means "the account exists but the
	// increment would overflow" — refuse rather than write an
	// unparseable float or silently clamp.
	res, err := r.q().Exec(`
		INSERT INTO accounts (id, token_id, owner, balance, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			balance = CAST(balance AS INTEGER) + excluded.balance,
			updated_at = excluded.updated_at
		WHERE CAST(balance AS INTEGER) <= (? - excluded.balance)
	`, id, tokenID, ownerB64, amountStr, time.Now().Unix(), maxInt64String)
	if err != nil {
		return nil, fmt.Errorf("try add balance: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try add balance rows: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("try add balance: balance would exceed maximum")
	}

	return r.GetAccountBalance(tokenID, owner)
}
