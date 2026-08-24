package sqlite

// Regression for ISS-076 (v1.70): the multi-statement WithTransaction flow
// (deferred BEGIN → reads → writes) over the service's real, unlimited
// connection pool used to fail a large fraction of concurrent transfers with
// "database is locked" (SQLITE_BUSY) — the write lock was only taken mid-way
// after a read snapshot, and the DSN had no _busy_timeout. The hardened DSN
// (dsn.go) serializes writers at BEGIN IMMEDIATE and waits up to 5s for the
// lock, so every concurrent transaction completes.

import (
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/token"
)

// TestTokenRepository_WithTransaction_ConcurrentNoSQLiteBusy runs
// transfer-shaped read-then-write transactions concurrently over an unlimited
// pool and asserts every one of them commits: no SQLITE_BUSY, and the ledger
// accounting is exact (owner starts with workers tokens, each transfer moves 1
// to the recipient).
//
// Pre-fix (bare "?_foreign_keys=ON" DSN) this failed ~10 of 16 workers with
// "database is locked"; with the hardened DSN all workers commit.
func TestTokenRepository_WithTransaction_ConcurrentNoSQLiteBusy(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey(make([]byte, 32))
	recipient := token.PublicKey(make([]byte, 32))
	recipient[0] = 1

	const workers = 24
	tok := token.NewToken(token.TokenID("C1"), "Concurrent", "C1", token.NewAmount(workers), owner)
	require.NoError(t, repo.SaveToken(tok))
	require.NoError(t, repo.SetAccountBalance("C1", owner, token.NewAmount(workers)))

	mgr := NewTxManager(repo.GetDB())
	amount := token.NewAmount(1)

	var wg sync.WaitGroup
	var mu sync.Mutex
	busy := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mgr.WithTransaction(func(tx *sql.Tx) error {
				r := repo.WithTx(tx)
				if _, err := r.GetToken("C1"); err != nil {
					return err
				}
				if _, err := r.GetAccountBalance("C1", owner); err != nil {
					return err
				}
				if _, err := r.TrySubtractBalance("C1", owner, amount); err != nil {
					return err
				}
				_, err := r.TryAddBalance("C1", recipient, amount)
				return err
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				msg := err.Error()
				if strings.Contains(msg, "database is locked") || strings.Contains(msg, "SQLITE_BUSY") {
					busy++
				}
			}
		}()
	}
	wg.Wait()

	require.Zerof(t, busy, "concurrent writer transactions must not fail with SQLITE_BUSY over the pooled DSN")
	// every transfer must have committed: the owner's balance is fully drained
	// and the recipient holds exactly `workers` tokens.
	ownerBal, err := repo.GetAccountBalance("C1", owner)
	require.NoError(t, err)
	require.Zero(t, ownerBal.Int64(), "owner balance must be fully drained (all transfers committed)")
	recipientBal, err := repo.GetAccountBalance("C1", recipient)
	require.NoError(t, err)
	require.Equal(t, int64(workers), recipientBal.Int64(), "recipient must collect exactly one token per committed transfer")
}
