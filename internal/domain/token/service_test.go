package token

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/events"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

// keyPairs caches ed25519 key pairs keyed by a small integer so that
// pubKey(n) / privKey(n) always return matching public/private keys
// across a test. This replaces the old approach of using make([]byte, 32)
// as a fake public key with make([]byte, 64) as a fake private key,
// which broke after the authorization middleware was added (the keys
// now must be a real, matching ed25519 pair).
var (
	keyPairs   = make(map[byte]ed25519.PrivateKey)
	keyPairsMu sync.Mutex
)

func keyPair(n byte) ed25519.PrivateKey {
	keyPairsMu.Lock()
	defer keyPairsMu.Unlock()
	if k, ok := keyPairs[n]; ok {
		return k
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("failed to generate ed25519 key pair: %v", err))
	}
	keyPairs[n] = priv
	_ = pub // pub is derivable from priv
	return priv
}

func pubKey(n byte) PublicKey {
	return PublicKey(keyPair(n).Public().(ed25519.PublicKey))
}

func privKey(n byte) PrivateKey {
	return PrivateKey(keyPair(n))
}

type mockTxManager struct {
	repo       *mockRepository
	shouldFail bool
	failStep   int
	step       int
}

func (m *mockTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	if m.repo != nil {
		m.repo.beginTx()
	}

	if m.shouldFail {
		m.step++
		if m.step == m.failStep {
			if m.repo != nil {
				m.repo.rollbackTx()
			}
			return fmt.Errorf("transaction failed at step %d", m.failStep)
		}
	}

	err := fn(nil)

	if err != nil && m.repo != nil {
		m.repo.rollbackTx()
		return err
	}

	if m.repo != nil {
		m.repo.commitTx()
	}

	return err
}

func newMockTxManager() *mockTxManager {
	return &mockTxManager{}
}

func newMockTxManagerWithRepo(repo *mockRepository) *mockTxManager {
	return &mockTxManager{repo: repo}
}

func newFailingTxManager(failStep int) *mockTxManager {
	return &mockTxManager{shouldFail: true, failStep: failStep}
}

func TestCreateToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       pubKey(1),
	}

	token, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	if token.Name() != "Test Token" {
		t.Errorf("expected name Test Token, got %s", token.Name())
	}
	if token.Symbol() != "TEST" {
		t.Errorf("expected symbol TEST, got %s", token.Symbol())
	}
}

// TestCreateToken_RejectsDuplicateSymbol locks the v1.62 data-loss fix: a
// token's ID is its symbol and persistence uses INSERT OR REPLACE keyed on it,
// so a second create with an existing symbol must be rejected (ErrTokenExists)
// rather than silently overwriting the first token and its balances
// (TASK-075, ISS-067).
func TestCreateToken_RejectsDuplicateSymbol(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.CreateToken(&CreateTokenRequest{
		Name: "First", Symbol: "DUP", TotalSupply: NewAmount(1000), Owner: pubKey(1),
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = service.CreateToken(&CreateTokenRequest{
		Name: "Second", Symbol: "DUP", TotalSupply: NewAmount(5000), Owner: pubKey(2),
	})
	if err != ErrTokenExists {
		t.Fatalf("second create with duplicate symbol: got %v, want ErrTokenExists", err)
	}

	// The first token must be intact (not overwritten by the rejected create).
	first, err := service.GetTokenInfo("DUP")
	if err != nil {
		t.Fatalf("GetTokenInfo failed: %v", err)
	}
	if first.Name() != "First" || first.TotalSupply().String() != "1000" {
		t.Errorf("first token was clobbered: got %q supply %s, want name First supply 1000",
			first.Name(), first.TotalSupply().String())
	}
}

func TestCreateToken_InvalidName(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       pubKey(1),
	}

	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateToken_InvalidSymbol(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "",
		TotalSupply: NewAmount(1000),
		Owner:       pubKey(1),
	}

	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error for empty symbol")
	}
}

func TestCreateToken_InvalidTotalSupply(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(0),
		Owner:       pubKey(1),
	}

	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error for zero total supply")
	}
}

func TestCreateToken_InvalidOwner(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       nil,
	}

	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error for nil owner")
	}
}

func TestGetTokenInfo(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	tok, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	info, err := service.GetTokenInfo(tok.ID())
	if err != nil {
		t.Fatalf("GetTokenInfo failed: %v", err)
	}
	if info.Name() != "Test Token" {
		t.Errorf("expected Test Token, got %s", info.Name())
	}
	if info.Symbol() != "TEST" {
		t.Errorf("expected TEST, got %s", info.Symbol())
	}
	if info.TotalSupply().Int64() != 1000 {
		t.Errorf("expected 1000, got %d", info.TotalSupply().Int64())
	}
}

func TestGetTokenInfo_NotFound(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	info, err := service.GetTokenInfo("NONEXISTENT")
	if err != nil {
		t.Fatalf("GetTokenInfo should not return error, got: %v", err)
	}
	if info != nil {
		t.Error("expected nil token for nonexistent token")
	}
}

func TestGetBalance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	balance, err := service.GetBalance("TEST", owner)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance.Int64() != 1000 {
		t.Errorf("expected balance 1000, got %d", balance.Int64())
	}
}

func TestMint_InvalidToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID: "NONEXISTENT",
		To:      recipient,
		Amount:  NewAmount(500),
	}

	_, err := service.Mint(mintReq)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestMint_InvalidRecipient(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         nil,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Error("expected error for nil recipient")
	}
}

func TestMint_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(0),
		PrivateKey: privKey(1),
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

// TestMint_SupplyOverflowRejected proves minting onto an already-saturated
// total_supply fails rather than silently wrapping the ledger into an
// unparseable value (mirrors the SQLite conditional bound).
func TestMint_SupplyOverflowRejected(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, err := service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "OVF",
		TotalSupply: &Amount{Int: new(big.Int).SetUint64(math.MaxInt64)},
		Owner:       owner,
	})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	_, err = service.Mint(&MintRequest{
		TokenID:    "OVF",
		To:         pubKey(2),
		Amount:     NewAmount(1),
		PrivateKey: privKey(1),
	})
	if err == nil {
		t.Fatal("mint onto a saturated supply must be rejected")
	}

	info, err := service.GetTokenInfo("OVF")
	if err != nil {
		t.Fatalf("GetTokenInfo failed: %v", err)
	}
	if got := info.TotalSupply().String(); got != "9223372036854775807" {
		t.Fatalf("total_supply must be unchanged, got %s", got)
	}
}

func TestMint(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err = service.Mint(mintReq)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	balance, err := service.GetBalance("TEST", recipient)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance.Int64() != 500 {
		t.Errorf("expected balance 500, got %d", balance.Int64())
	}
}

func TestTransfer_InvalidToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "NONEXISTENT",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(100),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestTransfer_InvalidFrom(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       nil,
		To:         recipient,
		Amount:     NewAmount(100),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error for nil from")
	}
}

func TestTransfer_InvalidTo(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         nil,
		Amount:     NewAmount(100),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error for nil to")
	}
}

func TestTransfer_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(0),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestTransfer(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	}

	_, err = service.Transfer(transferReq)
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	fromBalance, _ := service.GetBalance("TEST", owner)
	toBalance, _ := service.GetBalance("TEST", recipient)

	if fromBalance.Int64() != 700 {
		t.Errorf("expected sender balance 700, got %d", fromBalance.Int64())
	}
	if toBalance.Int64() != 300 {
		t.Errorf("expected recipient balance 300, got %d", toBalance.Int64())
	}
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(100),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(200),
		PrivateKey: privateKey,
	}

	_, err = service.Transfer(transferReq)
	if err != ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestApprove(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	spender := pubKey(2)
	approveReq := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err = service.Approve(approveReq)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	allowance, err := service.GetAllowance("TEST", owner, spender)
	if err != nil {
		t.Fatalf("GetAllowance failed: %v", err)
	}

	if allowance.Int64() != 500 {
		t.Errorf("expected allowance 500, got %d", allowance.Int64())
	}
}

func TestTransferFrom(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	}

	_, err := service.CreateToken(req)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	spender := pubKey(2)
	spenderKey := privKey(2)
	approveReq := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err = service.Approve(approveReq)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err = service.TransferFrom(transferFromReq)
	if err != nil {
		t.Fatalf("TransferFrom failed: %v", err)
	}

	ownerBalance, _ := service.GetBalance("TEST", owner)
	spenderAllowance, _ := service.GetAllowance("TEST", owner, spender)

	if ownerBalance.Int64() != 800 {
		t.Errorf("expected owner balance 800, got %d", ownerBalance.Int64())
	}
	if spenderAllowance.Int64() != 300 {
		t.Errorf("expected allowance 300, got %d", spenderAllowance.Int64())
	}
}

func TestIncreaseAllowance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})
	if err != nil {
		t.Fatalf("IncreaseAllowance failed: %v", err)
	}

	allowance, _ := service.GetAllowance("TEST", owner, spender)
	if allowance.Int64() != 150 {
		t.Errorf("expected allowance 150, got %d", allowance.Int64())
	}
}

func TestDecreaseAllowance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(30),
		PrivateKey: privKey(1),
	})
	if err != nil {
		t.Fatalf("DecreaseAllowance failed: %v", err)
	}

	allowance, _ := service.GetAllowance("TEST", owner, spender)
	if allowance.Int64() != 70 {
		t.Errorf("expected allowance 70, got %d", allowance.Int64())
	}
}

func TestBurn(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(400),
		PrivateKey: privateKey,
	})
	if err != nil {
		t.Fatalf("Burn failed: %v", err)
	}

	balance, _ := service.GetBalance("TEST", owner)
	if balance.Int64() != 600 {
		t.Errorf("expected balance 600, got %d", balance.Int64())
	}

	// Burn removes tokens from circulation: total_supply must drop
	// by the same amount to keep the invariant total_supply == sum
	// of balances.
	info, err := service.GetTokenInfo("TEST")
	if err != nil {
		t.Fatalf("GetTokenInfo failed: %v", err)
	}
	if info.TotalSupply().Int64() != 600 {
		t.Errorf("expected total supply 600 after burn, got %d", info.TotalSupply().Int64())
	}
}

func TestBurn_InsufficientBalance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(100),
		Owner:       owner,
	})

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(200),
		PrivateKey: privateKey,
	})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestGetTransferHistory(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	privateKey := privKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	for i := 0; i < 3; i++ {
		_, _ = service.Transfer(&TransferRequest{
			TokenID:    "TEST",
			From:       owner,
			To:         recipient,
			Amount:     NewAmount(10),
			PrivateKey: privateKey,
		})
	}

	history, err := service.GetTransferHistory("TEST", owner, 10, 0)
	if err != nil {
		t.Fatalf("GetTransferHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 events, got %d", len(history))
	}
}

type mockRepository struct {
	tokens    map[TokenID]*Token
	balances  map[string]*Amount
	approvals map[string]*Approval

	errGetToken              bool
	saveTokenError           bool
	setAccountBalanceError   bool
	setAccountBalanceToError bool
	saveApprovalError        bool

	// Failure injection for the atomic primitives introduced to
	// close TOCTOU windows. Tests use these to verify the
	// service's compensation/rollback behavior without depending
	// on real concurrency.
	trySubtractBalanceError bool
	tryAddBalanceError      bool
	tryDeductApprovalError  bool

	txBackup    map[string]*Amount
	txTokens    map[TokenID]*Token
	txApprovals map[string]*Approval
}

func NewMockRepository() *mockRepository {
	return &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
		txBackup:  make(map[string]*Amount),
		txTokens:  make(map[TokenID]*Token),
	}
}

func (m *mockRepository) beginTx() {
	m.txBackup = make(map[string]*Amount)
	for k, v := range m.balances {
		m.txBackup[k] = &Amount{new(big.Int).Set(v.Int)}
	}
	m.txTokens = make(map[TokenID]*Token)
	for k, v := range m.tokens {
		origToken := v
		backupToken := &Token{
			id:          origToken.id,
			name:        origToken.name,
			symbol:      origToken.symbol,
			totalSupply: &Amount{new(big.Int).Set(origToken.totalSupply.Int)},
			decimals:    origToken.decimals,
			owner:       origToken.owner,
			isMintable:  origToken.isMintable,
			isBurnable:  origToken.isBurnable,
			createdAt:   origToken.createdAt,
		}
		m.txTokens[k] = backupToken
	}
	m.txApprovals = make(map[string]*Approval)
	for k, v := range m.approvals {
		m.txApprovals[k] = NewApproval(v.tokenID, v.owner, v.spender, &Amount{new(big.Int).Set(v.amount.Int)})
	}
}

func (m *mockRepository) commitTx() {
	m.txBackup = nil
	m.txTokens = nil
}

func (m *mockRepository) rollbackTx() {
	for k, v := range m.txBackup {
		m.balances[k] = v
	}
	for k := range m.balances {
		if _, ok := m.txBackup[k]; !ok {
			delete(m.balances, k)
		}
	}
	for k := range m.tokens {
		if _, ok := m.txTokens[k]; !ok {
			delete(m.tokens, k)
		}
	}
	for k, v := range m.txTokens {
		m.tokens[k] = v
	}
	for k := range m.approvals {
		if _, ok := m.txApprovals[k]; !ok {
			delete(m.approvals, k)
		}
	}
	for k, v := range m.txApprovals {
		m.approvals[k] = v
	}
	m.txBackup = nil
	m.txTokens = nil
	m.txApprovals = nil
}

func (m *mockRepository) SaveToken(token *Token) error {
	if m.saveTokenError {
		return ErrTokenNotFound
	}
	m.tokens[token.ID()] = token
	return nil
}

func (m *mockRepository) GetToken(id TokenID) (*Token, error) {
	if m.errGetToken {
		return nil, ErrTokenNotFound
	}
	return m.tokens[id], nil
}

// WithTx satisfies TransactableRepository. The mock simulates transaction
// semantics through beginTx/commitTx/rollbackTx driven by mockTxManager, so a
// tx-scoped repository is identical to the base repository (tx is always nil
// when the mock tx manager invokes the callback).
func (m *mockRepository) WithTx(_ *sql.Tx) Repository {
	return m
}

func (m *mockRepository) SaveApproval(approval *Approval) error {
	if m.saveApprovalError {
		return fmt.Errorf("approval save error")
	}
	key := string(approval.TokenID()) + string(approval.Owner()) + string(approval.Spender())
	m.approvals[key] = approval
	return nil
}

func (m *mockRepository) GetApproval(tokenID TokenID, owner, spender PublicKey) (*Approval, error) {
	key := string(tokenID) + string(owner) + string(spender)
	return m.approvals[key], nil
}

func (m *mockRepository) GetApprovalsByOwner(tokenID TokenID, owner PublicKey) ([]*Approval, error) {
	var result []*Approval
	for _, approval := range m.approvals {
		if approval.TokenID() == tokenID && bytes.Equal(approval.Owner(), owner) {
			result = append(result, approval)
		}
	}
	return result, nil
}

func (m *mockRepository) GetAccountBalance(tokenID TokenID, owner PublicKey) (*Amount, error) {
	key := string(tokenID) + string(owner)
	if balance, ok := m.balances[key]; ok {
		return balance, nil
	}
	return NewAmount(0), nil
}

func (m *mockRepository) SetAccountBalance(tokenID TokenID, owner PublicKey, amount *Amount) error {
	if m.setAccountBalanceError {
		return fmt.Errorf("balance update error")
	}
	key := string(tokenID) + string(owner)
	if m.setAccountBalanceToError && !bytes.Equal(owner, pubKey(1)) {
		return fmt.Errorf("balance update error")
	}
	m.balances[key] = amount
	return nil
}

// TrySubtractBalance mirrors the SQLite primitive. The mock enforces
// the same ErrInsufficientBalance sentinel so transfer tests can
// assert errors.Is uniformly.
func (m *mockRepository) TrySubtractBalance(tokenID TokenID, owner PublicKey, amount *Amount) (*Amount, error) {
	if m.trySubtractBalanceError || m.setAccountBalanceError {
		return nil, fmt.Errorf("try subtract balance: %w", ErrInsufficientBalance)
	}
	key := string(tokenID) + string(owner)
	cur, ok := m.balances[key]
	if !ok {
		cur = NewAmount(0)
	}
	if cur.Cmp(amount) < 0 {
		return nil, fmt.Errorf("try subtract balance: %w", ErrInsufficientBalance)
	}
	newBal := &Amount{Int: new(big.Int).Sub(cur.Int, amount.Int)}
	m.balances[key] = newBal
	return newBal, nil
}

func (m *mockRepository) TryAddBalance(tokenID TokenID, owner PublicKey, amount *Amount) (*Amount, error) {
	if m.tryAddBalanceError || m.setAccountBalanceError {
		return nil, fmt.Errorf("try add balance: simulated failure")
	}
	key := string(tokenID) + string(owner)
	cur, ok := m.balances[key]
	if !ok {
		cur = NewAmount(0)
	}
	newBal := &Amount{Int: new(big.Int).Add(cur.Int, amount.Int)}
	// Mirror the SQLite primitive: refuse to push the balance past MaxInt64.
	if newBal.BitLen() > 63 {
		return nil, fmt.Errorf("try add balance: balance would exceed maximum")
	}
	m.balances[key] = newBal
	return newBal, nil
}

// TryAddToSupply mirrors the SQLite primitive: atomically adds
// amount to the token's total_supply, refusing to exceed MaxInt64.
func (m *mockRepository) TryAddToSupply(id TokenID, amount *Amount) (*Amount, error) {
	tok, ok := m.tokens[id]
	if !ok {
		return nil, ErrTokenNotFound
	}
	newSupply := &Amount{Int: new(big.Int).Add(tok.TotalSupply().Int, amount.Int)}
	if newSupply.BitLen() > 63 {
		return nil, fmt.Errorf("try add to supply: total supply would exceed maximum")
	}
	m.tokens[id] = NewToken(id, tok.Name(), tok.Symbol(), newSupply, tok.Owner())
	return newSupply, nil
}

// TrySubtractFromSupply mirrors the SQLite primitive: atomically subtracts
// amount from the token's total_supply (Burn).
func (m *mockRepository) TrySubtractFromSupply(id TokenID, amount *Amount) (*Amount, error) {
	tok, ok := m.tokens[id]
	if !ok {
		return nil, ErrTokenNotFound
	}
	if tok.TotalSupply().Cmp(amount) < 0 {
		return nil, fmt.Errorf("try subtract supply: total supply below burn amount")
	}
	newSupply := &Amount{Int: new(big.Int).Sub(tok.TotalSupply().Int, amount.Int)}
	m.tokens[id] = NewToken(id, tok.Name(), tok.Symbol(), newSupply, tok.Owner())
	return newSupply, nil
}

func (m *mockRepository) TryDeductApproval(tokenID TokenID, owner, spender PublicKey, amount *Amount) (*Amount, error) {
	if m.tryDeductApprovalError {
		return nil, fmt.Errorf("try deduct approval: %w", ErrInsufficientAllowance)
	}
	key := string(tokenID) + string(owner) + string(spender)
	cur, ok := m.approvals[key]
	if !ok {
		return nil, fmt.Errorf("try deduct approval: %w", ErrInsufficientAllowance)
	}
	if cur.Amount().Cmp(amount) < 0 {
		return nil, fmt.Errorf("try deduct approval: %w", ErrInsufficientAllowance)
	}
	newAmt := &Amount{Int: new(big.Int).Sub(cur.Amount().Int, amount.Int)}
	m.approvals[key] = NewApproval(tokenID, owner, spender, newAmt)
	return newAmt, nil
}

// TryAdjustApproval mirrors the SQLite primitive: applies a signed
// delta to the approval, creating it if missing, clamping at zero.
func (m *mockRepository) TryAdjustApproval(tokenID TokenID, owner, spender PublicKey, delta *Amount) (*Amount, error) {
	key := string(tokenID) + string(owner) + string(spender)
	cur, ok := m.approvals[key]
	var curAmount *Amount
	if ok {
		curAmount = cur.Amount()
	} else {
		curAmount = NewAmount(0)
	}
	newAmt := &Amount{Int: new(big.Int).Add(curAmount.Int, delta.Int)}
	if newAmt.Sign() < 0 {
		newAmt = NewAmount(0)
	}
	// Mirror the SQLite primitive's ceiling clamp at MaxInt64.
	if newAmt.BitLen() > 63 {
		newAmt = &Amount{Int: new(big.Int).Add(big.NewInt(0), big.NewInt(math.MaxInt64))}
	}
	m.approvals[key] = NewApproval(tokenID, owner, spender, newAmt)
	return newAmt, nil
}

type mockEventStore struct {
	transferEvents []*TransferEvent
	mintEvents     []*MintEvent
	burnEvents     []*BurnEvent
	approveEvents  []*ApproveEvent
}

func NewMockEventStore() *mockEventStore {
	return &mockEventStore{
		transferEvents: make([]*TransferEvent, 0),
		mintEvents:     make([]*MintEvent, 0),
		burnEvents:     make([]*BurnEvent, 0),
		approveEvents:  make([]*ApproveEvent, 0),
	}
}

func (m *mockEventStore) GetTransferEventsByOwner(tokenID TokenID, owner PublicKey, limit, offset int) ([]*TransferEvent, error) {
	var result []*TransferEvent
	for _, e := range m.transferEvents {
		if e.TokenID() == tokenID && (bytes.Equal(e.From(), owner) || bytes.Equal(e.To(), owner)) {
			result = append(result, e)
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if offset >= len(result) {
		return []*TransferEvent{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockEventStore) GetMintEventsByToken(tokenID TokenID) ([]*MintEvent, error) {
	var result []*MintEvent
	for _, e := range m.mintEvents {
		if e.TokenID() == tokenID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEventStore) GetBurnEventsByToken(tokenID TokenID) ([]*BurnEvent, error) {
	var result []*BurnEvent
	for _, e := range m.burnEvents {
		if e.TokenID() == tokenID {
			result = append(result, e)
		}
	}
	return result, nil
}

type mockEventBus struct {
	eventStore *mockEventStore
}

func newMockEventBus(es *mockEventStore) *mockEventBus {
	return &mockEventBus{eventStore: es}
}

func (m *mockEventBus) Publish(e events.Event) error {
	switch evt := e.(type) {
	case *TransferEvent:
		m.eventStore.transferEvents = append(m.eventStore.transferEvents, evt)
	case *MintEvent:
		m.eventStore.mintEvents = append(m.eventStore.mintEvents, evt)
	case *BurnEvent:
		m.eventStore.burnEvents = append(m.eventStore.burnEvents, evt)
	case *ApproveEvent:
		m.eventStore.approveEvents = append(m.eventStore.approveEvents, evt)
	}
	return nil
}

func (m *mockEventBus) Subscribe(eventType string, handler infraevents.Handler) func() {
	return func() {}
}

func (m *mockEventBus) SubscribeAll(handler infraevents.Handler) func() {
	return func() {}
}

type mockReplayProtection struct {
	nonces map[string]uint64
}

func newMockReplayProtection() *mockReplayProtection {
	return &mockReplayProtection{
		nonces: make(map[string]uint64),
	}
}

func (m *mockReplayProtection) GetLastNonce(tokenID string, owner []byte) (uint64, error) {
	key := tokenID + string(owner)
	return m.nonces[key], nil
}

func (m *mockReplayProtection) SaveNonce(tokenID string, owner []byte, nonce uint64) error {
	key := tokenID + string(owner)
	m.nonces[key] = nonce
	return nil
}

func (m *mockReplayProtection) ClaimNextNonce(tokenID string, owner []byte) (uint64, error) {
	key := tokenID + string(owner)
	m.nonces[key]++
	return m.nonces[key], nil
}

type mockBlockWriter struct {
	height int64
}

func (m *mockBlockWriter) AddBlock(data string) (int64, error) {
	m.height++
	return m.height, nil
}

func newTestService(repo TransactableRepository, eventStore *mockEventStore) *TokenService {
	return NewService(repo, newMockTxManager(), newMockEventBus(eventStore), eventStore, newMockReplayProtection(), &mockBlockWriter{})
}

func TestGetAllowance_NoApproval(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	allowance, err := service.GetAllowance("TEST", owner, spender)
	if err != nil {
		t.Fatalf("GetAllowance failed: %v", err)
	}

	if allowance.Int64() != 0 {
		t.Errorf("expected allowance 0, got %d", allowance.Int64())
	}
}

func TestMint_NonMintableToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	token := &Token{
		id:          "FIXED",
		name:        "Fixed Token",
		symbol:      "FIXED",
		totalSupply: NewAmount(1000),
		owner:       owner,
		isMintable:  false,
		isBurnable:  true,
	}
	repo.tokens["FIXED"] = token

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "FIXED",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err := service.Mint(mintReq)
	if err != ErrTokenNotMintable {
		t.Errorf("expected ErrTokenNotMintable, got %v", err)
	}
}

func TestMint_InvalidTokenRepoError(t *testing.T) {
	repo := &mockRepository{
		tokens:      make(map[TokenID]*Token),
		balances:    make(map[string]*Amount),
		approvals:   make(map[string]*Approval),
		errGetToken: true,
	}
	eventStore := NewMockEventStore()
	service := newTestServiceWithRepo(repo, eventStore)

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Error("expected error for repo failure")
	}
}

func TestTransferFrom_NilApproval(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err != ErrInsufficientAllowance {
		t.Errorf("expected ErrInsufficientAllowance for nil approval, got %v", err)
	}
}

func TestTransferFrom_InsufficientAllowance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})

	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err != ErrInsufficientAllowance {
		t.Errorf("expected ErrInsufficientAllowance, got %v", err)
	}
}

func TestTransferFrom_OwnerInsufficientBalance(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(100),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	})

	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err != ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestTransferFrom_InvalidOwner(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      nil,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err == nil {
		t.Error("expected error for nil owner")
	}
}

func TestTransferFrom_InvalidSpender(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    nil,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err == nil {
		t.Error("expected error for nil spender")
	}
}

func TestTransferFrom_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(0),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestTransferFrom_InvalidToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	spender := pubKey(2)
	spenderKey := privKey(2)
	recipient := pubKey(3)
	transferFromReq := &TransferFromRequest{
		TokenID:    "NONEXISTENT",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	}

	_, err := service.TransferFrom(transferFromReq)
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestTransfer_Atomicity_FromBalanceUpdateFails(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	recipient := pubKey(2)
	privateKey := privKey(1)
	repo.trySubtractBalanceError = true
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error when from balance update fails")
	}
}

func TestTransfer_Atomicity_ToBalanceUpdateFails(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	recipient := pubKey(2)
	privateKey := privKey(1)
	repo.tryAddBalanceError = true
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error when to balance update fails")
	}
}

func TestApprove_InvalidToken(t *testing.T) {
	repo := &mockRepository{
		tokens:      make(map[TokenID]*Token),
		balances:    make(map[string]*Amount),
		approvals:   make(map[string]*Approval),
		errGetToken: true,
	}
	eventStore := NewMockEventStore()
	service := newTestServiceWithRepo(repo, eventStore)

	owner := pubKey(1)
	approveReq := &ApproveRequest{
		TokenID: "NONEXISTENT",
		Owner:   owner,
		Spender: pubKey(2),
		Amount:  NewAmount(500),
	}

	_, err := service.Approve(approveReq)
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func TestApprove_InvalidOwner(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	approveReq := &ApproveRequest{
		TokenID: "TEST",
		Owner:   nil,
		Spender: pubKey(2),
		Amount:  NewAmount(500),
	}

	_, err := service.Approve(approveReq)
	if err == nil {
		t.Error("expected error for nil owner")
	}
}

func TestApprove_InvalidSpender(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	approveReq := &ApproveRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: nil,
		Amount:  NewAmount(500),
	}

	_, err := service.Approve(approveReq)
	if err == nil {
		t.Error("expected error for nil spender")
	}
}

func TestApprove_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	approveReq := &ApproveRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: pubKey(2),
		Amount:  NewAmount(0),
	}

	_, err := service.Approve(approveReq)
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestApprove_UpdatesExistingApproval(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})

	_, err := service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(200),
		PrivateKey: privKey(1),
	})
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	allowance, _ := service.GetAllowance("TEST", owner, spender)
	if allowance.Int64() != 200 {
		t.Errorf("expected allowance 200, got %d", allowance.Int64())
	}
}

func TestBurn_NonBurnableToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	token := &Token{
		id:          "FIXED",
		name:        "Fixed Token",
		symbol:      "FIXED",
		totalSupply: NewAmount(1000),
		owner:       owner,
		isMintable:  true,
		isBurnable:  false,
	}
	repo.tokens["FIXED"] = token

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "FIXED",
		From:       owner,
		Amount:     NewAmount(400),
		PrivateKey: privateKey,
	})
	if err != ErrTokenNotBurnable {
		t.Errorf("expected ErrTokenNotBurnable, got %v", err)
	}
}

func TestBurn_InvalidFrom(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       nil,
		Amount:     NewAmount(400),
		PrivateKey: privateKey,
	})
	if err == nil {
		t.Error("expected error for nil from")
	}
}

func TestBurn_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(0),
		PrivateKey: privateKey,
	})
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestBurn_InvalidToken(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "NONEXISTENT",
		From:       pubKey(1),
		Amount:     NewAmount(400),
		PrivateKey: privateKey,
	})
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestGetTransferHistory_DefaultLimit(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	events, err := service.GetTransferHistory("TEST", owner, 0, 0)
	if err != nil {
		t.Fatalf("GetTransferHistory failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestGetTransferHistory_WithPagination(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	privateKey := privKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	for i := 0; i < 10; i++ {
		_, _ = service.Transfer(&TransferRequest{
			TokenID:    "TEST",
			From:       owner,
			To:         recipient,
			Amount:     NewAmount(10),
			PrivateKey: privateKey,
		})
	}

	history, err := service.GetTransferHistory("TEST", owner, 5, 0)
	if err != nil {
		t.Fatalf("GetTransferHistory failed: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("expected 5 events, got %d", len(history))
	}
}

func TestGetTransferHistory_WithOffset(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	privateKey := privKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	for i := 0; i < 10; i++ {
		_, _ = service.Transfer(&TransferRequest{
			TokenID:    "TEST",
			From:       owner,
			To:         recipient,
			Amount:     NewAmount(10),
			PrivateKey: privateKey,
		})
	}

	history, err := service.GetTransferHistory("TEST", owner, 5, 5)
	if err != nil {
		t.Fatalf("GetTransferHistory failed: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("expected 5 events, got %d", len(history))
	}
}

func TestIncreaseAllowance_NoExistingApproval(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})
	if err != nil {
		t.Fatalf("IncreaseAllowance failed: %v", err)
	}

	allowance, _ := service.GetAllowance("TEST", owner, spender)
	if allowance.Int64() != 100 {
		t.Errorf("expected allowance 100, got %d", allowance.Int64())
	}
}

func TestDecreaseAllowance_NoExistingApproval(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(10),
		PrivateKey: privKey(1),
	})

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(5),
		PrivateKey: privKey(1),
	})
	if err != nil {
		t.Fatalf("DecreaseAllowance failed: %v", err)
	}

	allowance, _ := service.GetAllowance("TEST", owner, spender)
	if allowance.Int64() != 5 {
		t.Errorf("expected allowance 5, got %d", allowance.Int64())
	}
}

func TestDecreaseAllowance_ClampsToZero(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})

	_, err := service.Approve(&ApproveRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(0),
	})
	if err == nil {
		t.Error("expected error when setting allowance to 0")
	}
}

func TestCreateToken_RepoError(t *testing.T) {
	repo := &mockRepository{
		tokens:         make(map[TokenID]*Token),
		balances:       make(map[string]*Amount),
		approvals:      make(map[string]*Approval),
		saveTokenError: true,
	}
	eventStore := NewMockEventStore()
	service := newTestServiceWithRepo(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       pubKey(1),
	}

	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error when save fails")
	}
}

func TestCreateToken_SetBalanceError(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	service := newTestServiceWithRepo(repo, eventStore)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       pubKey(1),
	}

	repo.setAccountBalanceError = true
	_, err := service.CreateToken(req)
	if err == nil {
		t.Error("expected error when set balance fails")
	}
}

func TestTransferFrom_SetBalanceError(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	spender := pubKey(2)
	repo.approvals[string(token.ID())+string(owner)+string(spender)] = NewApproval("TEST", owner, spender, NewAmount(500))

	spenderKey := privKey(2)
	recipient := pubKey(3)

	repo.tryAddBalanceError = true
	_, err := service.TransferFrom(&TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	})
	if err == nil {
		t.Error("expected error when to balance update fails")
	}
}

func TestTransferFrom_AllowanceDeductionFailure(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	spender := pubKey(2)
	repo.approvals[string(token.ID())+string(owner)+string(spender)] = NewApproval("TEST", owner, spender, NewAmount(500))

	spenderKey := privKey(2)
	recipient := pubKey(3)

	// The allowance is deducted through the atomic TryDeductApproval
	// primitive; saveApprovalError (which previously exercised a redundant
	// SaveApproval) no longer applies. Inject a deduction failure instead.
	repo.tryDeductApprovalError = true
	_, err := service.TransferFrom(&TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	})
	if err == nil {
		t.Error("expected error when update approval fails")
	}
	// Because the deduction is atomic, a rejection leaves no partial state:
	// the allowance, balances, and approval row are all untouched.
	approval, _ := repo.GetApproval("TEST", owner, spender)
	if approval == nil || approval.Amount().Int64() != 500 {
		t.Errorf("allowance should be unchanged (500), got %+v", approval)
	}
	if got, _ := repo.GetAccountBalance("TEST", owner); got.Int64() != 1000 {
		t.Errorf("owner balance should be unchanged (1000), got %d", got.Int64())
	}
}

// TestTransferFrom_AtomicityRollbackOnCreditFailure proves the mid-transaction
// atomicity claim: the allowance deduction and the owner debit succeed, the
// recipient credit then fails, and the entire transfer rolls back — no double
// deduction, no partial debit, no orphan allowance change.
func TestTransferFrom_AtomicityRollbackOnCreditFailure(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	// Bound the tx manager to the repo so a failing step triggers a true
	// rollback (beginTx snapshots, rollbackTx restores).
	service := NewService(repo, newMockTxManagerWithRepo(repo), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	spender := pubKey(2)
	repo.approvals[string(token.ID())+string(owner)+string(spender)] = NewApproval("TEST", owner, spender, NewAmount(500))

	spenderKey := privKey(2)
	recipient := pubKey(3)

	// The recipient credit is the last mutation in the transaction; making it
	// fail forces a rollback of the allowance deduction and owner debit that
	// already succeeded on the in-tx state.
	repo.tryAddBalanceError = true
	_, err := service.TransferFrom(&TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	})
	if err == nil {
		t.Fatal("expected error when recipient credit fails")
	}

	approval, _ := repo.GetApproval("TEST", owner, spender)
	if approval == nil || approval.Amount().Int64() != 500 {
		t.Errorf("allowance must be rolled back to 500, got %+v", approval)
	}
	if got, _ := repo.GetAccountBalance("TEST", owner); got.Int64() != 1000 {
		t.Errorf("owner balance must be rolled back to 1000, got %d", got.Int64())
	}
	if got, _ := repo.GetAccountBalance("TEST", recipient); got.Int64() != 0 {
		t.Errorf("recipient must have no balance after rollback, got %d", got.Int64())
	}
}

func newTestServiceWithRepo(repo *mockRepository, eventStore *mockEventStore) *TokenService {
	return NewService(repo, newMockTxManager(), newMockEventBus(eventStore), eventStore, newMockReplayProtection(), &mockBlockWriter{})
}

// TestTransfer_PublishFailureDoesNotRollbackCommittedTransfer encodes the
// post-ISS-074 contract: the audit event is published AFTER the token
// transaction commits (the event bus writes to a separate events store that a
// token DB rollback cannot undo — and must not abort a value transfer). A
// failing event publish therefore surfaces as an error from the service but
// leaves the already-committed transfer in place. The previous behavior
// (publish INSIDE the tx) inverted this: the publish could be rolled back
// together with the token state only because both sat in one in-memory mock,
// which is impossible across the real tokens.db / events.db split.
func TestTransfer_PublishFailureDoesNotRollbackCommittedTransfer(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	txManager := newMockTxManager()
	eventBus := &failingEventBus{err: fmt.Errorf("publish failed")}
	service := NewService(repo, txManager, eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error when event publish fails")
	}

	// The transfer itself is committed: publish runs after the commit, so an
	// event-store outage loses the audit event, never the transfer.
	fromBalance := repo.balances[string(token.ID())+string(owner)]
	if fromBalance.Int64() != 700 {
		t.Errorf("from balance should be committed (700), got %d", fromBalance.Int64())
	}
}

func TestTransferFrom_AtomicityRollbackOnToBalanceFailure(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
		txBackup:  make(map[string]*Amount),
		txTokens:  make(map[TokenID]*Token),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	eventBus := newMockEventBus(eventStore)
	txManager := newMockTxManagerWithRepo(repo)
	service := NewService(repo, txManager, eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	spender := pubKey(2)
	repo.approvals[string(token.ID())+string(owner)+string(spender)] = NewApproval("TEST", owner, spender, NewAmount(500))

	recipient := pubKey(3)
	spenderKey := privKey(2)

	repo.trySubtractBalanceError = true
	_, err := service.TransferFrom(&TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(200),
		Spender:    spender,
		SpenderKey: spenderKey,
	})
	if err == nil {
		t.Error("expected error when to balance update fails")
	}

	ownerBalance := repo.balances[string(token.ID())+string(owner)]
	if ownerBalance.Int64() != 1000 {
		t.Errorf("owner balance should be unchanged (1000), got %d", ownerBalance.Int64())
	}

	originalAllowance := repo.approvals[string(token.ID())+string(owner)+string(spender)]
	if originalAllowance.Amount().Int64() != 500 {
		t.Errorf("allowance should be unchanged (500), got %d", originalAllowance.Amount().Int64())
	}
}

func TestMint_AtomicityRollbackOnBalanceUpdateFailure(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
		txBackup:  make(map[string]*Amount),
		txTokens:  make(map[TokenID]*Token),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	eventBus := newMockEventBus(eventStore)
	txManager := newMockTxManagerWithRepo(repo)
	service := NewService(repo, txManager, eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	recipient := pubKey(2)
	repo.setAccountBalanceError = true
	_, err := service.Mint(&MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	})
	if err == nil {
		t.Error("expected error when balance update fails")
	}

	savedToken := repo.tokens["TEST"]
	if savedToken.TotalSupply().Int64() != 1000 {
		t.Errorf("total supply should be unchanged (1000), got %d", savedToken.TotalSupply().Int64())
	}
}

func TestBurn_AtomicityRollbackOnBalanceUpdateFailure(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	eventBus := newMockEventBus(eventStore)
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	privateKey := privKey(1)
	repo.setAccountBalanceError = true
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(400),
		PrivateKey: privateKey,
	})
	if err == nil {
		t.Error("expected error when balance update fails")
	}

	balance := repo.balances[string(token.ID())+string(owner)]
	if balance.Int64() != 1000 {
		t.Errorf("balance should be unchanged (1000), got %d", balance.Int64())
	}
}

func TestBurn_SupplyBelowBurnAmountRollsBackBalance(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	eventBus := newMockEventBus(eventStore)
	// The manager is bound to the repo so a failing step triggers a true
	// rollback of the whole transaction (beginTx snapshots, rollbackTx
	// restores), exactly like the SQLite path.
	service := NewService(repo, newMockTxManagerWithRepo(repo), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	// Deliberately break the ledger invariant: the owner holds more than the
	// recorded total_supply, so a burn large enough for the balance debit to
	// succeed still trips the supply decrement (total_supply < amount).
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(100), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	privateKey := privKey(1)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(200),
		PrivateKey: privateKey,
	})
	if err == nil {
		t.Fatal("expected error when total_supply is below the burn amount")
	}

	// The balance debit ran inside the same transaction as the failed supply
	// decrement, so it must roll back — no partial burn.
	balance := repo.balances[string(token.ID())+string(owner)]
	if balance.Int64() != 1000 {
		t.Errorf("balance should be rolled back (1000), got %d", balance.Int64())
	}
	if tok, _ := repo.GetToken(token.ID()); tok.TotalSupply().Int64() != 100 {
		t.Errorf("total supply should be unchanged (100), got %d", tok.TotalSupply().Int64())
	}
}

func TestTransfer_AtomicityTransactionFailureDoesNotCorruptState(t *testing.T) {
	repo := &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	replay := newMockReplayProtection()
	eventBus := newMockEventBus(eventStore)
	txManager := newFailingTxManager(1)
	service := NewService(repo, txManager, eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	token := NewToken("TEST", "Test Token", "TEST", NewAmount(1000), owner)
	repo.tokens[token.ID()] = token
	repo.balances[string(token.ID())+string(owner)] = NewAmount(1000)

	recipient := pubKey(2)
	privateKey := privKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Error("expected error when transaction fails")
	}

	fromBalance := repo.balances[string(token.ID())+string(owner)]
	if fromBalance.Int64() != 1000 {
		t.Errorf("from balance should be 1000, got %d", fromBalance.Int64())
	}

	toBalance := repo.balances[string(token.ID())+string(recipient)]
	if toBalance != nil {
		t.Errorf("to balance should be nil (not created), got %v", toBalance)
	}
}

// TestTxRollback_DoesNotPublishPhantomEvent is the regression test for
// ISS-074: Mint/Transfer/TransferFrom/Burn used to publish their audit event
// INSIDE the TxManager transaction. The event bus writes to a separate events
// store (events.db) that the token DB rollback cannot undo, so when a later
// step in the same transaction failed (e.g. the atomic balance primitive
// returns ErrInsufficientBalance under a concurrent double-spend) the event
// was already persisted — leaving a phantom event in GetTransferHistory even
// though the transfer never happened.
//
// After the fix the event is published only AFTER WithTransaction commits;
// a rolled-back transaction therefore publishes nothing.
//
// Pre-fix each case below ends with eventStore.{type}Events non-empty; the
// fix makes them empty. A positive control (successful transfer) still
// persists exactly one event, so the audit trail is not lost on the happy path.
func TestTxRollback_DoesNotPublishPhantomEvent(t *testing.T) {
	newService := func() (*mockRepository, *mockEventStore, *TokenService, *mockTxManager) {
		repo := &mockRepository{
			tokens:    make(map[TokenID]*Token),
			balances:  make(map[string]*Amount),
			approvals: make(map[string]*Approval),
		}
		eventStore := NewMockEventStore()
		chain := blockchain.NewBlockChain()
		replay := newMockReplayProtection()
		eventBus := newMockEventBus(eventStore)
		txManager := newMockTxManagerWithRepo(repo)
		service := NewService(repo, txManager, eventBus, eventStore, replay, chain)
		return repo, eventStore, service, txManager
	}

	owner := pubKey(1)
	privateKey := privKey(1)
	recipient := pubKey(2)

	t.Run("mint rolls back without publishing a mint event", func(t *testing.T) {
		repo, eventStore, service, _ := newService()
		if _, err := service.CreateToken(&CreateTokenRequest{
			Name: "Test Token", Symbol: "TEST", TotalSupply: NewAmount(1000), Owner: owner,
		}); err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		// Publish happens (pre-fix) before TryAddBalance; fail the balance
		// write so the transaction rolls back after publish already ran.
		repo.tryAddBalanceError = true
		_, err := service.Mint(&MintRequest{TokenID: "TEST", To: recipient, Amount: NewAmount(100), PrivateKey: privateKey})
		if err == nil {
			t.Fatal("mint must fail when a tx step fails")
		}
		if len(eventStore.mintEvents) != 0 {
			t.Errorf("no phantom mint event after rollback, got %d", len(eventStore.mintEvents))
		}
	})

	t.Run("transfer rolls back without publishing a transfer event", func(t *testing.T) {
		repo, eventStore, service, _ := newService()
		if _, err := service.CreateToken(&CreateTokenRequest{
			Name: "Test Token", Symbol: "TEST", TotalSupply: NewAmount(1000), Owner: owner,
		}); err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		repo.balances[string("TEST")+string(owner)] = NewAmount(1000)

		// Publish happens (pre-fix) before TrySubtractBalance; fail the debit
		// so the transaction rolls back after publish already ran.
		repo.trySubtractBalanceError = true
		_, err := service.Transfer(&TransferRequest{TokenID: "TEST", From: owner, To: recipient, Amount: NewAmount(100), PrivateKey: privateKey})
		if err == nil {
			t.Fatal("transfer must fail when a tx step fails")
		}
		if len(eventStore.transferEvents) != 0 {
			t.Errorf("no phantom transfer event after rollback, got %d", len(eventStore.transferEvents))
		}
	})

	t.Run("transferfrom rolls back without publishing a transfer event", func(t *testing.T) {
		repo, eventStore, service, _ := newService()
		if _, err := service.CreateToken(&CreateTokenRequest{
			Name: "Test Token", Symbol: "TEST", TotalSupply: NewAmount(1000), Owner: owner,
		}); err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		repo.balances[string("TEST")+string(owner)] = NewAmount(1000)
		repo.approvals[string("TEST")+string(owner)+string(recipient)] = NewApproval("TEST", owner, recipient, NewAmount(500))

		// Publish happens (pre-fix) before TryDeductApproval; fail the
		// allowance deduction so the transaction rolls back after publish.
		repo.tryDeductApprovalError = true
		_, err := service.TransferFrom(&TransferFromRequest{
			TokenID: "TEST", Owner: owner, To: pubKey(3), Amount: NewAmount(100),
			Spender: recipient, SpenderKey: privKey(2),
		})
		if err == nil {
			t.Fatal("transferfrom must fail when a tx step fails")
		}
		if len(eventStore.transferEvents) != 0 {
			t.Errorf("no phantom transferfrom event after rollback, got %d", len(eventStore.transferEvents))
		}
	})

	t.Run("burn rolls back without publishing a burn event", func(t *testing.T) {
		repo, eventStore, service, _ := newService()
		if _, err := service.CreateToken(&CreateTokenRequest{
			Name: "Test Token", Symbol: "TEST", TotalSupply: NewAmount(1000), Owner: owner,
		}); err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		repo.balances[string("TEST")+string(owner)] = NewAmount(1000)

		// Publish happens (pre-fix) before TrySubtractBalance; fail the debit
		// so the transaction rolls back after publish already ran.
		repo.trySubtractBalanceError = true
		_, err := service.Burn(&BurnRequest{TokenID: "TEST", From: owner, Amount: NewAmount(100), PrivateKey: privateKey})
		if err == nil {
			t.Fatal("burn must fail when a tx step fails")
		}
		if len(eventStore.burnEvents) != 0 {
			t.Errorf("no phantom burn event after rollback, got %d", len(eventStore.burnEvents))
		}
	})

	t.Run("successful transfer still publishes exactly one event", func(t *testing.T) {
		// Positive control: the audit trail is preserved on the happy path.
		repo, eventStore, service, _ := newService()
		if _, err := service.CreateToken(&CreateTokenRequest{
			Name: "Test Token", Symbol: "TEST", TotalSupply: NewAmount(1000), Owner: owner,
		}); err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		repo.balances[string("TEST")+string(owner)] = NewAmount(1000)

		if _, err := service.Transfer(&TransferRequest{TokenID: "TEST", From: owner, To: recipient, Amount: NewAmount(100), PrivateKey: privateKey}); err != nil {
			t.Fatalf("Transfer failed: %v", err)
		}
		if len(eventStore.transferEvents) != 1 {
			t.Errorf("successful transfer persists exactly one event, got %d", len(eventStore.transferEvents))
		}
	})
}

type failingEventBus struct {
	err error
}

func (f *failingEventBus) Publish(e events.Event) error {
	return f.err
}

func (f *failingEventBus) Subscribe(eventType string, handler infraevents.Handler) func() {
	return func() {}
}

func (f *failingEventBus) SubscribeAll(handler infraevents.Handler) func() {
	return func() {}
}

// TestNoOpTxManager_ExecutesCallback is a regression test for the bug
// where noOpTxManager.WithTransaction returned nil without calling fn.
//
// Before the fix, NewServiceWithoutTx produced a service whose Mint,
// Transfer, TransferFrom, and Burn methods silently skipped all
// database operations (TryAddToSupply, TryAddBalance, TrySubtractBalance,
// etc. never ran). A mint would appear to succeed but no supply or
// balance would change.
//
// This test verifies that noOpTxManager actually invokes the callback
// by minting tokens through a service built with NewServiceWithoutTx
// and checking that the recipient's balance is non-zero.
func TestNoOpTxManager_ExecutesCallback(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	service := NewServiceWithoutTx(repo, newMockEventBus(eventStore), eventStore, newMockReplayProtection(), chain)

	owner := pubKey(1)
	_, err := service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	}
	_, err = service.Mint(mintReq)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	// If noOpTxManager called fn(nil), TryAddToSupply and TryAddBalance
	// have run. If it returned nil without calling fn, this balance is 0.
	balance, err := service.GetBalance("TEST", recipient)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	if balance == nil {
		t.Fatal("balance should not be nil after mint")
	}
	if balance.Int64() != 500 {
		t.Errorf("expected recipient balance 500 after mint, got %d — "+
			"noOpTxManager.WithTransaction likely did not call fn", balance.Int64())
	}
}

// TestNoOpTxManager_TransferWorksThroughNoOpTx is a second regression
// check: Transfer also relies on WithTransaction calling fn. We verify
// that balances actually move through a service built with
// NewServiceWithoutTx.
func TestNoOpTxManager_TransferWorksThroughNoOpTx(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	service := NewServiceWithoutTx(repo, newMockEventBus(eventStore), eventStore, newMockReplayProtection(), chain)

	owner := pubKey(1)
	recipient := pubKey(2)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	privateKey := privKey(1)
	_, err := service.Transfer(&TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(300),
		PrivateKey: privateKey,
	})
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	fromBal, _ := service.GetBalance("TEST", owner)
	toBal, _ := service.GetBalance("TEST", recipient)

	if fromBal.Int64() != 700 {
		t.Errorf("expected from balance 700, got %d", fromBal.Int64())
	}
	if toBal.Int64() != 300 {
		t.Errorf("expected to balance 300, got %d", toBal.Int64())
	}
}

func TestIncreaseAllowance_InvalidOwner(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   PublicKey{},
		Spender: pubKey(2),
		Amount:  NewAmount(50),
	})
	if err == nil {
		t.Fatal("Expected error for invalid owner public key")
	}
}

func TestIncreaseAllowance_InvalidSpender(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   pubKey(1),
		Spender: PublicKey{},
		Amount:  NewAmount(50),
	})
	if err == nil {
		t.Fatal("Expected error for invalid spender public key")
	}
}

func TestIncreaseAllowance_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   pubKey(1),
		Spender: pubKey(2),
		Amount:  NewAmount(-50),
	})
	if err == nil {
		t.Fatal("Expected error for negative amount")
	}
}

func TestIncreaseAllowance_PublishError(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, newMockTxManager(), &failingEventBus{err: errors.New("publish failed")}, NewMockEventStore(), newMockReplayProtection(), &mockBlockWriter{})

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})
	if err == nil {
		t.Fatal("Expected error for publish failure")
	}
}

func TestDecreaseAllowance_InvalidOwner(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   PublicKey{},
		Spender: pubKey(2),
		Amount:  NewAmount(50),
	})
	if err == nil {
		t.Fatal("Expected error for invalid owner public key")
	}
}

func TestDecreaseAllowance_InvalidSpender(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   pubKey(1),
		Spender: PublicKey{},
		Amount:  NewAmount(50),
	})
	if err == nil {
		t.Fatal("Expected error for invalid spender public key")
	}
}

func TestDecreaseAllowance_InvalidAmount(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   pubKey(1),
		Spender: pubKey(2),
		Amount:  NewAmount(-50),
	})
	if err == nil {
		t.Fatal("Expected error for negative amount")
	}
}

func TestDecreaseAllowance_PublishError(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, newMockTxManager(), &failingEventBus{err: errors.New("publish failed")}, NewMockEventStore(), newMockReplayProtection(), &mockBlockWriter{})

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(100),
		PrivateKey: privKey(1),
	})

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})
	if err == nil {
		t.Fatal("Expected error for publish failure")
	}
}

// ============================================================
// Authorization tests — verify PrivateKey matches expected
// PublicKey for each mutating operation.
// ============================================================

func TestMint_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	// Use privKey(2) — does NOT match owner pubKey(1)
	recipient := pubKey(3)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: privKey(2),
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in Mint")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestMint_NilPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	mintReq := &MintRequest{
		TokenID:    "TEST",
		To:         recipient,
		Amount:     NewAmount(500),
		PrivateKey: nil,
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Fatal("expected error for nil private key in Mint")
	}
}

func TestBurn_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	// Use privKey(2) — does NOT match From pubKey(1)
	burnReq := &BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(100),
		PrivateKey: privKey(2),
	}

	_, err := service.Burn(burnReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in Burn")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestTransfer_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	recipient := pubKey(2)
	// Use privKey(3) — does NOT match From pubKey(1)
	transferReq := &TransferRequest{
		TokenID:    "TEST",
		From:       owner,
		To:         recipient,
		Amount:     NewAmount(100),
		PrivateKey: privKey(3),
	}

	_, err := service.Transfer(transferReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in Transfer")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestTransferFrom_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	// Set up an approval
	_, _ = service.Approve(&ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(500),
		PrivateKey: privKey(1),
	})

	recipient := pubKey(3)
	// Use privKey(4) — does NOT match Spender pubKey(2)
	transferFromReq := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      owner,
		To:         recipient,
		Amount:     NewAmount(100),
		Spender:    spender,
		SpenderKey: privKey(4),
	}

	_, err := service.TransferFrom(transferFromReq)
	if err == nil {
		t.Fatal("expected error for unauthorized spender key in TransferFrom")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestApprove_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	// Use privKey(3) — does NOT match Owner pubKey(1)
	approveReq := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(500),
		PrivateKey: privKey(3),
	}

	_, err := service.Approve(approveReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in Approve")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestIncreaseAllowance_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	// Use privKey(3) — does NOT match Owner pubKey(1)
	allowanceReq := &AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(3),
	}

	_, err := service.IncreaseAllowance(allowanceReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in IncreaseAllowance")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDecreaseAllowance_UnauthorizedPrivateKey(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	chain := blockchain.NewBlockChain()
	eventBus := newMockEventBus(eventStore)
	replay := newMockReplayProtection()
	service := NewService(repo, newMockTxManager(), eventBus, eventStore, replay, chain)

	owner := pubKey(1)
	_, _ = service.CreateToken(&CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: NewAmount(1000),
		Owner:       owner,
	})

	spender := pubKey(2)
	// Use privKey(3) — does NOT match Owner pubKey(1)
	allowanceReq := &AllowanceRequest{
		TokenID:    "TEST",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(30),
		PrivateKey: privKey(3),
	}

	_, err := service.DecreaseAllowance(allowanceReq)
	if err == nil {
		t.Fatal("expected error for unauthorized private key in DecreaseAllowance")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// TestIncreaseAllowance_TokenNotFound guards the dangling-token gap: adjusting
// allowance for a token that does not exist must be rejected, not silently
// create an orphaned allowance row.
func TestIncreaseAllowance_TokenNotFound(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	spender := pubKey(2)
	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID:    "NOPE",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
	if _, exists := repo.approvals["NOPE"+string(owner)+string(spender)]; exists {
		t.Error("allowance row should not exist for a non-existent token")
	}
}

// TestDecreaseAllowance_TokenNotFound guards the dangling-token gap on the
// decrease side.
func TestDecreaseAllowance_TokenNotFound(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	spender := pubKey(2)
	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID:    "NOPE",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(10),
		PrivateKey: privKey(1),
	})
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
	if _, exists := repo.approvals["NOPE"+string(owner)+string(spender)]; exists {
		t.Error("allowance row should not exist for a non-existent token")
	}
}

// TestApprove_TokenNotFound guards the dangling-token gap on the approve
// side. Approve previously only checked the GetToken error (not nil-token),
// so a repo returning (nil, nil) for a missing token let SaveApproval create
// an orphaned allowance row referencing a token that never existed.
func TestApprove_TokenNotFound(t *testing.T) {
	repo := NewMockRepository()
	eventStore := NewMockEventStore()
	service := newTestService(repo, eventStore)

	owner := pubKey(1)
	spender := pubKey(2)
	_, err := service.Approve(&ApproveRequest{
		TokenID:    "NOPE",
		Owner:      owner,
		Spender:    spender,
		Amount:     NewAmount(50),
		PrivateKey: privKey(1),
	})
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
	if _, exists := repo.approvals["NOPE"+string(owner)+string(spender)]; exists {
		t.Error("approval row should not exist for a non-existent token")
	}
}
