package lottery

import (
	"strings"
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

func TestEndToEndLottery(t *testing.T) {
	// 1. 准备参与者
	participants := []string{"张三", "李四", "王五", "赵六", "钱七"}
	seed := "e2e-test-seed"

	// 2. 生成 VRF 密钥对
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// 3. 生成 VRF 输出和证明
	output, proof, err := VRFProve(sk, []byte(seed))
	if err != nil {
		t.Fatalf("VRFProve failed: %v", err)
	}

	// 4. 验证 VRF (基本验证)
	if len(output) == 0 {
		t.Error("VRF output should not be empty")
	}
	if len(proof) == 0 {
		t.Error("VRF proof should not be empty")
	}

	// 5. 选择中奖者
	winners := SelectWinners(output, participants, 3)
	if len(winners) != 3 {
		t.Fatalf("Expected 3 winners, got %d", len(winners))
	}

	// 6. 转换地址
	for _, w := range winners {
		addr := NameToAddress(w)
		if len(addr) != 42 {
			t.Errorf("Invalid address length for %s: %d", w, len(addr))
		}
	}

	// 7. 创建记录
	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = NameToAddress(w)
	}
	record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

	// 8. 验证记录
	if record.Seed != seed {
		t.Errorf("Seed mismatch")
	}
	if len(record.Winners) != 3 {
		t.Errorf("Winners count mismatch")
	}
	if record.ID == "" {
		t.Error("ID should not be empty")
	}

	// 9. JSON 序列化
	jsonStr, err := record.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(jsonStr) == 0 {
		t.Error("JSON should not be empty")
	}

	_ = pk
}

func TestValidateParticipantName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"张三", false},
		{"John Doe", false},
		{"Alice-Bob", false},
		{"", true},
		{strings.Repeat("a", 101), true},
		{"test@name", true},
		{"test#name", true},
	}

	for _, tt := range tests {
		err := ValidateParticipantName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateParticipantName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestValidateSeed(t *testing.T) {
	tests := []struct {
		seed    string
		wantErr bool
	}{
		{"abc", false},
		{"test-seed-123", false},
		{"", true},
		{"ab", true},
		{strings.Repeat("a", 257), true},
	}

	for _, tt := range tests {
		err := ValidateSeed(tt.seed)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateSeed(%q) error = %v, wantErr %v", tt.seed, err, tt.wantErr)
		}
	}
}

func TestValidateParticipants(t *testing.T) {
	tests := []struct {
		participants []string
		wantErr      bool
	}{
		{[]string{"Alice", "Bob"}, false},
		{[]string{"张三", "李四"}, false},
		{[]string{}, true},
		{[]string{"Alice", "Bob", "Alice"}, true},
		{[]string{""}, true},
	}

	for _, tt := range tests {
		err := ValidateParticipants(tt.participants)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateParticipants(%v) error = %v, wantErr %v", tt.participants, err, tt.wantErr)
		}
	}
}

func TestValidateWinnerCount(t *testing.T) {
	tests := []struct {
		count            int
		participantCount int
		wantErr          bool
	}{
		{1, 10, false},
		{3, 10, false},
		{10, 10, false},
		{0, 10, true},
		{-1, 10, true},
		{101, 10, true},
		{11, 10, true},
	}

	for _, tt := range tests {
		err := ValidateWinnerCount(tt.count, tt.participantCount)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateWinnerCount(%d, %d) error = %v, wantErr %v", tt.count, tt.participantCount, err, tt.wantErr)
		}
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello\tworld", "helloworld"},
		{"hello\nworld", "helloworld"},
		{"normal text", "normal text"},
	}

	for _, tt := range tests {
		result := SanitizeString(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLotteryRecord_GetID(t *testing.T) {
	record := &LotteryRecord{ID: "test-id"}

	if record.GetID() != "test-id" {
		t.Errorf("GetID() = %v, want 'test-id'", record.GetID())
	}
}
