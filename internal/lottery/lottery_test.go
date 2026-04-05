package lottery

import (
	"testing"
)

func TestSelectWinners(t *testing.T) {
	participants := []string{"张三", "李四", "王五", "赵六", "钱七"}
	output := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

	winners := SelectWinners(output, participants, 3)

	if len(winners) != 3 {
		t.Errorf("SelectWinners() = %v, want 3 winners", len(winners))
	}

	seen := make(map[string]bool)
	for _, w := range winners {
		if seen[w] {
			t.Errorf("Duplicate winner: %s", w)
		}
		seen[w] = true
	}

	for _, w := range winners {
		found := false
		for _, p := range participants {
			if w == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Winner %s is not a participant", w)
		}
	}
}

func TestSelectWinnersNotEnoughParticipants(t *testing.T) {
	participants := []string{"张三", "李四"}
	output := []byte{0x00, 0x01, 0x02, 0x03}

	winners := SelectWinners(output, participants, 5)

	if len(winners) != 2 {
		t.Errorf("SelectWinners() = %v, want 2 (all participants)", len(winners))
	}
}

func TestSelectWinnersEmptyParticipants(t *testing.T) {
	participants := []string{}
	output := []byte{0x00, 0x01}

	winners := SelectWinners(output, participants, 3)

	if len(winners) != 0 {
		t.Errorf("SelectWinners() = %v, want 0", len(winners))
	}
}

func TestCreateLotteryRecord(t *testing.T) {
	participants := []string{"张三", "李四", "王五"}
	seed := "test-seed"
	winners := []string{"王五"}
	winnerAddrs := []string{NameToAddress("王五")}
	output := []byte{0x01, 0x02, 0x03}
	proof := []byte{0x04, 0x05, 0x06}

	record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 1)

	if record.Seed != seed {
		t.Errorf("Seed = %v, want %v", record.Seed, seed)
	}
	if len(record.Participants) != 3 {
		t.Errorf("Participants len = %v, want 3", len(record.Participants))
	}
	if len(record.Winners) != 1 {
		t.Errorf("Winners len = %v, want 1", len(record.Winners))
	}
	if record.BlockHeight != 1 {
		t.Errorf("BlockHeight = %v, want 1", record.BlockHeight)
	}
	if record.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestLotteryRecordToJSON(t *testing.T) {
	record := &LotteryRecord{
		ID:      "test-id",
		Seed:    "test-seed",
		Winners: []string{"张三"},
	}

	jsonStr, err := record.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonStr) == 0 {
		t.Error("JSON should not be empty")
	}
}
