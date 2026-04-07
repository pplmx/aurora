package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
)

type inMemoryTokenRepo struct {
	tokens    map[token.TokenID]*token.Token
	balances  map[string]*token.Amount
	approvals map[string]*token.Approval
}

func newInMemoryTokenRepo() *inMemoryTokenRepo {
	return &inMemoryTokenRepo{
		tokens:    make(map[token.TokenID]*token.Token),
		balances:  make(map[string]*token.Amount),
		approvals: make(map[string]*token.Approval),
	}
}

func (r *inMemoryTokenRepo) SaveToken(t *token.Token) error {
	r.tokens[t.ID()] = t
	return nil
}

func (r *inMemoryTokenRepo) GetToken(id token.TokenID) (*token.Token, error) {
	return r.tokens[id], nil
}

func (r *inMemoryTokenRepo) SaveApproval(a *token.Approval) error {
	key := string(a.TokenID()) + "|" + string(a.Owner()) + "|" + string(a.Spender())
	r.approvals[key] = a
	return nil
}

func (r *inMemoryTokenRepo) GetApproval(tokenID token.TokenID, owner, spender token.PublicKey) (*token.Approval, error) {
	key := string(tokenID) + "|" + string(owner) + "|" + string(spender)
	return r.approvals[key], nil
}

func (r *inMemoryTokenRepo) GetApprovalsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.Approval, error) {
	var result []*token.Approval
	for _, a := range r.approvals {
		if a.TokenID() == tokenID && string(a.Owner()) == string(owner) {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *inMemoryTokenRepo) GetAccountBalance(tokenID token.TokenID, owner token.PublicKey) (*token.Amount, error) {
	key := string(tokenID) + "|" + string(owner)
	if balance, ok := r.balances[key]; ok {
		return balance, nil
	}
	return token.NewAmount(0), nil
}

func (r *inMemoryTokenRepo) SetAccountBalance(tokenID token.TokenID, owner token.PublicKey, amount *token.Amount) error {
	key := string(tokenID) + "|" + string(owner)
	r.balances[key] = amount
	return nil
}

type inMemoryEventStore struct {
	transferEvents []*token.TransferEvent
	mintEvents     []*token.MintEvent
	burnEvents     []*token.BurnEvent
	approveEvents  []*token.ApproveEvent
	nonces         map[string]uint64
}

func newInMemoryEventStore() *inMemoryEventStore {
	return &inMemoryEventStore{
		transferEvents: make([]*token.TransferEvent, 0),
		mintEvents:     make([]*token.MintEvent, 0),
		burnEvents:     make([]*token.BurnEvent, 0),
		approveEvents:  make([]*token.ApproveEvent, 0),
		nonces:         make(map[string]uint64),
	}
}

func (e *inMemoryEventStore) SaveTransferEvent(event *token.TransferEvent) error {
	e.transferEvents = append(e.transferEvents, event)
	return nil
}

func (e *inMemoryEventStore) SaveMintEvent(event *token.MintEvent) error {
	e.mintEvents = append(e.mintEvents, event)
	return nil
}

func (e *inMemoryEventStore) SaveBurnEvent(event *token.BurnEvent) error {
	e.burnEvents = append(e.burnEvents, event)
	return nil
}

func (e *inMemoryEventStore) SaveApproveEvent(event *token.ApproveEvent) error {
	e.approveEvents = append(e.approveEvents, event)
	return nil
}

func (e *inMemoryEventStore) GetTransferEventsByToken(tokenID token.TokenID) ([]*token.TransferEvent, error) {
	var result []*token.TransferEvent
	for _, event := range e.transferEvents {
		if event.TokenID() == tokenID {
			result = append(result, event)
		}
	}
	return result, nil
}

func (e *inMemoryEventStore) GetTransferEventsByOwner(tokenID token.TokenID, owner token.PublicKey) ([]*token.TransferEvent, error) {
	var result []*token.TransferEvent
	for _, event := range e.transferEvents {
		if event.TokenID() == tokenID && (string(event.From()) == string(owner) || string(event.To()) == string(owner)) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (e *inMemoryEventStore) GetMintEventsByToken(tokenID token.TokenID) ([]*token.MintEvent, error) {
	var result []*token.MintEvent
	for _, event := range e.mintEvents {
		if event.TokenID() == tokenID {
			result = append(result, event)
		}
	}
	return result, nil
}

func (e *inMemoryEventStore) GetBurnEventsByToken(tokenID token.TokenID) ([]*token.BurnEvent, error) {
	var result []*token.BurnEvent
	for _, event := range e.burnEvents {
		if event.TokenID() == tokenID {
			result = append(result, event)
		}
	}
	return result, nil
}

func (e *inMemoryEventStore) GetLastNonce(tokenID token.TokenID, owner token.PublicKey) (uint64, error) {
	key := string(tokenID) + "|" + string(owner)
	return e.nonces[key], nil
}

func (e *inMemoryEventStore) SaveNonce(tokenID token.TokenID, owner token.PublicKey, nonce uint64) error {
	key := string(tokenID) + "|" + string(owner)
	e.nonces[key] = nonce
	return nil
}

func TestTokenE2E_FullFlow(t *testing.T) {
	blockchain.ResetForTest()

	repo := newInMemoryTokenRepo()
	eventStore := newInMemoryEventStore()
	chain := blockchain.InitBlockChain()
	svc := token.NewService(repo, eventStore, chain)

	_, ownerPriv, _ := ed25519.GenerateKey(rand.Reader)
	ownerPub := token.PublicKey(ownerPriv.Public().(ed25519.PublicKey))

	_, recipientPriv, _ := ed25519.GenerateKey(rand.Reader)
	recipientPub := token.PublicKey(recipientPriv.Public().(ed25519.PublicKey))

	ownerPrivKey := []byte(ownerPriv)
	_ = []byte(recipientPriv)

	createReq := &token.CreateTokenRequest{
		Name:        "TestToken",
		Symbol:      "TEST",
		TotalSupply: token.NewAmount(1000000),
		Owner:       ownerPub,
	}

	tok, err := svc.CreateToken(createReq)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Name() != "TestToken" {
		t.Errorf("Name = %v, want TestToken", tok.Name())
	}
	if tok.Symbol() != "TEST" {
		t.Errorf("Symbol = %v, want TEST", tok.Symbol())
	}

	ownerBalance, err := svc.GetBalance(tok.ID(), ownerPub)
	if err != nil {
		t.Fatal(err)
	}
	if ownerBalance.Cmp(token.NewAmount(1000000)) != 0 {
		t.Errorf("Initial balance = %v, want 1000000", ownerBalance)
	}

	mintReq := &token.MintRequest{
		TokenID:    tok.ID(),
		To:         ownerPub,
		Amount:     token.NewAmount(500),
		PrivateKey: ownerPrivKey,
	}
	_, err = svc.Mint(mintReq)
	if err != nil {
		t.Fatal(err)
	}

	ownerBalance, err = svc.GetBalance(tok.ID(), ownerPub)
	if err != nil {
		t.Fatal(err)
	}
	if ownerBalance.Cmp(token.NewAmount(1000500)) != 0 {
		t.Errorf("Balance after mint = %v, want 1000500", ownerBalance)
	}

	transferReq := &token.TransferRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		To:         recipientPub,
		Amount:     token.NewAmount(100),
		PrivateKey: ownerPrivKey,
	}
	_, err = svc.Transfer(transferReq)
	if err != nil {
		t.Fatal(err)
	}

	ownerBalance, err = svc.GetBalance(tok.ID(), ownerPub)
	if err != nil {
		t.Fatal(err)
	}
	if ownerBalance.Cmp(token.NewAmount(1000400)) != 0 {
		t.Errorf("Owner balance after transfer = %v, want 1000400", ownerBalance)
	}

	recipientBalance, err := svc.GetBalance(tok.ID(), recipientPub)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance.Cmp(token.NewAmount(100)) != 0 {
		t.Errorf("Recipient balance after transfer = %v, want 100", recipientBalance)
	}

	burnReq := &token.BurnRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		Amount:     token.NewAmount(50),
		PrivateKey: ownerPrivKey,
	}
	_, err = svc.Burn(burnReq)
	if err != nil {
		t.Fatal(err)
	}

	finalBalance, err := svc.GetBalance(tok.ID(), ownerPub)
	if err != nil {
		t.Fatal(err)
	}
	expectedBalance := token.NewAmount(1000350)
	if finalBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("Final balance = %v, want %v", finalBalance, expectedBalance)
	}

	mintEvents, _ := eventStore.GetMintEventsByToken(tok.ID())
	if len(mintEvents) != 1 {
		t.Errorf("len(mintEvents) = %v, want 1", len(mintEvents))
	}

	transferEvents, _ := eventStore.GetTransferEventsByToken(tok.ID())
	if len(transferEvents) != 1 {
		t.Errorf("len(transferEvents) = %v, want 1", len(transferEvents))
	}

	t.Log("E2E test passed!")
}
