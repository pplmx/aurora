package test

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
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
