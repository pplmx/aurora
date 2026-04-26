package token

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/big"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/events"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

func pubKey(n byte) PublicKey {
	key := make(PublicKey, 32)
	for i := range key {
		key[i] = n
	}
	return key
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
		TokenID: "TEST",
		To:      nil,
		Amount:  NewAmount(500),
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
		TokenID: "TEST",
		To:      recipient,
		Amount:  NewAmount(0),
	}

	_, err := service.Mint(mintReq)
	if err == nil {
		t.Error("expected error for zero amount")
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
		TokenID: "TEST",
		To:      recipient,
		Amount:  NewAmount(500),
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
	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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

	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(500),
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
	spenderKey := make([]byte, 64)
	approveReq := &ApproveRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(500),
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(100),
	})

	_, err := service.IncreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(50),
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(100),
	})

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(30),
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

	privateKey := make([]byte, 64)
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

	privateKey := make([]byte, 64)
	_, err := service.Burn(&BurnRequest{
		TokenID:    "TEST",
		From:       owner,
		Amount:     NewAmount(200),
		PrivateKey: privateKey,
	})
	if err != ErrInsufficientBalance {
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
	privateKey := make([]byte, 64)
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

	txBackup map[string]*Amount
	txTokens map[TokenID]*Token
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
	m.txBackup = nil
	m.txTokens = nil
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

type mockBlockWriter struct {
	height int64
}

func (m *mockBlockWriter) AddBlock(data string) (int64, error) {
	m.height++
	return m.height, nil
}

func newTestService(repo Repository, eventStore *mockEventStore) *TokenService {
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
		TokenID: "FIXED",
		To:      recipient,
		Amount:  NewAmount(500),
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
		TokenID: "TEST",
		To:      recipient,
		Amount:  NewAmount(500),
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
	spenderKey := make([]byte, 64)
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(50),
	})

	spenderKey := make([]byte, 64)
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(500),
	})

	spenderKey := make([]byte, 64)
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
	spenderKey := make([]byte, 64)
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

	spenderKey := make([]byte, 64)
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
	spenderKey := make([]byte, 64)
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
	spenderKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
	repo.setAccountBalanceError = true
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
	privateKey := make([]byte, 64)
	repo.setAccountBalanceToError = true
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(100),
	})

	_, err := service.Approve(&ApproveRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(200),
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

	privateKey := make([]byte, 64)
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

	privateKey := make([]byte, 64)
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

	privateKey := make([]byte, 64)
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

	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(100),
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(10),
	})

	_, err := service.DecreaseAllowance(&AllowanceRequest{
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(5),
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
		TokenID: "TEST",
		Owner:   owner,
		Spender: spender,
		Amount:  NewAmount(50),
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

	spenderKey := make([]byte, 64)
	recipient := pubKey(3)

	repo.setAccountBalanceToError = true
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

func TestTransferFrom_UpdateApprovalError(t *testing.T) {
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

	spenderKey := make([]byte, 64)
	recipient := pubKey(3)

	repo.saveApprovalError = true
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
}

func newTestServiceWithRepo(repo *mockRepository, eventStore *mockEventStore) *TokenService {
	return NewService(repo, newMockTxManager(), newMockEventBus(eventStore), eventStore, newMockReplayProtection(), &mockBlockWriter{})
}

func TestTransfer_AtomicityRollbackOnPublishFailure(t *testing.T) {
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
	privateKey := make([]byte, 64)
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

	fromBalance := repo.balances[string(token.ID())+string(owner)]
	if fromBalance.Int64() != 1000 {
		t.Errorf("from balance should be unchanged (1000), got %d", fromBalance.Int64())
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
	spenderKey := make([]byte, 64)

	repo.setAccountBalanceToError = true
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
		TokenID: "TEST",
		To:      recipient,
		Amount:  NewAmount(500),
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

	privateKey := make([]byte, 64)
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
	privateKey := make([]byte, 64)
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
