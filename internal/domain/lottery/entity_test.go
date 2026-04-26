package lottery

import (
	"encoding/hex"
	"testing"
)

func TestGetWinners(t *testing.T) {
	record := &LotteryRecord{
		Winners: []string{"Alice", "Bob"},
	}
	got := record.GetWinners()
	if len(got) != 2 || got[0] != "Alice" || got[1] != "Bob" {
		t.Errorf("GetWinners() = %v, want [Alice Bob]", got)
	}
}

func TestLotteryRecord_ToJSON(t *testing.T) {
	record := &LotteryRecord{
		ID:           "test-id",
		Seed:         "test-seed",
		Participants: []string{"Alice", "Bob"},
		Winners:      []string{"Alice"},
	}

	jsonStr, err := record.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if jsonStr == "" {
		t.Error("ToJSON() returned empty string")
	}

	record2 := &LotteryRecord{}
	if err := record2.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if record2.ID != record.ID || record2.Seed != record.Seed {
		t.Errorf("FromJSON() got ID=%s, Seed=%s, want ID=%s, Seed=%s",
			record2.ID, record2.Seed, record.ID, record.Seed)
	}
}

func TestLotteryRecord_FromJSON_Invalid(t *testing.T) {
	record := &LotteryRecord{}
	err := record.FromJSON("invalid json{{{")
	if err == nil {
		t.Error("FromJSON() expected error for invalid JSON")
	}
}

func TestCreateLotteryRecord(t *testing.T) {
	seed := "test-seed-123"
	participants := []string{"Alice", "Bob", "Charlie"}
	winners := []string{"Alice"}
	winnerAddrs := []string{"0x123"}
	output := []byte("test-output-32-bytes........")
	proof := []byte("test-proof-64-bytes....................................")
	blockHeight := int64(12345)

	record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, blockHeight)

	if record.ID == "" {
		t.Error("CreateLotteryRecord() ID should not be empty")
	}
	if len(record.ID) != 16 {
		t.Errorf("CreateLotteryRecord() ID length = %d, want 16", len(record.ID))
	}
	if record.Seed != seed {
		t.Errorf("CreateLotteryRecord() Seed = %s, want %s", record.Seed, seed)
	}
	if len(record.VRFOutput) == 0 {
		t.Error("CreateLotteryRecord() VRFOutput should not be empty")
	}
	if len(record.VRFProof) == 0 {
		t.Error("CreateLotteryRecord() VRFProof should not be empty")
	}
	if record.BlockHeight != blockHeight {
		t.Errorf("CreateLotteryRecord() BlockHeight = %d, want %d", record.BlockHeight, blockHeight)
	}
	if record.Timestamp == 0 {
		t.Error("CreateLotteryRecord() Timestamp should be set")
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello\x00world", "helloworld"},
		{"test\x1Fmore", "testmore"},
		{"normal text", "normal text"},
		{"  multiple   spaces  ", "multiple   spaces"},
	}

	for _, tt := range tests {
		got := SanitizeString(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNameToAddress(t *testing.T) {
	addr := NameToAddress("Alice")
	if addr == "" {
		t.Error("NameToAddress() returned empty")
	}
	if len(addr) != 42 { // 0x + 40 hex chars
		t.Errorf("NameToAddress() length = %d, want 42", len(addr))
	}
	if addr[:2] != "0x" {
		t.Errorf("NameToAddress() should start with 0x, got %s", addr[:2])
	}

	_, err := hex.DecodeString(addr[2:])
	if err != nil {
		t.Errorf("NameToAddress() should produce valid hex: %v", err)
	}
}

func TestSelectWinners_EmptyParticipants(t *testing.T) {
	output := []byte("test-output-32-bytes..........")
	winners := SelectWinners(output, []string{}, 1)
	if len(winners) != 0 {
		t.Errorf("SelectWinners() with empty participants = %v, want []", winners)
	}
}

func TestSelectWinners_CountExceedsParticipants(t *testing.T) {
	output := []byte("test-output-32-bytes..........")
	participants := []string{"Alice", "Bob"}
	winners := SelectWinners(output, participants, 5)
	if len(winners) != 2 {
		t.Errorf("SelectWinners() count > participants = %d, want 2", len(winners))
	}
}

func TestSelectWinners_ExactCount(t *testing.T) {
	output := []byte("test-output-32-bytes..........")
	participants := []string{"Alice", "Bob", "Charlie"}
	winners := SelectWinners(output, participants, 3)
	if len(winners) != 3 {
		t.Errorf("SelectWinners() exact count = %d, want 3", len(winners))
	}
}

func TestSelectWinners_OneWinner(t *testing.T) {
	output := []byte("test-output-32-bytes..........")
	participants := []string{"Alice", "Bob", "Charlie"}
	winners := SelectWinners(output, participants, 1)
	if len(winners) != 1 {
		t.Errorf("SelectWinners() one winner = %d, want 1", len(winners))
	}
}

func TestSelectWinners_AllUnique(t *testing.T) {
	output := []byte("test-output-32-bytes..........")
	participants := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	winners := SelectWinners(output, participants, 3)
	seen := make(map[string]bool)
	for _, w := range winners {
		if seen[w] {
			t.Errorf("SelectWinners() produced duplicate winner: %s", w)
		}
		seen[w] = true
	}
}

func TestValidateParticipantName_Empty(t *testing.T) {
	err := ValidateParticipantName("")
	if err == nil {
		t.Error("ValidateParticipantName() empty should error")
	}
}

func TestValidateParticipantName_TooLong(t *testing.T) {
	longName := ""
	for i := 0; i < MaxParticipantNameLength+1; i++ {
		longName += "a"
	}
	err := ValidateParticipantName(longName)
	if err == nil {
		t.Error("ValidateParticipantName() too long should error")
	}
}

func TestValidateParticipantName_InvalidChars(t *testing.T) {
	invalid := []string{"test@name", "name#1", "bad<>name", "test!name"}
	for _, name := range invalid {
		err := ValidateParticipantName(name)
		if err == nil {
			t.Errorf("ValidateParticipantName(%q) should error for invalid chars", name)
		}
	}
}

func TestValidateParticipantName_Valid(t *testing.T) {
	valid := []string{"Alice", "Bob 123", "test-name", "Name_With", "日本語", "中文"}
	for _, name := range valid {
		err := ValidateParticipantName(name)
		if err != nil {
			t.Errorf("ValidateParticipantName(%q) should not error: %v", name, err)
		}
	}
}

func TestValidateSeed_TooShort(t *testing.T) {
	err := ValidateSeed("ab")
	if err == nil {
		t.Error("ValidateSeed() 'ab' should error")
	}
}

func TestValidateSeed_TooLong(t *testing.T) {
	longSeed := ""
	for i := 0; i < MaxSeedLength+1; i++ {
		longSeed += "a"
	}
	err := ValidateSeed(longSeed)
	if err == nil {
		t.Error("ValidateSeed() too long should error")
	}
}

func TestValidateParticipants_Duplicate(t *testing.T) {
	err := ValidateParticipants([]string{"Alice", "Bob", "Alice"})
	if err == nil {
		t.Error("ValidateParticipants() duplicate should error")
	}
}

func TestValidateParticipants_TooMany(t *testing.T) {
	participants := make([]string, MaxParticipants+1)
	for i := range participants {
		participants[i] = "Participant"
	}
	err := ValidateParticipants(participants)
	if err == nil {
		t.Error("ValidateParticipants() too many should error")
	}
}

func TestValidateWinnerCount_Negative(t *testing.T) {
	err := ValidateWinnerCount(-1, 10)
	if err == nil {
		t.Error("ValidateWinnerCount() negative should error")
	}
}

func TestValidateWinnerCount_TooMany(t *testing.T) {
	err := ValidateWinnerCount(MaxWinners+1, 1000)
	if err == nil {
		t.Error("ValidateWinnerCount() > MaxWinners should error")
	}
}

func TestValidateWinnerCount_ExceedsParticipants(t *testing.T) {
	err := ValidateWinnerCount(10, 5)
	if err == nil {
		t.Error("ValidateWinnerCount() > participants should error")
	}
}

func TestValidateWinnerCount_Zero(t *testing.T) {
	err := ValidateWinnerCount(0, 10)
	if err == nil {
		t.Error("ValidateWinnerCount() zero should error")
	}
}

func TestLotteryRecord_Validate(t *testing.T) {
	tests := []struct {
		name    string
		record  *LotteryRecord
		wantErr bool
	}{
		{
			name: "valid record with seed and participants",
			record: &LotteryRecord{
				ID:           "test-id",
				Seed:         "test-seed",
				Participants: []string{"Alice", "Bob", "Charlie"},
				Winners:      []string{"Alice"},
			},
			wantErr: false,
		},
		{
			name: "empty seed",
			record: &LotteryRecord{
				ID:           "test-id",
				Seed:         "",
				Participants: []string{"Alice", "Bob"},
				Winners:      []string{"Alice"},
			},
			wantErr: true,
		},
		{
			name: "seed too short",
			record: &LotteryRecord{
				ID:           "test-id",
				Seed:         "ab",
				Participants: []string{"Alice", "Bob"},
				Winners:      []string{"Alice"},
			},
			wantErr: true,
		},
		{
			name: "winners more than participants",
			record: &LotteryRecord{
				ID:           "test-id",
				Seed:         "test-seed",
				Participants: []string{"Alice"},
				Winners:      []string{"Alice", "Bob"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLotteryRecord_Getters(t *testing.T) {
	record := &LotteryRecord{
		ID:           "test-id",
		Seed:         "seed",
		Participants: []string{"A", "B", "C"},
		Winners:      []string{"A"},
	}

	if record.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", record.ID)
	}
	if len(record.Participants) != 3 {
		t.Errorf("Participants length = %v, want 3", len(record.Participants))
	}
	if len(record.Winners) != 1 {
		t.Errorf("Winners length = %v, want 1", len(record.Winners))
	}
}
