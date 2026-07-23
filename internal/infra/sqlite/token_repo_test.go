package sqlite

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/token"
)

func setupTokenTestDB(t *testing.T) (*TokenRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_token.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create token repository: %v", err)
	}

	cleanup := func() {
		if repo != nil {
			_ = repo.Close()
		}
		_ = os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewTokenRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
	if repo.db == nil {
		t.Fatal("Database connection should not be nil")
	}
}

func TestTokenRepository_SaveToken(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	testToken := token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), token.PublicKey([]byte("owner")))

	err := repo.SaveToken(testToken)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}
}

func TestTokenRepository_GetToken(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	testToken := token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), token.PublicKey([]byte("owner")))

	err := repo.SaveToken(testToken)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	retrieved, err := repo.GetToken("TEST")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Token should not be nil")
	}

	if retrieved.ID() != "TEST" {
		t.Errorf("Expected token ID 'TEST', got '%s'", retrieved.ID())
	}

	if retrieved.Name() != "Test Token" {
		t.Errorf("Expected token name 'Test Token', got '%s'", retrieved.Name())
	}
}

func TestTokenRepository_GetToken_NotFound(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	_, err := repo.GetToken("NOTEXIST")
	if err != token.ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestTokenRepository_SaveApproval(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))
	approval := token.NewApproval("TEST", owner, spender, token.NewAmount(500))

	err := repo.SaveApproval(approval)
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}
}

func TestTokenRepository_GetApproval(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))
	approval := token.NewApproval("TEST", owner, spender, token.NewAmount(500))

	err := repo.SaveApproval(approval)
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	retrieved, err := repo.GetApproval("TEST", owner, spender)
	if err != nil {
		t.Fatalf("Failed to get approval: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Approval should not be nil")
	}

	if retrieved.Amount().String() != "500" {
		t.Errorf("Expected amount '500', got '%s'", retrieved.Amount().String())
	}
}

func TestTokenRepository_GetApproval_NotFound(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))

	retrieved, err := repo.GetApproval("TEST", owner, spender)
	if err != nil {
		t.Fatalf("Failed to get approval: %v", err)
	}

	if retrieved != nil {
		t.Error("Approval should be nil for non-existent approval")
	}
}

func TestTokenRepository_GetApprovalsByOwner(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	owner := token.PublicKey([]byte("owner"))
	spender1 := token.PublicKey([]byte("spender1"))
	spender2 := token.PublicKey([]byte("spender2"))

	err := repo.SaveApproval(token.NewApproval("TEST", owner, spender1, token.NewAmount(100)))
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	err = repo.SaveApproval(token.NewApproval("TEST2", owner, spender2, token.NewAmount(200)))
	if err != nil {
		t.Fatalf("Failed to save approval: %v", err)
	}

	approvals, err := repo.GetApprovalsByOwner("TEST", owner)
	if err != nil {
		t.Fatalf("Failed to get approvals: %v", err)
	}

	if len(approvals) != 1 {
		t.Errorf("Expected 1 approval, got %d", len(approvals))
	}
}

func TestTokenRepository_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewTokenRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Double close should not fail: %v", err)
	}
}

// TestTokenRepository_TryDeductApproval_ConcurrentNoDoubleSpend
// proves the conditional UPDATE in TryDeductApproval closes the TOCTOU
// window that previously let two concurrent TransferFrom calls both
// deduct the same allowance.
//
// Setup: owner grants spender an allowance of 100. Two concurrent
// transfers of 60 each should result in exactly one success (100-60=40)
// and one token.ErrInsufficientAllowance (40 < 60). With the old
// GetApproval → check → SaveApproval pattern, both would succeed and
// the allowance would end at -20 (or wrap-around for unsigned types).
func TestTokenRepository_TryDeductApproval_ConcurrentNoDoubleSpend(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	// Serialize through one connection (same :memory: caveat as the
	// voting concurrent test).
	repo.db.SetMaxOpenConns(1)

	const owner = "owner-1"
	const spender = "spender-1"
	const tokenID = "tok-1"

	ownerBytes := []byte(owner)
	spenderBytes := []byte(spender)

	// Grant allowance of 100.
	hundred := token.NewAmount(100)
	require.NoError(t, repo.SaveApproval(token.NewApproval(
		token.TokenID(tokenID), ownerBytes, spenderBytes, hundred,
	)))

	const goroutines = 8 // > 1 would have double-spaced under the bug
	results := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			amount := token.NewAmount(60)
			_, err := repo.TryDeductApproval(
				token.TokenID(tokenID), ownerBytes, spenderBytes, amount,
			)
			results <- err
		}()
	}

	var success, insufficient int
	for i := 0; i < goroutines; i++ {
		err := <-results
		switch err {
		case nil:
			success++
		default:
			if errors.Is(err, token.ErrInsufficientAllowance) {
				insufficient++
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}

	assert.Equal(t, 1, success, "exactly one goroutine should succeed")
	assert.Equal(t, goroutines-1, insufficient, "all others should be insufficient")

	// Allowance must end at 100 - 60 = 40, not negative.
	final, err := repo.GetApproval(token.TokenID(tokenID), ownerBytes, spenderBytes)
	require.NoError(t, err)
	require.NotNil(t, final)
	assert.Equal(t, int64(40), final.Amount().Int.Int64(),
		"final allowance must be exactly 40, not negative (which would indicate double-spend)")
}

// TestTokenRepository_TryDeductApproval_InsufficientReturnsError
// proves the primitive returns token.ErrInsufficientAllowance for missing
// or insufficient allowances.
func TestTokenRepository_TryDeductApproval_InsufficientReturnsError(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()
	repo.db.SetMaxOpenConns(1)

	t.Run("missing allowance", func(t *testing.T) {
		_, err := repo.TryDeductApproval(
			"tok-1", []byte("owner"), []byte("spender"), token.NewAmount(10),
		)
		assert.ErrorIs(t, err, token.ErrInsufficientAllowance)
	})

	t.Run("amount exceeds allowance", func(t *testing.T) {
		require.NoError(t, repo.SaveApproval(token.NewApproval(
			"tok-2", []byte("o"), []byte("s"), token.NewAmount(5),
		)))
		_, err := repo.TryDeductApproval(
			"tok-2", []byte("o"), []byte("s"), token.NewAmount(10),
		)
		assert.ErrorIs(t, err, token.ErrInsufficientAllowance)
	})
}

// TestTokenRepository_TrySubtractBalance_ConcurrentNoDoubleSpend
// proves the conditional UPDATE in TrySubtractBalance closes the
// TOCTOU window in Transfer: two concurrent transfers of 60 from
// the same account (balance 100) must result in exactly one success
// and one token.ErrInsufficientBalance.
func TestTokenRepository_TrySubtractBalance_ConcurrentNoDoubleSpend(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()
	repo.db.SetMaxOpenConns(1)

	tokenID := token.TokenID("tok-1")
	owner := []byte("owner")
	require.NoError(t, repo.SetAccountBalance(tokenID, owner, token.NewAmount(100)))

	const goroutines = 8
	results := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := repo.TrySubtractBalance(tokenID, owner, token.NewAmount(60))
			results <- err
		}()
	}

	var success, insufficient int
	for i := 0; i < goroutines; i++ {
		switch err := <-results; {
		case err == nil:
			success++
		case errors.Is(err, token.ErrInsufficientBalance):
			insufficient++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 1, success, "exactly one goroutine should succeed")
	assert.Equal(t, goroutines-1, insufficient)

	final, err := repo.GetAccountBalance(tokenID, owner)
	require.NoError(t, err)
	assert.Equal(t, int64(40), final.Int.Int64(),
		"final balance must be exactly 40, not negative (which would indicate double-spend)")
}

// TestTokenRepository_TryAddBalance_ConcurrentCreatesOrIncrements
// proves INSERT ... ON CONFLICT DO UPDATE handles both the create
// and increment cases without a race.
func TestTokenRepository_TryAddBalance_ConcurrentCreatesOrIncrements(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()
	repo.db.SetMaxOpenConns(1)

	tokenID := token.TokenID("tok-2")
	owner := []byte("recipient")

	const goroutines = 8
	done := make(chan struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := repo.TryAddBalance(tokenID, owner, token.NewAmount(10))
			assert.NoError(t, err)
			done <- struct{}{}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}

	final, err := repo.GetAccountBalance(tokenID, owner)
	require.NoError(t, err)
	assert.Equal(t, int64(80), final.Int.Int64(),
		"8 goroutines adding 10 each must produce exactly 80, not 10 (lost updates)")
}

// TestTokenRepository_TryAdjustApproval_ConcurrentIncrements proves
// the atomic UPSERT in TryAdjustApproval closes the TOCTOU window
// that previously made IncreaseAllowance silently lose concurrent
// increments. The pre-fix code did GetApproval → compute →
// SaveApproval, so 16 concurrent +10 increments from allowance 50
// would produce anything from 60 to 160; the post-fix code MUST
// produce exactly 50 + 16*10 = 210.
func TestTokenRepository_TryAdjustApproval_ConcurrentIncrements(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewTokenRepository(filepath.Join(dir, "tokens.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	tokenID := token.TokenID("TEST")
	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))

	// Seed allowance = 50 directly via the repo's own primitive.
	_, err = repo.TryAdjustApproval(tokenID, owner, spender, token.NewAmount(50))
	require.NoError(t, err)

	// 16 concurrent +10 increments. Pre-fix: lost-update bug means
	// the final allowance is somewhere in [60, 210], not 210.
	// Post-fix: exactly 210.
	const goroutines = 16
	const delta = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, err := repo.TryAdjustApproval(tokenID, owner, spender, token.NewAmount(int64(delta))); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent increment returned error: %v", err)
	}

	final, err := repo.GetApproval(tokenID, owner, spender)
	require.NoError(t, err)
	require.NotNil(t, final)
	expected := int64(50 + goroutines*delta)
	if got := final.Amount().Int64(); got != expected {
		t.Fatalf("after %d concurrent +%d increments: allowance = %d, want %d (lost-update bug)",
			goroutines, delta, got, expected)
	}
}

// TestTokenRepository_TryAdjustApproval_ConcurrentDecrements is the
// dual of the increment test: 16 concurrent -5 decrements from
// allowance 100 must produce exactly 100 - 16*5 = 20, not less
// (over-decrement) and not more (under-decrement / lost update).
func TestTokenRepository_TryAdjustApproval_ConcurrentDecrements(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewTokenRepository(filepath.Join(dir, "tokens.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	tokenID := token.TokenID("TEST")
	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))

	_, err = repo.TryAdjustApproval(tokenID, owner, spender, token.NewAmount(100))
	require.NoError(t, err)

	const goroutines = 16
	const delta = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Express "subtract 5" as -5 (the primitive takes a signed delta).
			neg := token.NewAmount(int64(-delta))
			_, _ = repo.TryAdjustApproval(tokenID, owner, spender, neg)
		}()
	}
	wg.Wait()

	final, err := repo.GetApproval(tokenID, owner, spender)
	require.NoError(t, err)
	require.NotNil(t, final)
	expected := int64(100 - goroutines*delta)
	if got := final.Amount().Int64(); got != expected {
		t.Fatalf("after %d concurrent -%d decrements: allowance = %d, want %d",
			goroutines, delta, got, expected)
	}
}

// TestTokenRepository_TryAdjustApproval_ClampAtZeroOnOverDecrement
// ensures the primitive mirrors DecreaseAllowance semantics: trying
// to subtract more than the current allowance clamps at zero
// rather than producing a negative stored value.
func TestTokenRepository_TryAdjustApproval_ClampAtZeroOnOverDecrement(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewTokenRepository(filepath.Join(dir, "tokens.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	tokenID := token.TokenID("TEST")
	owner := token.PublicKey([]byte("owner"))
	spender := token.PublicKey([]byte("spender"))

	_, err = repo.TryAdjustApproval(tokenID, owner, spender, token.NewAmount(30))
	require.NoError(t, err)

	// Try to subtract 100 (signed -100) — must clamp to 0.
	neg := token.NewAmount(int64(-100))
	newAmt, err := repo.TryAdjustApproval(tokenID, owner, spender, neg)
	require.NoError(t, err)
	if newAmt.Sign() != 0 {
		t.Errorf("over-decrement should clamp at 0, got %s", newAmt.String())
	}

	// And re-read confirms.
	final, err := repo.GetApproval(tokenID, owner, spender)
	require.NoError(t, err)
	if got := final.Amount().Int64(); got != 0 {
		t.Errorf("stored allowance after over-decrement = %d, want 0", got)
	}
}

// TestTokenRepository_TrySubtractBalance_ConcurrentBurnsNoOverdraw
// proves the TrySubtractBalance atomic primitive prevents
// concurrent burns from overdrawing an account.
//
// This is the regression test for Round 31's Burn TOCTOU fix.
// Round 20 added TrySubtractBalance to fix Transfer/Mint/
// TransferFrom, but missed Burn — which still used
// GetAccountBalance → Cmp(amount) → SetAccountBalance. Two
// concurrent burns of 60 from balance 100 both observed 100,
// both passed Cmp(100) >= 60, both wrote 40 — silent
// overdraw of 20.
//
// Post-fix: every burn goes through TrySubtractBalance's
// conditional UPDATE (WHERE CAST(balance AS INTEGER) >= ?).
// Exactly 1 of the 2 racing burns succeeds; the other gets
// token.ErrInsufficientBalance.
func TestTokenRepository_TrySubtractBalance_ConcurrentBurnsNoOverdraw(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewTokenRepository(filepath.Join(dir, "tokens.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	tokenID := token.TokenID("BURN")
	owner := token.PublicKey([]byte("burner"))

	// Seed an account with 100 tokens by minting a token and
	// crediting the owner the full supply via TryAddBalance
	// (Round 20's other atomic primitive).
	_, err = repo.TryAddBalance(tokenID, owner, token.NewAmount(100))
	require.NoError(t, err)

	// 2 concurrent burns of 60 each — total would be 120 from
	// balance 100. Pre-fix: both succeed, balance becomes 40
	// (overdraw of 20 silently accepted). Post-fix: exactly 1
	// succeeds, 1 gets token.ErrInsufficientBalance, balance is 40.
	const goroutines = 2
	const burnEach = 60
	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, results[idx] = repo.TrySubtractBalance(tokenID, owner, token.NewAmount(int64(burnEach)))
		}(i)
	}
	wg.Wait()

	successes := 0
	insufficients := 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, token.ErrInsufficientBalance):
			insufficients++
		default:
			t.Errorf("unexpected error from TrySubtractBalance: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful burn, got %d (lost-update or overdraw bug): results=%v", successes, results)
	}
	if insufficients != 1 {
		t.Fatalf("expected exactly 1 token.ErrInsufficientBalance, got %d: results=%v", insufficients, results)
	}

	finalBalance, err := repo.GetAccountBalance(tokenID, owner)
	require.NoError(t, err)
	expected := int64(100 - burnEach)
	if got := finalBalance.Int64(); got != expected {
		t.Errorf("final balance = %d, want %d (overdraw bug: %d extra burned)", got, expected, got-expected)
	}
}

// TestTokenRepository_TryAddToSupply_ConcurrentNoLostUpdate is the
// regression test for the Round 32 Mint TOCTOU fix.
//
// Pre-fix behaviour: TokenService.Mint did GetToken →
// token.AddToSupply (in-memory increment) → SaveToken (full-row
// write). Two concurrent mints of +100 from total_supply=1000
// both observed 1000, both computed 1100 in memory, and both
// wrote 1100 — the second mint was silently lost. Final
// total_supply: 1100. Audit log: 2 mint events. Sum of all
// mint amounts: 200. The sum exceeds total_supply, an
// invariant violation.
//
// Post-fix behaviour: Mint uses TryAddToSupply's atomic UPDATE
// (SET total_supply = CAST(total_supply AS INTEGER) + ? WHERE
// id = ?). 8 concurrent +100 mints from total_supply=1000 must
// produce exactly 1000 + 8*100 = 1800.
func TestTokenRepository_TryAddToSupply_ConcurrentNoLostUpdate(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewTokenRepository(filepath.Join(dir, "tokens.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	tokenID := token.TokenID("MINT")
	owner := []byte("minter")

	// Seed: create a token with total_supply=1000 directly via
	// the SQLite layer (CreateToken service flow is exercised
	// in TestCreateToken elsewhere; this test focuses on the
	// atomic increment primitive).
	_, err = repo.db.Exec(`
		INSERT INTO tokens (id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tokenID, "Test", "TST", "1000", 8, base64.StdEncoding.EncodeToString(owner), 1, 1, time.Now().Unix())
	require.NoError(t, err)

	const goroutines = 8
	const mintEach = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errCh := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, err := repo.TryAddToSupply(tokenID, token.NewAmount(int64(mintEach))); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent mint returned error: %v", err)
	}

	got, err := repo.GetToken(tokenID)
	require.NoError(t, err)
	require.NotNil(t, got)
	expected := int64(1000 + goroutines*mintEach)
	if supply := got.TotalSupply().Int64(); supply != expected {
		t.Fatalf("after %d concurrent +%d mints: total_supply = %d, want %d (lost-update bug)",
			goroutines, mintEach, supply, expected)
	}
}

func TestTokenRepository_GetDB(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	require.NotNil(t, repo.GetDB(), "GetDB should return non-nil database handle")
}
