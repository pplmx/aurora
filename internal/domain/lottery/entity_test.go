package lottery

import (
	"testing"
)

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
