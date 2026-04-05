package test

import (
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/lottery"
)

func TestLotteryE2E_FullFlow(t *testing.T) {
	participants := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	seed := "e2e-test-seed-123"
	count := 3

	_, sk, err := lottery.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	output, proof, err := lottery.VRFProve(sk, []byte(seed))
	if err != nil {
		t.Fatalf("VRFProve failed: %v", err)
	}

	winners := lottery.SelectWinners(output, participants, count)
	if len(winners) != count {
		t.Errorf("Expected %d winners, got %d", count, len(winners))
	}

	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = lottery.NameToAddress(w)
	}

	record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

	chain := blockchain.InitBlockChain()
	jsonData, err := record.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	height, err := chain.AddLotteryRecord(jsonData)
	if err != nil {
		t.Fatalf("AddLotteryRecord failed: %v", err)
	}
	record.BlockHeight = height

	if record.BlockHeight != 1 {
		t.Errorf("Expected block height 1, got %d", record.BlockHeight)
	}

	data, err := chain.GetBlockData(1)
	if err != nil {
		t.Fatalf("GetBlockData failed: %v", err)
	}
	if data != jsonData {
		t.Error("Block data mismatch")
	}

	t.Logf("Lottery ID: %s", record.ID)
	t.Logf("Winners: %v", winners)
	t.Logf("Block Height: %d", height)
}

func TestLotteryE2E_MultipleLotteries(t *testing.T) {
	chain := blockchain.InitBlockChain()
	initialHeight := len(chain.Blocks)

	seeds := []string{"seed-1", "seed-2", "seed-3"}
	participants := []string{"A", "B", "C", "D", "E"}

	for _, seed := range seeds {
		_, sk, _ := lottery.GenerateKeyPair()
		output, proof, _ := lottery.VRFProve(sk, []byte(seed))
		winners := lottery.SelectWinners(output, participants, 2)

		winnerAddrs := make([]string, len(winners))
		for i, w := range winners {
			winnerAddrs[i] = lottery.NameToAddress(w)
		}

		record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)
		jsonData, _ := record.ToJSON()
		chain.AddLotteryRecord(jsonData)
	}

	if len(chain.Blocks) != initialHeight+3 {
		t.Errorf("Expected %d blocks, got %d", initialHeight+3, len(chain.Blocks))
	}

	records := chain.GetLotteryRecords()
	if len(records) != 3 {
		t.Errorf("Expected 3 records, got %d", len(records))
	}

	t.Logf("Total blocks: %d", len(chain.Blocks))
	t.Logf("Total lottery records: %d", len(records))
}

func TestLotteryE2E_VerifyIntegrity(t *testing.T) {
	participants := []string{"Player1", "Player2", "Player3", "Player4", "Player5"}
	seed := "integrity-test-seed"

	_, sk, _ := lottery.GenerateKeyPair()
	output, proof, _ := lottery.VRFProve(sk, []byte(seed))
	winners := lottery.SelectWinners(output, participants, 2)

	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = lottery.NameToAddress(w)
	}

	record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)
	jsonData, _ := record.ToJSON()

	if record.Seed != seed {
		t.Error("Seed mismatch")
	}
	if len(record.Participants) != len(participants) {
		t.Error("Participants count mismatch")
	}
	if len(record.Winners) != 2 {
		t.Error("Winners count mismatch")
	}
	if record.VRFOutput == "" {
		t.Error("VRFOutput should not be empty")
	}
	if record.VRFProof == "" {
		t.Error("VRFProof should not be empty")
	}

	_ = jsonData

	for _, winner := range winners {
		found := false
		for _, p := range participants {
			if winner == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Winner %s is not in participants", winner)
		}
	}
}

func TestLotteryE2E_AddressConversion(t *testing.T) {
	tests := []struct {
		name     string
		wantLen  int
		wantPref string
	}{
		{"Alice", 42, "0x"},
		{"Bob", 42, "0x"},
		{"张三", 42, "0x"},
		{"", 42, "0x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lottery.NameToAddress(tt.name)
			if len(got) != tt.wantLen {
				t.Errorf("NameToAddress(%q) len = %d, want %d", tt.name, len(got), tt.wantLen)
			}
			if got[:2] != tt.wantPref {
				t.Errorf("NameToAddress(%q) = %v, want prefix %v", tt.name, got, tt.wantPref)
			}
		})
	}

	addr1 := lottery.NameToAddress("Test")
	addr2 := lottery.NameToAddress("Test")
	if addr1 != addr2 {
		t.Error("Same name should produce same address")
	}

	addr3 := lottery.NameToAddress("Test1")
	addr4 := lottery.NameToAddress("Test2")
	if addr3 == addr4 {
		t.Error("Different names should produce different addresses")
	}
}

func TestLotteryE2E_HistoryRetrieval(t *testing.T) {
	chain := blockchain.InitBlockChain()
	initialCount := len(chain.GetLotteryRecords())

	_, sk, _ := lottery.GenerateKeyPair()
	output, proof, _ := lottery.VRFProve(sk, []byte("history-test"))
	winners := lottery.SelectWinners(output, []string{"A", "B", "C", "D"}, 1)
	winnerAddrs := []string{lottery.NameToAddress(winners[0])}

	record := lottery.CreateLotteryRecord("history-test", []string{"A", "B", "C", "D"}, winners, winnerAddrs, output, proof, 0)
	jsonData, _ := record.ToJSON()
	chain.AddLotteryRecord(jsonData)

	records := chain.GetLotteryRecords()
	if len(records) != initialCount+1 {
		t.Errorf("Expected %d records, got %d", initialCount+1, len(records))
	}

	found := false
	for _, r := range records {
		if len(r) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("Should have at least one record")
	}
}
