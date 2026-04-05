package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
)

type mockTokenService struct {
	tokens    map[token.TokenID]*token.Token
	balances  map[string]*token.Amount
	approvals map[string]*token.Approval
	events    struct {
		transfers []*token.TransferEvent
		mints     []*token.MintEvent
		burns     []*token.BurnEvent
		approves  []*token.ApproveEvent
	}
	nonces map[string]uint64
}

func newMockTokenService() *mockTokenService {
	return &mockTokenService{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
		nonces:    make(map[string]uint64),
	}
}

func (m *mockTokenService) CreateToken(req *token.CreateTokenRequest) (*token.Token, error) {
	t := token.NewToken(token.TokenID(req.Symbol), req.Name, req.Symbol, req.TotalSupply, req.Owner)
	m.tokens[t.ID()] = t
	balanceKey := string(req.Owner) + "|" + string(t.ID())
	m.balances[balanceKey] = req.TotalSupply
	return t, nil
}

func (m *mockTokenService) GetTokenInfo(tokenID token.TokenID) (*token.Token, error) {
	return m.tokens[tokenID], nil
}

func (m *mockTokenService) Mint(req *token.MintRequest) (*token.MintEvent, error) {
	t, ok := m.tokens[req.TokenID]
	if !ok || t == nil {
		return nil, token.ErrTokenNotFound
	}

	event := token.NewMintEvent(req.TokenID, req.To, req.Amount)
	m.events.mints = append(m.events.mints, event)
	balanceKey := string(req.To) + "|" + string(req.TokenID)
	current := m.balances[balanceKey]
	if current == nil {
		current = token.NewAmount(0)
	}
	m.balances[balanceKey] = &token.Amount{current.Int.Add(current.Int, req.Amount.Int)}
	return event, nil
}

func (m *mockTokenService) Transfer(req *token.TransferRequest) (*token.TransferEvent, error) {
	balanceKey := string(req.From) + "|" + string(req.TokenID)
	fromBalance := m.balances[balanceKey]
	if fromBalance == nil || fromBalance.Cmp(req.Amount) < 0 {
		return nil, token.ErrInsufficientBalance
	}

	nonce := m.nonces[string(req.From)+string(req.TokenID)]
	nonce++
	m.nonces[string(req.From)+string(req.TokenID)] = nonce

	signature := ed25519.Sign(req.PrivateKey, []byte("mock-signature"))
	event := token.NewTransferEvent(req.TokenID, req.From, req.To, req.Amount, nonce, signature)
	m.events.transfers = append(m.events.transfers, event)

	fromNewBalance := &token.Amount{fromBalance.Int.Sub(fromBalance.Int, req.Amount.Int)}
	m.balances[balanceKey] = fromNewBalance

	toBalanceKey := string(req.To) + "|" + string(req.TokenID)
	toBalance := m.balances[toBalanceKey]
	if toBalance == nil {
		toBalance = token.NewAmount(0)
	}
	toNewBalance := &token.Amount{toBalance.Int.Add(toBalance.Int, req.Amount.Int)}
	m.balances[toBalanceKey] = toNewBalance

	return event, nil
}

func (m *mockTokenService) TransferFrom(req *token.TransferFromRequest) (*token.TransferEvent, error) {
	return nil, nil
}

func (m *mockTokenService) Approve(req *token.ApproveRequest) (*token.ApproveEvent, error) {
	approval := token.NewApproval(req.TokenID, req.Owner, req.Spender, req.Amount)
	key := string(req.Owner) + "|" + string(req.Spender) + "|" + string(req.TokenID)
	m.approvals[key] = approval
	event := token.NewApproveEvent(req.TokenID, req.Owner, req.Spender, req.Amount)
	m.events.approves = append(m.events.approves, event)
	return event, nil
}

func (m *mockTokenService) IncreaseAllowance(req *token.AllowanceRequest) (*token.ApproveEvent, error) {
	return nil, nil
}

func (m *mockTokenService) DecreaseAllowance(req *token.AllowanceRequest) (*token.ApproveEvent, error) {
	return nil, nil
}

func (m *mockTokenService) Burn(req *token.BurnRequest) (*token.BurnEvent, error) {
	balanceKey := string(req.From) + "|" + string(req.TokenID)
	fromBalance := m.balances[balanceKey]
	if fromBalance == nil || fromBalance.Cmp(req.Amount) < 0 {
		return nil, token.ErrInsufficientBalance
	}

	event := token.NewBurnEvent(req.TokenID, req.From, req.Amount)
	m.events.burns = append(m.events.burns, event)

	fromNewBalance := &token.Amount{fromBalance.Int.Sub(fromBalance.Int, req.Amount.Int)}
	m.balances[balanceKey] = fromNewBalance

	return event, nil
}

func (m *mockTokenService) GetBalance(tokenID token.TokenID, owner token.PublicKey) (*token.Amount, error) {
	balanceKey := string(owner) + "|" + string(tokenID)
	return m.balances[balanceKey], nil
}

func (m *mockTokenService) GetAllowance(tokenID token.TokenID, owner, spender token.PublicKey) (*token.Amount, error) {
	key := string(owner) + "|" + string(spender) + "|" + string(tokenID)
	if approval, ok := m.approvals[key]; ok {
		return approval.Amount(), nil
	}
	return token.NewAmount(0), nil
}

func (m *mockTokenService) GetTransferHistory(tokenID token.TokenID, owner token.PublicKey, limit int) ([]*token.TransferEvent, error) {
	return m.events.transfers, nil
}

func toTokenPublicKey(pub ed25519.PublicKey) token.PublicKey {
	return token.PublicKey(pub)
}

func TestCreateTokenUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	uc := NewCreateTokenUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: "1000000",
		Owner:       encodeBase64(pub),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Name != "Test Token" {
		t.Errorf("Expected name 'Test Token', got '%s'", resp.Name)
	}

	if resp.Symbol != "TEST" {
		t.Errorf("Expected symbol 'TEST', got '%s'", resp.Symbol)
	}
}

func TestCreateTokenUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewCreateTokenUseCase(service)

	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: "1000000",
		Owner:       "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestCreateTokenUseCase_InvalidTotalSupply(t *testing.T) {
	service := newMockTokenService()

	uc := NewCreateTokenUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	req := &CreateTokenRequest{
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: "invalid",
		Owner:       encodeBase64(pub),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid total supply")
	}
}

func TestMintUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, priv, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	uc := NewMintUseCase(service)

	req := &MintRequest{
		TokenID:    "TEST",
		To:         encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.TokenID != "TEST" {
		t.Errorf("Expected token ID 'TEST', got '%s'", resp.TokenID)
	}

	if resp.Amount != "100" {
		t.Errorf("Expected amount '100', got '%s'", resp.Amount)
	}
}

func TestMintUseCase_InvalidTo(t *testing.T) {
	service := newMockTokenService()

	uc := NewMintUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)
	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	req := &MintRequest{
		TokenID:    "TEST",
		To:         "!!!invalid!!!",
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid to")
	}
}

func TestMintUseCase_InvalidPrivateKey(t *testing.T) {
	service := newMockTokenService()

	uc := NewMintUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	req := &MintRequest{
		TokenID:    "TEST",
		To:         encodeBase64(pub),
		Amount:     "100",
		PrivateKey: "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid private key")
	}
}

func TestMintUseCase_TokenNotFound(t *testing.T) {
	service := newMockTokenService()

	uc := NewMintUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &MintRequest{
		TokenID:    "NOTEXIST",
		To:         encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for nonexistent token")
	}
}

func TestTransferUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub1, priv1, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub1))
	service.balances[string(toTokenPublicKey(pub1))+"|TEST"] = token.NewAmount(1000)

	uc := NewTransferUseCase(service)

	req := &TransferRequest{
		TokenID:    "TEST",
		From:       encodeBase64(pub1),
		To:         encodeBase64(pub2),
		Amount:     "100",
		PrivateKey: encodeBase64(priv1),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Amount != "100" {
		t.Errorf("Expected amount '100', got '%s'", resp.Amount)
	}
}

func TestTransferUseCase_InsufficientBalance(t *testing.T) {
	service := newMockTokenService()

	pub1, priv1, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub1))
	service.balances[string(toTokenPublicKey(pub1))+"|TEST"] = token.NewAmount(50)

	uc := NewTransferUseCase(service)

	req := &TransferRequest{
		TokenID:    "TEST",
		From:       encodeBase64(pub1),
		To:         encodeBase64(pub2),
		Amount:     "100",
		PrivateKey: encodeBase64(priv1),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for insufficient balance")
	}
}

func TestTransferUseCase_InvalidFrom(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &TransferRequest{
		TokenID:    "TEST",
		From:       "!!!invalid!!!",
		To:         encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid from")
	}
}

func TestTransferUseCase_InvalidTo(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &TransferRequest{
		TokenID:    "TEST",
		From:       encodeBase64(pub),
		To:         "!!!invalid!!!",
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid to")
	}
}

func TestBurnUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, priv, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))
	service.balances[string(toTokenPublicKey(pub))+"|TEST"] = token.NewAmount(1000)

	uc := NewBurnUseCase(service)

	req := &BurnRequest{
		TokenID:    "TEST",
		From:       encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Amount != "100" {
		t.Errorf("Expected amount '100', got '%s'", resp.Amount)
	}
}

func TestBurnUseCase_InsufficientBalance(t *testing.T) {
	service := newMockTokenService()

	pub, priv, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))
	service.balances[string(toTokenPublicKey(pub))+"|TEST"] = token.NewAmount(50)

	uc := NewBurnUseCase(service)

	req := &BurnRequest{
		TokenID:    "TEST",
		From:       encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for insufficient balance")
	}
}

func TestBurnUseCase_InvalidFrom(t *testing.T) {
	service := newMockTokenService()

	uc := NewBurnUseCase(service)

	_, priv, _ := ed25519.GenerateKey(nil)

	req := &BurnRequest{
		TokenID:    "TEST",
		From:       "!!!invalid!!!",
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid from")
	}
}

func TestBurnUseCase_TokenNotFound(t *testing.T) {
	service := newMockTokenService()

	uc := NewBurnUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &BurnRequest{
		TokenID:    "NOTEXIST",
		From:       encodeBase64(pub),
		Amount:     "100",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for nonexistent token")
	}
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
