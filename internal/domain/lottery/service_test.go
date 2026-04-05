package lottery

import (
	"encoding/hex"
	"testing"
)

func TestLotteryService_DrawWinners(t *testing.T) {
	service := NewService()

	participants := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	seed := "test-seed-123"
	count := 2

	winners, winnerAddrs, output, proof, err := service.DrawWinners(participants, seed, count)
	if err != nil {
		t.Fatalf("DrawWinners failed: %v", err)
	}

	if len(winners) != count {
		t.Errorf("Expected %d winners, got %d", count, len(winners))
	}

	if len(winnerAddrs) != count {
		t.Errorf("Expected %d winner addresses, got %d", count, len(winnerAddrs))
	}

	if len(output) == 0 {
		t.Error("Expected non-empty VRF output")
	}

	if len(proof) == 0 {
		t.Error("Expected non-empty VRF proof")
	}

	for _, winner := range winners {
		found := false
		for _, p := range participants {
			if winner == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Winner %s not in participants", winner)
		}
	}
}

func TestLotteryService_DrawWinners_InvalidParticipants(t *testing.T) {
	service := NewService()

	_, _, _, _, err := service.DrawWinners([]string{}, "seed", 1)
	if err == nil {
		t.Fatal("Expected error for empty participants")
	}
}

func TestLotteryService_DrawWinners_InvalidSeed(t *testing.T) {
	service := NewService()

	_, _, _, _, err := service.DrawWinners([]string{"A", "B"}, "", 1)
	if err == nil {
		t.Fatal("Expected error for empty seed")
	}
}

func TestLotteryService_DrawWinners_InvalidWinnerCount(t *testing.T) {
	service := NewService()

	_, _, _, _, err := service.DrawWinners([]string{"A", "B"}, "seed", 5)
	if err == nil {
		t.Fatal("Expected error for winner count > participants")
	}
}

func TestLotteryService_DrawWinners_ZeroWinners(t *testing.T) {
	service := NewService()

	_, _, _, _, err := service.DrawWinners([]string{"A", "B"}, "seed", 0)
	if err == nil {
		t.Fatal("Expected error for zero winners")
	}
}

func TestLotteryService_VerifyDraw(t *testing.T) {
	service := NewService()

	participants := []string{"Alice", "Bob", "Charlie"}
	winners, _, output, proof, err := service.DrawWinners(participants, "test-seed", 1)
	if err != nil {
		t.Fatalf("DrawWinners failed: %v", err)
	}

	record := &LotteryRecord{
		Seed:      "test-seed",
		Winners:   winners,
		VRFOutput: hex.EncodeToString(output),
		VRFProof:  hex.EncodeToString(proof),
	}

	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	valid, err := service.VerifyDraw(record, pk)
	if err != nil {
		t.Fatalf("VerifyDraw failed: %v", err)
	}

	if !valid {
		t.Error("Expected draw to be valid")
	}
}

func TestLotteryService_VerifyDraw_NilRecord(t *testing.T) {
	service := NewService()

	valid, err := service.VerifyDraw(nil, nil)
	if err != nil {
		t.Fatalf("VerifyDraw failed: %v", err)
	}

	if valid {
		t.Error("Expected nil record to be invalid")
	}
}

func TestLotteryService_VerifyDraw_InvalidProof(t *testing.T) {
	service := NewService()

	record := &LotteryRecord{
		Seed:      "test-seed",
		Winners:   []string{"Alice"},
		VRFOutput: "invalid-output",
		VRFProof:  "invalid-proof",
	}

	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	_, err = service.VerifyDraw(record, pk)
	if err == nil {
		t.Fatal("Expected error for invalid VRF proof")
	}
}
