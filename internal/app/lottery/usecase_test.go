package lottery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/stretchr/testify/require"
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

	require.NotNil(t, resp)

	if len(resp.Winners) != 2 {
		t.Errorf("Expected 2 winners, got %d", len(resp.Winners))
	}

	if blockChain.height != 1 {
		t.Errorf("Expected 1 block added, got %d", blockChain.height)
	}

	// v1.31: a freshly created draw must carry its VRF public key and be
	// marked Verified (its proof binds to the persisted key and its winners
	// match the deterministic selection), so the module's fairness claim is
	// re-checkable from the record alone.
	if resp.Verified != true {
		t.Error("Expected newly created draw to be Verified=true")
	}
	if len(lotteryRepo.records) != 1 || lotteryRepo.records[0].VRFPublicKey == "" {
		t.Error("Expected the stored record to persist the VRF public key")
	}

	// Regression: LotteryResponse previously had no json tags, so the
	// POST /api/v1/lottery/create payload was PascalCase, inconsistent with the
	// rest of the snake_case API. Lock the contract.
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	body := string(raw)
	for _, want := range []string{`"id"`, `"block_height"`, `"winner_addresses"`, `"vrf_proof"`, `"vrf_output"`} {
		if !strings.Contains(body, want) {
			t.Errorf("lottery create JSON missing %s (got %s)", want, body)
		}
	}
	if strings.Contains(body, `"ID"`) {
		t.Errorf("lottery create JSON should be snake_case, got PascalCase key: %s", body)
	}
}

func TestVerifyLotteryUseCase_ValidDraw(t *testing.T) {
	lotteryRepo := &mockLotteryRepo{}
	blockChain := &mockBlockChain{}
	if _, err := NewCreateLotteryUseCase(lotteryRepo, blockChain).Execute(CreateLotteryRequest{
		Participants: "Alice,Bob,Charlie",
		Seed:         "verify-seed",
		WinnerCount:  1,
	}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	id := lotteryRepo.records[0].ID

	resp, err := NewVerifyLotteryUseCase(lotteryRepo, lottery.NewService()).Execute(VerifyLotteryRequest{ID: id})
	require.NoError(t, err)
	require.True(t, resp.Valid)
	require.Equal(t, id, resp.ID)
}

func TestVerifyLotteryUseCase_TamperedWinners(t *testing.T) {
	lotteryRepo := &mockLotteryRepo{}
	blockChain := &mockBlockChain{}
	if _, err := NewCreateLotteryUseCase(lotteryRepo, blockChain).Execute(CreateLotteryRequest{
		Participants: "Alice,Bob,Charlie",
		Seed:         "verify-seed-2",
		WinnerCount:  1,
	}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// Mutate the stored winner so it no longer matches SelectWinners(output).
	record := lotteryRepo.records[0]
	other := firstDifferent(record.Participants, record.Winners[0])
	record.Winners[0] = other

	resp, err := NewVerifyLotteryUseCase(lotteryRepo, lottery.NewService()).Execute(VerifyLotteryRequest{ID: record.ID})
	require.NoError(t, err)
	require.False(t, resp.Valid)
}

func TestVerifyLotteryUseCase_NotFound(t *testing.T) {
	lotteryRepo := &mockLotteryRepo{}
	_, err := NewVerifyLotteryUseCase(lotteryRepo, lottery.NewService()).Execute(VerifyLotteryRequest{ID: "missing"})
	require.ErrorIs(t, err, lottery.ErrNotFound)
}

func firstDifferent(participants []string, current string) string {
	for _, p := range participants {
		if p != current {
			return p
		}
	}
	return current
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
