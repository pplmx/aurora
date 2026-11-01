// The allowance (approval) half of the token repository. It lives beside the
// balance primitives because both are conditional-UPDATE atomicity, not
// because they share a table: every function here answers "may this spender
// move this amount", and the conditional UPDATE is what makes concurrent
// callers either succeed with a strictly-decreasing allowance or fail.
package sqlite

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/domain/token"
)

func (r *TokenRepository) SaveApproval(approval *token.Approval) error {
	ownerB64 := base64.StdEncoding.EncodeToString(approval.Owner())
	spenderB64 := base64.StdEncoding.EncodeToString(approval.Spender())
	amountStr := approval.Amount().String()
	id := fmt.Sprintf("%s-%s-%s", approval.TokenID(), ownerB64, spenderB64)

	_, err := r.q().Exec(`
		INSERT OR REPLACE INTO allowances (id, token_id, owner, spender, amount, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, approval.TokenID(), ownerB64, spenderB64, amountStr, approval.ExpiresAt().Unix(), time.Now().Unix())
	return err
}

// maxInt64String is SQLite's largest signed 64-bit integer, used as an
// arithmetic ceiling for the CAST-based primitives. Amounts are individually
// capped below this by token.ValidateAmount, but cumulative totals (supply,
// balances) can still reach it over many operations; past that boundary
// SQLite's INTEGER arithmetic silently converts to IEEE-754 REAL and stores an
// unparseable float (e.g. "9.223372036854776e+18") that token.NewAmountFromString
// cannot read back. The conditional bounds below make such overflows a hard
// error instead of corrupting the ledger.
const maxInt64String = "9223372036854775807"

// TryAddToSupply atomically adds amount to the token's
// total_supply column. The UPDATE is conditional on the token
// existing AND on the resulting total_supply staying within signed
// 64-bit range; an increment that would overflow is rejected.
//
// This closes the TOCTOU window in TokenService.Mint: the
// pre-fix flow did GetToken → token.AddToSupply(in-memory
// increment) → SaveToken(full-row write). Two concurrent mints
// both read the same total_supply, both added their amount in
// memory, and the last SaveToken clobbered the other mint's
// increment — silently producing less total_supply than the
// sum of all mints.
func (r *TokenRepository) TryAddToSupply(id token.TokenID, amount *token.Amount) (*token.Amount, error) {
	res, err := r.q().Exec(`
		UPDATE tokens SET total_supply = CAST(total_supply AS INTEGER) + ?
		WHERE id = ?
		  AND CAST(total_supply AS INTEGER) <= (? - ?)
	`, amount.String(), id, maxInt64String, amount.String())
	if err != nil {
		return nil, fmt.Errorf("try add to supply: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("try add to supply rows: %w", err)
	}
	if affected == 0 {
		// Either the token is missing or the increment would overflow
		// MaxInt64. Distinguish so callers surface the right error and
		// (critically) never mint onto an already-saturated supply.
		existing, err := r.GetToken(id)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, token.ErrTokenNotFound
		}
		return nil, fmt.Errorf("try add to supply: total supply would exceed maximum")
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
	res, err := r.q().Exec(`
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
		return nil, fmt.Errorf("try deduct approval: %w", token.ErrInsufficientAllowance)
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
	//
	// The comparison casts the bound delta to INTEGER. `amount` is a TEXT
	// column and the driver binds deltaStr as TEXT, and SQLite orders every
	// TEXT value above every INTEGER: `'-100' >= 0` is TRUE, so the uncast
	// form stored '-100' and handed it back to the caller. CAST makes the
	// sign test numeric, which is what the CASE was always meant to be.
	//
	// On update, clamp between 0 and MaxInt64: the MAX keeps decreases at 0
	// and the MIN (CAST to INTEGER so the ceiling compares numerically, not
	// as text>numeric per SQLite type ordering) prevents an increase past
	// MaxInt64 from silently becoming an unparseable REAL float.
	res, err := r.q().Exec(`
		INSERT INTO allowances (id, token_id, owner, spender, amount, expires_at, updated_at)
		VALUES (?, ?, ?, ?, CASE WHEN CAST(? AS INTEGER) >= 0 THEN ? ELSE '0' END, 0, ?)
		ON CONFLICT(token_id, owner, spender) DO UPDATE SET
		  amount = MIN(CAST(? AS INTEGER), MAX(0, CAST(amount AS INTEGER) + CAST(? AS INTEGER))),
		  updated_at = ?
	`, id, tokenID, ownerB64, spenderB64, deltaStr, deltaStr, now, maxInt64String, deltaStr, now)
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

	row := r.q().QueryRow("SELECT token_id, owner, spender, amount, expires_at FROM allowances WHERE token_id = ? AND owner = ? AND spender = ?",
		tokenID, ownerB64, spenderB64)

	var tkID, ownB64, spendB64, amountStr string
	var expiresAt int64

	err := row.Scan(&tkID, &ownB64, &spendB64, &amountStr, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
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
