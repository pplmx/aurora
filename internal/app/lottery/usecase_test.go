package lottery

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

type mockLotteryRepo struct {
	records []*lottery.LotteryRecord
}

func (m *mockLotteryRepo) Save(record *lottery.LotteryRecord) error {
	m.records = append(m.records, record)
	return nil
}

func (m *mockLotteryRepo) GetByID(id string) (*lottery.LotteryRecord, error) {
	for _, r := range m.records {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	return m.records, nil
}

func (m *mockLotteryRepo) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	var result []*lottery.LotteryRecord
	for _, r := range m.records {
		if r.BlockHeight == height {
			result = append(result, r)
		}
	}
	return result, nil
}

type mockBlockChain struct {
	blocks []string
	height int64
}

func (m *mockBlockChain) AddLotteryRecord(data string) (int64, error) {
	m.height++
	m.blocks = append(m.blocks, data)
	return m.height, nil
}

func TestCreateLotteryUseCase_Execute(t *testing.T) {
	lotteryRepo := &mockLotteryRepo{}
	blockChain := &mockBlockChain{}

	uc := NewCreateLotteryUseCase(lotteryRepo, blockChain)

	req := CreateLotteryRequest{
		Participants: "Alice,Bob,Charlie",
		Seed:         "test-seed",
		WinnerCount:  2,
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if len(resp.Winners) != 2 {
		t.Errorf("Expected 2 winners, got %d", len(resp.Winners))
	}

	if blockChain.height != 1 {
		t.Errorf("Expected 1 block added, got %d", blockChain.height)
	}
}

func TestCreateLotteryUseCase_InvalidInput(t *testing.T) {
	lotteryRepo := &mockLotteryRepo{}
	blockChain := &mockBlockChain{}

	uc := NewCreateLotteryUseCase(lotteryRepo, blockChain)

	tests := []struct {
		name    string
		req     CreateLotteryRequest
		wantErr bool
	}{
		{
			name: "empty participants",
			req: CreateLotteryRequest{
				Participants: "",
				Seed:         "seed",
				WinnerCount:  1,
			},
			wantErr: true,
		},
		{
			name: "empty seed",
			req: CreateLotteryRequest{
				Participants: "Alice,Bob",
				Seed:         "",
				WinnerCount:  1,
			},
			wantErr: true,
		},
		{
			name: "zero winners",
			req: CreateLotteryRequest{
				Participants: "Alice,Bob",
				Seed:         "seed",
				WinnerCount:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
