package token

import (
	"bytes"
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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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
	service := NewService(repo, eventBus, eventStore, replay, chain)

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

	history, err := service.GetTransferHistory("TEST", owner, 10)
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
}

func NewMockRepository() *mockRepository {
	return &mockRepository{
		tokens:    make(map[TokenID]*Token),
		balances:  make(map[string]*Amount),
		approvals: make(map[string]*Approval),
	}
}

func (m *mockRepository) SaveToken(token *Token) error {
	m.tokens[token.ID()] = token
	return nil
}

func (m *mockRepository) GetToken(id TokenID) (*Token, error) {
	return m.tokens[id], nil
}

func (m *mockRepository) SaveApproval(approval *Approval) error {
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
	key := string(tokenID) + string(owner)
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

func (m *mockEventStore) GetTransferEventsByOwner(tokenID TokenID, owner PublicKey) ([]*TransferEvent, error) {
	var result []*TransferEvent
	for _, e := range m.transferEvents {
		if e.TokenID() == tokenID && (bytes.Equal(e.From(), owner) || bytes.Equal(e.To(), owner)) {
			result = append(result, e)
		}
	}
	return result, nil
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
	return NewService(repo, newMockEventBus(eventStore), eventStore, newMockReplayProtection(), &mockBlockWriter{})
}
