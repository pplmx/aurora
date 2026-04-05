package lottery

import (
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

type CreateLotteryUseCase struct {
	lotteryRepo lottery.Repository
	blockRepo   interface {
		AddLotteryRecord(data string) (int64, error)
	}
	service lottery.Service
}

func NewCreateLotteryUseCase(
	lotteryRepo lottery.Repository,
	blockRepo interface {
		AddLotteryRecord(data string) (int64, error)
	},
) *CreateLotteryUseCase {
	return &CreateLotteryUseCase{
		lotteryRepo: lotteryRepo,
		blockRepo:   blockRepo,
		service:     lottery.NewService(),
	}
}

func (uc *CreateLotteryUseCase) Execute(req CreateLotteryRequest) (*LotteryResponse, error) {
	participants := strings.Split(req.Participants, ",")
	for i := range participants {
		participants[i] = lottery.SanitizeString(participants[i])
	}
	participants = removeEmpty(participants)

	if err := lottery.ValidateParticipants(participants); err != nil {
		return nil, fmt.Errorf("invalid participants: %w", err)
	}

	seed := lottery.SanitizeString(req.Seed)
	if err := lottery.ValidateSeed(seed); err != nil {
		return nil, fmt.Errorf("invalid seed: %w", err)
	}

	if err := lottery.ValidateWinnerCount(req.WinnerCount, len(participants)); err != nil {
		return nil, fmt.Errorf("invalid winner count: %w", err)
	}

	winners, winnerAddrs, output, proof, err := uc.service.DrawWinners(participants, seed, req.WinnerCount)
	if err != nil {
		return nil, fmt.Errorf("failed to draw winners: %w", err)
	}

	record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

	jsonData, err := record.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize record: %w", err)
	}

	height, err := uc.blockRepo.AddLotteryRecord(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to add to blockchain: %w", err)
	}

	record.BlockHeight = height

	if err := uc.lotteryRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save to repository: %w", err)
	}

	return &LotteryResponse{
		ID:              record.ID,
		BlockHeight:     record.BlockHeight,
		Seed:            record.Seed,
		Participants:    record.Participants,
		Winners:         record.Winners,
		WinnerAddresses: record.WinnerAddresses,
		VRFProof:        record.VRFProof,
		VRFOutput:       record.VRFOutput,
		Timestamp:       record.Timestamp,
		Verified:        record.Verified,
	}, nil
}

func removeEmpty(s []string) []string {
	result := make([]string, 0, len(s))
	for _, str := range s {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}
