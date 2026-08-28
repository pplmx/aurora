package token

import (
	"math"
	"math/big"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPrimitiveRepo builds an inmemRepo with the given token registered so the
// supply primitives have something to operate on.
func newPrimitiveRepo(t *testing.T, supply int64) (*inmemRepo, token.TokenID, token.PublicKey) {
	t.Helper()
	repo := &inmemRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}
	id := token.TokenID("tok-1")
	pub := token.PublicKey("owner-key")
	repo.tokens[id] = token.NewToken(id, "Test", "TST", token.NewAmount(supply), pub)
	return repo, id, pub
}

func TestInmemRepo_TrySubtractFromSupply(t *testing.T) {
	repo, id, _ := newPrimitiveRepo(t, 100)

	// unknown token surfaces the sentinel.
	_, err := repo.TrySubtractFromSupply("nope", token.NewAmount(1))
	assert.ErrorIs(t, err, token.ErrTokenNotFound)

	// burning more than the supply violates the ledger invariant.
	_, err = repo.TrySubtractFromSupply(id, token.NewAmount(101))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below burn amount")

	// happy path decrements total_supply.
	after, err := repo.TrySubtractFromSupply(id, token.NewAmount(40))
	require.NoError(t, err)
	assert.Equal(t, "60", after.String())
	assert.Equal(t, int64(60), repo.tokens[id].TotalSupply().Int64())
}

func TestInmemRepo_TryDeductApproval(t *testing.T) {
	repo := &inmemRepo{approvals: make(map[string]*token.Approval)}
	id := token.TokenID("tok-1")
	owner := token.PublicKey("owner")
	spender := token.PublicKey("spender")
	repo.approvals[string(id)+string(owner)+string(spender)] =
		token.NewApproval(id, owner, spender, token.NewAmount(100))

	// no approval for an unknown spender.
	_, err := repo.TryDeductApproval(id, owner, token.PublicKey("other"), token.NewAmount(1))
	assert.ErrorIs(t, err, token.ErrInsufficientAllowance)

	// deduction exceeding the approved amount.
	_, err = repo.TryDeductApproval(id, owner, spender, token.NewAmount(200))
	assert.ErrorIs(t, err, token.ErrInsufficientAllowance)

	// happy path decrements the allowance.
	after, err := repo.TryDeductApproval(id, owner, spender, token.NewAmount(30))
	require.NoError(t, err)
	assert.Equal(t, "70", after.String())
}

func TestInmemRepo_TryAdjustApproval(t *testing.T) {
	repo := &inmemRepo{approvals: make(map[string]*token.Approval)}
	id := token.TokenID("tok-1")
	owner := token.PublicKey("owner")
	spender := token.PublicKey("spender")

	// no prior approval: delta becomes the new allowance.
	amt, err := repo.TryAdjustApproval(id, owner, spender, token.NewAmount(50))
	require.NoError(t, err)
	assert.Equal(t, "50", amt.String())

	// positive delta on an existing approval accumulates.
	amt, err = repo.TryAdjustApproval(id, owner, spender, token.NewAmount(10))
	require.NoError(t, err)
	assert.Equal(t, "60", amt.String())

	// pushing below zero clamps to 0 instead of going negative.
	amt, err = repo.TryAdjustApproval(id, owner, spender, token.NewAmount(-100))
	require.NoError(t, err)
	assert.Equal(t, "0", amt.String())

	// pushing past MaxInt64 clamps at MaxInt64 (mirrors the SQLite ceiling).
	max := token.NewAmount(math.MaxInt64)
	amt, err = repo.TryAdjustApproval(id, owner, spender,
		&token.Amount{Int: new(big.Int).SetInt64(math.MaxInt64)})
	require.NoError(t, err)
	assert.Equal(t, max.String(), amt.String())
}

func TestInmemRepo_TryAddBalance_OverflowRejected(t *testing.T) {
	repo, id, pub := newPrimitiveRepo(t, 100)
	key := string(id) + string(pub)
	repo.balances[key] = token.NewAmount(math.MaxInt64)

	_, err := repo.TryAddBalance(id, pub, token.NewAmount(1))
	require.Error(t, err, "balance past MaxInt64 must be rejected")
	assert.Contains(t, err.Error(), "exceed maximum")
}

func TestInmemRepo_TryAddToSupply_OverflowRejected(t *testing.T) {
	repo, id, pub := newPrimitiveRepo(t, 100)
	repo.tokens[id] = token.NewToken(id, "Test", "TST", token.NewAmount(math.MaxInt64), pub)

	_, err := repo.TryAddToSupply(id, token.NewAmount(1))
	require.Error(t, err, "supply past MaxInt64 must be rejected")
	assert.Contains(t, err.Error(), "exceed maximum")
}

func TestInmemEventBus_Subscribe_SubscribeAll(t *testing.T) {
	bus := newInmemEventBus(&inmemEventStore{})
	handler := infraevents.Handler(func(events.Event) error { return nil })

	// The in-memory TUI bus registers no handlers; both must return a harmless
	// no-op unsubscribe closure rather than nil.
	unsub := bus.Subscribe("token.transfer", handler)
	assert.NotNil(t, unsub)
	unsub()
	unsub = bus.SubscribeAll(handler)
	assert.NotNil(t, unsub)
	unsub()
}

func TestInmemRepo_WithTx(t *testing.T) {
	repo, _, _ := newPrimitiveRepo(t, 10)
	txRepo := repo.WithTx(nil)
	assert.Same(t, repo, txRepo, "the TUI repo is its own transaction-scoped repository")
}

func TestMintView_RendersWithToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "mint"
	app.currentToken = token.NewToken("tok-1", "Test", "TST", token.NewAmount(100), app.ownerKey)
	view := app.mintView()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "TST")
}

func TestTransferView_RendersWithToken(t *testing.T) {
	app := NewTokenApp()
	app.view = "transfer"
	app.currentToken = token.NewToken("tok-1", "Test", "TST", token.NewAmount(100), app.ownerKey)
	view := app.transferView()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, i18n.GetText("token.tui.from_label"))
}

func TestLoadHistory_NoToken(t *testing.T) {
	app := NewTokenApp()
	app.currentToken = nil
	app.loadHistory() // must not panic; sets viewport to the no-token hint
	assert.NotEmpty(t, app.historyView())
}

func TestLoadHistory_Empty(t *testing.T) {
	app := NewTokenApp()
	app.currentToken = token.NewToken("tok-1", "Test", "TST", token.NewAmount(100), app.ownerKey)
	app.loadHistory()
	assert.NotEmpty(t, app.historyView())
}
