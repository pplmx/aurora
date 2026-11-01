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
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestTokenRepository_SaveToken_ConcurrentDuplicateCreate is the regression
// test for the CreateToken read-then-write TOCTOU (ISS-098): CreateToken
// pre-checked GetToken outside any transaction then wrote the row with
// INSERT OR REPLACE, so two concurrent creates of the same symbol both passed
// the pre-check and the second REPLACE silently destroyed the first token's
// row and owner balance. SaveToken is now insert-only with the DB as the
// arbiter (ON CONFLICT DO NOTHING → ErrTokenExists), so exactly one racing
// create wins and, crucially, the winner's row survives intact.
func TestTokenRepository_SaveToken_ConcurrentDuplicateCreate(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	mgr := NewTxManager(repo.GetDB())
	owner := token.PublicKey(make([]byte, 32))

	const workers = 16
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, conflicts := 0, 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mgr.WithTransaction(func(tx *sql.Tx) error {
				r := repo.WithTx(tx)
				tok := token.NewToken(token.TokenID("RACE"), "Race", "RACE", token.NewAmount(1000), owner)
				if err := r.SaveToken(tok); err != nil {
					return err
				}
				return r.SetAccountBalance("RACE", owner, token.NewAmount(1000))
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else if errors.Is(err, token.ErrTokenExists) {
				conflicts++
			}
		}()
	}
	wg.Wait()

	require.Equal(t, 1, successes, "exactly one racing create must succeed")
	require.Equal(t, workers-1, conflicts, "every other racing create must observe the duplicate")

	// The winner's row must be intact, not blanked by a REPLACE from a loser.
	got, err := repo.GetToken("RACE")
	require.NoError(t, err)
	require.Equal(t, int64(1000), got.TotalSupply().Int64(), "surviving token's supply must be the winner's full amount")
	bal, err := repo.GetAccountBalance("RACE", owner)
	require.NoError(t, err)
	require.Equal(t, int64(1000), bal.Int64(), "surviving token's owner balance must be the winner's full amount")
}

// TestTokenRepository_SaveToken_SequentialDuplicateRejects enforces the
// insert-only contract even without concurrency: a second SaveToken for an
// existing ID returns ErrTokenExists instead of overwriting.
func TestTokenRepository_SaveToken_SequentialDuplicateRejects(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey(make([]byte, 32))
	tok := token.NewToken(token.TokenID("DUP"), "Dup", "DUP", token.NewAmount(5), owner)
	require.NoError(t, repo.SaveToken(tok))

	err := repo.SaveToken(token.NewToken(token.TokenID("DUP"), "Dup2", "DUP", token.NewAmount(99), owner))
	require.ErrorIs(t, err, token.ErrTokenExists)

	got, err := repo.GetToken("DUP")
	require.NoError(t, err)
	require.Equal(t, int64(5), got.TotalSupply().Int64(), "original row must be untouched by the rejected duplicate")
}

// TestTokenRepository_TwoPoolsShareOneFile covers the shape a single-process
// goroutine test cannot reach: two independent *sql.DB pools over the same
// database file. Two pools share no Go state — no mutex, no shared connection —
// and contend only through SQLite's file lock, which is what two CLI processes
// or a CLI and the API do against one aurora.db. That is a real deployment
// shape (a cron job and a server), and it is the one where a missing
// _busy_timeout hurts most: the loser of the lock used to fail instantly rather
// than wait.
//
// The hardening lives in dsn.go, so this test fails if either pool is opened
// with a bare path.
func TestTokenRepository_TwoPoolsShareOneFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "shared.db")

	first, err := NewTokenRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Close() })

	owner := token.PublicKey(make([]byte, 32))
	recipient := token.PublicKey(make([]byte, 32))
	recipient[0] = 1
	require.NoError(t, first.SaveToken(
		token.NewToken(token.TokenID("TWO"), "Two", "TWO", token.NewAmount(1000), owner)))
	require.NoError(t, first.SetAccountBalance("TWO", owner, token.NewAmount(1000)))

	// The second pool opens the same file after schema setup, the way a second
	// process would find an existing database.
	second, err := NewTokenRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })

	repos := []*TokenRepository{first, second}
	const rounds = 20
	amount := token.NewAmount(1)

	var wg sync.WaitGroup
	errs := make(chan error, len(repos)*rounds)
	for i := 0; i < rounds; i++ {
		for _, repo := range repos {
			wg.Add(1)
			go func(r *TokenRepository) {
				defer wg.Done()
				mgr := NewTxManager(r.GetDB())
				errs <- mgr.WithTransaction(func(tx *sql.Tx) error {
					txRepo := r.WithTx(tx)
					if _, err := txRepo.TrySubtractBalance("TWO", owner, amount); err != nil {
						return err
					}
					_, err := txRepo.TryAddBalance("TWO", recipient, amount)
					return err
				})
			}(repo)
		}
	}
	wg.Wait()
	close(errs)

	// No "database is locked": every write either waited for the file lock or
	// followed a completed one.
	for err := range errs {
		require.NoError(t, err, "two pools over one file must not fail on the file lock")
	}

	// The moves are all-or-nothing, so the totals must still add up: a lost
	// update would break this even when no error surfaced.
	sender, err := first.GetAccountBalance("TWO", owner)
	require.NoError(t, err)
	receiver, err := second.GetAccountBalance("TWO", recipient)
	require.NoError(t, err)

	moved := int64(rounds * len(repos))
	assert.Equal(t, int64(1000)-moved, sender.Int64(), "every debit must have applied exactly once")
	assert.Equal(t, moved, receiver.Int64(), "every credit must have applied exactly once")
	assert.Equal(t, int64(1000), sender.Int64()+receiver.Int64(), "no value created or destroyed")
}
