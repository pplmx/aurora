package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/stretchr/testify/assert"
)

func TestLotteryE2E_ErrorHandling_InsufficientParticipants(t *testing.T) {
	_, sk, _ := lottery.GenerateKeyPair()

	participants := []string{"Alice", "Bob"}
	output, _, _ := lottery.VRFProve(sk, []byte("seed"))
	winners := lottery.SelectWinners(output, participants, 5)

	assert.True(t, len(winners) <= len(participants), "Should select at most len(participants) winners")
}

func TestLotteryE2E_ErrorHandling_EmptySeed(t *testing.T) {
	_, sk, _ := lottery.GenerateKeyPair()

	participants := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	output, proof, _ := lottery.VRFProve(sk, []byte(""))

	winners := lottery.SelectWinners(output, participants, 3)
	assert.Len(t, winners, 3, "Should still select winners with empty seed")

	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = lottery.NameToAddress(w)
	}

	record := lottery.CreateLotteryRecord("", participants, winners, winnerAddrs, output, proof, 0)
	assert.NotEmpty(t, record.ID)
}

func TestNFTE2E_ErrorHandling_EmptyName(t *testing.T) {
	pubkey := make([]byte, 32)
	copy(pubkey, []byte("creator-public-key-32-bytes-long!!"))

	nftItem := nft.NewNFT("", "A description", "", "", pubkey, pubkey)
	assert.Equal(t, "", nftItem.Name, "NFT should allow empty name")
}

func TestRecoveryScenario_LotteryWithDuplicateWinners(t *testing.T) {
	_, sk, _ := lottery.GenerateKeyPair()

	participants := []string{"Alice", "Bob", "Charlie", "Alice", "Bob"}
	output, proof, _ := lottery.VRFProve(sk, []byte("test-seed"))

	winners := lottery.SelectWinners(output, participants, 3)
	assert.Len(t, winners, 3, "Should select 3 winners")

	for _, w := range winners {
		assert.NotEmpty(t, w, "Winner should not be empty")
	}

	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = lottery.NameToAddress(w)
	}

	record := lottery.CreateLotteryRecord("test-seed", participants, winners, winnerAddrs, output, proof, 0)
	assert.NotEmpty(t, record.ID)
}

func TestRecoveryScenario_NFTCreation(t *testing.T) {
	pubkey := make([]byte, 32)
	copy(pubkey, []byte("test-public-key-32-bytes-long!!!!"))

	nftItem := nft.NewNFT("TestNFT", "Description", "", "", pubkey, pubkey)
	assert.NotNil(t, nftItem)
	assert.Equal(t, "TestNFT", nftItem.Name)
}

func TestTokenE2E_ErrorHandling_InsufficientBalance(t *testing.T) {
	repo := newInMemoryTokenRepo()
	eventStore := newInMemoryEventStore()
	eventBus := newE2EEventBus(eventStore)
	replay := newE2EReplayProtection()
	chain := blockchain.InitBlockChain()
	txManager := &noOpTxManager{}
	svc := token.NewService(repo, txManager, eventBus, eventStore, replay, chain)

	_, ownerPriv, _ := ed25519.GenerateKey(rand.Reader)
	ownerPub := token.PublicKey(ownerPriv.Public().(ed25519.PublicKey))

	_, recipientPriv, _ := ed25519.GenerateKey(rand.Reader)
	recipientPub := token.PublicKey(recipientPriv.Public().(ed25519.PublicKey))

	tok, err := svc.CreateToken(&token.CreateTokenRequest{
		Name:        "TestToken",
		Symbol:      "TEST",
		TotalSupply: token.NewAmount(100),
		Owner:       ownerPub,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Transfer(&token.TransferRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		To:         recipientPub,
		Amount:     token.NewAmount(200),
		PrivateKey: []byte(ownerPriv),
	})
	assert.Error(t, err, "Transfer exceeding balance should fail")
	assert.True(t, errors.Is(err, token.ErrInsufficientBalance), "Error should wrap ErrInsufficientBalance")

	ownerBal, _ := svc.GetBalance(tok.ID(), ownerPub)
	assert.Equal(t, token.NewAmount(100), ownerBal, "Owner balance should be unchanged")

	recipientBal, _ := svc.GetBalance(tok.ID(), recipientPub)
	assert.Equal(t, token.NewAmount(0), recipientBal, "Recipient balance should remain zero")
}

func TestTokenE2E_ErrorHandling_BurnInsufficientBalance(t *testing.T) {
	repo := newInMemoryTokenRepo()
	eventStore := newInMemoryEventStore()
	eventBus := newE2EEventBus(eventStore)
	replay := newE2EReplayProtection()
	chain := blockchain.InitBlockChain()
	txManager := &noOpTxManager{}
	svc := token.NewService(repo, txManager, eventBus, eventStore, replay, chain)

	_, ownerPriv, _ := ed25519.GenerateKey(rand.Reader)
	ownerPub := token.PublicKey(ownerPriv.Public().(ed25519.PublicKey))

	tok, err := svc.CreateToken(&token.CreateTokenRequest{
		Name:        "TestToken",
		Symbol:      "TEST",
		TotalSupply: token.NewAmount(50),
		Owner:       ownerPub,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Burn(&token.BurnRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		Amount:     token.NewAmount(100),
		PrivateKey: []byte(ownerPriv),
	})
	assert.Error(t, err, "Burn exceeding balance should fail")

	bal, _ := svc.GetBalance(tok.ID(), ownerPub)
	assert.Equal(t, token.NewAmount(50), bal, "Balance should be unchanged after failed burn")
}

func TestTokenE2E_ErrorHandling_TransferToSelf(t *testing.T) {
	repo := newInMemoryTokenRepo()
	eventStore := newInMemoryEventStore()
	eventBus := newE2EEventBus(eventStore)
	replay := newE2EReplayProtection()
	chain := blockchain.InitBlockChain()
	txManager := &noOpTxManager{}
	svc := token.NewService(repo, txManager, eventBus, eventStore, replay, chain)

	_, ownerPriv, _ := ed25519.GenerateKey(rand.Reader)
	ownerPub := token.PublicKey(ownerPriv.Public().(ed25519.PublicKey))

	tok, err := svc.CreateToken(&token.CreateTokenRequest{
		Name:        "TestToken",
		Symbol:      "TEST",
		TotalSupply: token.NewAmount(1000),
		Owner:       ownerPub,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Transfer(&token.TransferRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		To:         ownerPub,
		Amount:     token.NewAmount(100),
		PrivateKey: []byte(ownerPriv),
	})
	assert.NoError(t, err, "Transfer to self should succeed")

	bal, _ := svc.GetBalance(tok.ID(), ownerPub)
	assert.Equal(t, token.NewAmount(1000), bal, "Balance should be unchanged for transfer to self")
}

func TestTokenE2E_ErrorHandling_ZeroAmountTransfer(t *testing.T) {
	repo := newInMemoryTokenRepo()
	eventStore := newInMemoryEventStore()
	eventBus := newE2EEventBus(eventStore)
	replay := newE2EReplayProtection()
	chain := blockchain.InitBlockChain()
	txManager := &noOpTxManager{}
	svc := token.NewService(repo, txManager, eventBus, eventStore, replay, chain)

	_, ownerPriv, _ := ed25519.GenerateKey(rand.Reader)
	ownerPub := token.PublicKey(ownerPriv.Public().(ed25519.PublicKey))

	_, recipientPriv, _ := ed25519.GenerateKey(rand.Reader)
	recipientPub := token.PublicKey(recipientPriv.Public().(ed25519.PublicKey))

	tok, err := svc.CreateToken(&token.CreateTokenRequest{
		Name:        "TestToken",
		Symbol:      "TEST",
		TotalSupply: token.NewAmount(1000),
		Owner:       ownerPub,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Transfer(&token.TransferRequest{
		TokenID:    tok.ID(),
		From:       ownerPub,
		To:         recipientPub,
		Amount:     token.NewAmount(0),
		PrivateKey: []byte(ownerPriv),
	})
	assert.Error(t, err, "Transfer of zero amount should fail")
}
