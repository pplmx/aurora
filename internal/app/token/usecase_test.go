package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/stretchr/testify/require"
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
	newInt := new(big.Int).Add(current.Int, req.Amount.Int)
	m.balances[balanceKey] = &token.Amount{Int: newInt}
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

	fromNewBalanceInt := new(big.Int).Sub(fromBalance.Int, req.Amount.Int)
	fromNewBalance := &token.Amount{Int: fromNewBalanceInt}
	m.balances[balanceKey] = fromNewBalance

	toBalanceKey := string(req.To) + "|" + string(req.TokenID)
	toBalance := m.balances[toBalanceKey]
	if toBalance == nil {
		toBalance = token.NewAmount(0)
	}
	toNewBalanceInt := new(big.Int).Add(toBalance.Int, req.Amount.Int)
	toNewBalance := &token.Amount{Int: toNewBalanceInt}
	m.balances[toBalanceKey] = toNewBalance

	return event, nil
}

func (m *mockTokenService) TransferFrom(req *token.TransferFromRequest) (*token.TransferEvent, error) {
	allowanceKey := string(req.Owner) + "|" + string(req.Spender) + "|" + string(req.TokenID)
	allowance := m.approvals[allowanceKey]
	if allowance == nil || allowance.Amount().Cmp(req.Amount) < 0 {
		return nil, token.ErrInsufficientAllowance
	}

	balanceKey := string(req.Owner) + "|" + string(req.TokenID)
	fromBalance := m.balances[balanceKey]
	if fromBalance == nil || fromBalance.Cmp(req.Amount) < 0 {
		return nil, token.ErrInsufficientBalance
	}

	nonce := m.nonces[string(req.Spender)+string(req.TokenID)]
	nonce++
	m.nonces[string(req.Spender)+string(req.TokenID)] = nonce

	signature := ed25519.Sign(req.SpenderKey, []byte("mock-signature-from"))
	event := token.NewTransferEvent(req.TokenID, req.Owner, req.To, req.Amount, nonce, signature)
	m.events.transfers = append(m.events.transfers, event)

	fromNewBalanceInt := new(big.Int).Sub(fromBalance.Int, req.Amount.Int)
	m.balances[balanceKey] = &token.Amount{Int: fromNewBalanceInt}

	toBalanceKey := string(req.To) + "|" + string(req.TokenID)
	toBalance := m.balances[toBalanceKey]
	if toBalance == nil {
		toBalance = token.NewAmount(0)
	}
	toNewBalanceInt := new(big.Int).Add(toBalance.Int, req.Amount.Int)
	m.balances[toBalanceKey] = &token.Amount{Int: toNewBalanceInt}

	newAllowanceInt := new(big.Int).Sub(allowance.Amount().Int, req.Amount.Int)
	m.approvals[allowanceKey] = token.NewApproval(req.TokenID, req.Owner, req.Spender, &token.Amount{Int: newAllowanceInt})

	return event, nil
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

	burnNewBalanceInt := new(big.Int).Sub(fromBalance.Int, req.Amount.Int)
	fromNewBalance := &token.Amount{Int: burnNewBalanceInt}
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
	require.NoError(t, err)
	require.NotNil(t, resp)

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
	require.NoError(t, err)
	require.NotNil(t, resp)

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
	require.NoError(t, err)
	require.NotNil(t, resp)

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
	require.NoError(t, err)
	require.NotNil(t, resp)

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

func TestApproveUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, priv, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	uc := NewApproveUseCase(service)

	req := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		Spender:    encodeBase64(pub2),
		Amount:     "500",
		PrivateKey: encodeBase64(priv),
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Amount != "500" {
		t.Errorf("Expected amount '500', got '%s'", resp.Amount)
	}

	if resp.TokenID != "TEST" {
		t.Errorf("Expected token ID 'TEST', got '%s'", resp.TokenID)
	}
}

func TestApproveUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewApproveUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      "!!!invalid!!!",
		Spender:    encodeBase64(pub),
		Amount:     "500",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestApproveUseCase_InvalidSpender(t *testing.T) {
	service := newMockTokenService()

	uc := NewApproveUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)

	req := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		Spender:    "!!!invalid!!!",
		Amount:     "500",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid spender")
	}
}

func TestApproveUseCase_InvalidPrivateKey(t *testing.T) {
	service := newMockTokenService()

	uc := NewApproveUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	req := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		Spender:    encodeBase64(pub2),
		Amount:     "500",
		PrivateKey: "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid private key")
	}
}

func TestApproveUseCase_InvalidAmount(t *testing.T) {
	service := newMockTokenService()

	uc := NewApproveUseCase(service)

	pub, priv, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	req := &ApproveRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		Spender:    encodeBase64(pub2),
		Amount:     "invalid",
		PrivateKey: encodeBase64(priv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid amount")
	}
}

func TestTransferFromUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	spenderPub, spenderPriv, _ := ed25519.GenerateKey(nil)
	toPub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(ownerPub))
	service.balances[string(toTokenPublicKey(ownerPub))+"|TEST"] = token.NewAmount(1000)
	service.approvals[string(toTokenPublicKey(ownerPub))+"|"+string(toTokenPublicKey(spenderPub))+"|TEST"] = token.NewApproval("TEST", toTokenPublicKey(ownerPub), toTokenPublicKey(spenderPub), token.NewAmount(500))

	uc := NewTransferFromUseCase(service)
	_ = ownerPriv

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(ownerPub),
		To:         encodeBase64(toPub),
		Amount:     "100",
		Spender:    encodeBase64(spenderPub),
		SpenderKey: encodeBase64(spenderPriv),
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Amount != "100" {
		t.Errorf("Expected amount '100', got '%s'", resp.Amount)
	}
}

func TestTransferFromUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferFromUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      "!!!invalid!!!",
		To:         encodeBase64(pub),
		Amount:     "100",
		Spender:    encodeBase64(pub),
		SpenderKey: encodeBase64(pub),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestTransferFromUseCase_InvalidTo(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferFromUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	_ = service

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		To:         "!!!invalid!!!",
		Amount:     "100",
		Spender:    encodeBase64(pub),
		SpenderKey: encodeBase64(pub),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid to")
	}
}

func TestTransferFromUseCase_InvalidSpender(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferFromUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		To:         encodeBase64(pub),
		Amount:     "100",
		Spender:    "!!!invalid!!!",
		SpenderKey: encodeBase64(pub),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid spender")
	}
}

func TestTransferFromUseCase_InvalidSpenderKey(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferFromUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		To:         encodeBase64(pub2),
		Amount:     "100",
		Spender:    encodeBase64(pub2),
		SpenderKey: "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid spender key")
	}
}

func TestTransferFromUseCase_InvalidAmount(t *testing.T) {
	service := newMockTokenService()

	uc := NewTransferFromUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(pub),
		To:         encodeBase64(pub2),
		Amount:     "invalid",
		Spender:    encodeBase64(pub2),
		SpenderKey: encodeBase64(pub2),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid amount")
	}
}

func TestTransferFromUseCase_InsufficientAllowance(t *testing.T) {
	service := newMockTokenService()

	ownerPub, _, _ := ed25519.GenerateKey(nil)
	spenderPub, spenderPriv, _ := ed25519.GenerateKey(nil)
	toPub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(ownerPub))
	service.balances[string(toTokenPublicKey(ownerPub))+"|TEST"] = token.NewAmount(1000)
	service.approvals[string(toTokenPublicKey(ownerPub))+"|"+string(toTokenPublicKey(spenderPub))+"|TEST"] = token.NewApproval("TEST", toTokenPublicKey(ownerPub), toTokenPublicKey(spenderPub), token.NewAmount(50))

	uc := NewTransferFromUseCase(service)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(ownerPub),
		To:         encodeBase64(toPub),
		Amount:     "100",
		Spender:    encodeBase64(spenderPub),
		SpenderKey: encodeBase64(spenderPriv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for insufficient allowance")
	}
}

func TestTransferFromUseCase_InsufficientBalance(t *testing.T) {
	service := newMockTokenService()

	ownerPub, _, _ := ed25519.GenerateKey(nil)
	spenderPub, spenderPriv, _ := ed25519.GenerateKey(nil)
	toPub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(ownerPub))
	service.balances[string(toTokenPublicKey(ownerPub))+"|TEST"] = token.NewAmount(50)
	service.approvals[string(toTokenPublicKey(ownerPub))+"|"+string(toTokenPublicKey(spenderPub))+"|TEST"] = token.NewApproval("TEST", toTokenPublicKey(ownerPub), toTokenPublicKey(spenderPub), token.NewAmount(500))

	uc := NewTransferFromUseCase(service)

	req := &TransferFromRequest{
		TokenID:    "TEST",
		Owner:      encodeBase64(ownerPub),
		To:         encodeBase64(toPub),
		Amount:     "100",
		Spender:    encodeBase64(spenderPub),
		SpenderKey: encodeBase64(spenderPriv),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for insufficient balance")
	}
}

func TestGetBalanceUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))
	service.balances[string(toTokenPublicKey(pub))+"|TEST"] = token.NewAmount(500)

	uc := NewGetBalanceUseCase(service)

	req := &BalanceRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Amount != "500" {
		t.Errorf("Expected amount '500', got '%s'", resp.Amount)
	}
}

func TestGetBalanceUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewGetBalanceUseCase(service)

	req := &BalanceRequest{
		TokenID: "TEST",
		Owner:   "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestGetBalanceUseCase_NonExistentOwner(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	uc := NewGetBalanceUseCase(service)

	req := &BalanceRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Amount != "0" {
		t.Errorf("Expected amount '0', got '%s'", resp.Amount)
	}
}

func TestGetAllowanceUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))
	service.approvals[string(toTokenPublicKey(pub))+"|"+string(toTokenPublicKey(pub2))+"|TEST"] = token.NewApproval("TEST", toTokenPublicKey(pub), toTokenPublicKey(pub2), token.NewAmount(300))

	uc := NewGetAllowanceUseCase(service)

	req := &AllowanceRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
		Spender: encodeBase64(pub2),
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Amount != "300" {
		t.Errorf("Expected amount '300', got '%s'", resp.Amount)
	}
}

func TestGetAllowanceUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewGetAllowanceUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)

	req := &AllowanceRequest{
		TokenID: "TEST",
		Owner:   "!!!invalid!!!",
		Spender: encodeBase64(pub),
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestGetAllowanceUseCase_InvalidSpender(t *testing.T) {
	service := newMockTokenService()

	uc := NewGetAllowanceUseCase(service)

	pub, _, _ := ed25519.GenerateKey(nil)

	req := &AllowanceRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
		Spender: "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid spender")
	}
}

func TestGetAllowanceUseCase_NoAllowance(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	uc := NewGetAllowanceUseCase(service)

	req := &AllowanceRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
		Spender: encodeBase64(pub2),
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Amount != "0" {
		t.Errorf("Expected amount '0', got '%s'", resp.Amount)
	}
}

func TestGetHistoryUseCase_Execute(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))
	service.events.transfers = []*token.TransferEvent{
		token.NewTransferEvent("TEST", toTokenPublicKey(pub), toTokenPublicKey(pub), token.NewAmount(100), 1, nil),
		token.NewTransferEvent("TEST", toTokenPublicKey(pub), toTokenPublicKey(pub), token.NewAmount(200), 2, nil),
	}

	uc := NewGetHistoryUseCase(service)

	req := &HistoryRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
		Limit:   10,
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if len(resp.Transfers) != 2 {
		t.Errorf("Expected 2 transfers, got %d", len(resp.Transfers))
	}
}

func TestGetHistoryUseCase_InvalidOwner(t *testing.T) {
	service := newMockTokenService()

	uc := NewGetHistoryUseCase(service)

	req := &HistoryRequest{
		TokenID: "TEST",
		Owner:   "!!!invalid!!!",
		Limit:   10,
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestGetHistoryUseCase_EmptyHistory(t *testing.T) {
	service := newMockTokenService()

	pub, _, _ := ed25519.GenerateKey(nil)

	service.tokens["TEST"] = token.NewToken("TEST", "Test Token", "TEST", token.NewAmount(1000000), toTokenPublicKey(pub))

	uc := NewGetHistoryUseCase(service)

	req := &HistoryRequest{
		TokenID: "TEST",
		Owner:   encodeBase64(pub),
		Limit:   10,
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp.Transfers) != 0 {
		t.Errorf("Expected 0 transfers, got %d", len(resp.Transfers))
	}
}
