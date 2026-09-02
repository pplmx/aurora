package lottery

import (
	"encoding/hex"
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

	if req.WinnerCount == 0 {
		// REST/CLI parity: an omitted winner_count (the JSON zero value, so a
		// client posting the doc's minimal body) behaves like the CLI's
		// omitted `-c`, which falls back to the configured lottery.defaultCount
		// (default 3) — instead of a confusing WINNER_COUNT_NOT_POSITIVE 400.
		// Mirrors the add-source use case defaulting interval 0→60.
		req.WinnerCount = lottery.DefaultWinnerCount
	}

	if err := lottery.ValidateWinnerCount(req.WinnerCount, len(participants)); err != nil {
		return nil, fmt.Errorf("invalid winner count: %w", err)
	}

	winners, winnerAddrs, output, proof, publicKey, err := uc.service.DrawWinners(participants, seed, req.WinnerCount)
	if err != nil {
		return nil, fmt.Errorf("failed to draw winners: %w", err)
	}

	record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, publicKey, 0)

	jsonData, err := record.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize record: %w", err)
	}

	height, err := uc.blockRepo.AddLotteryRecord(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to add to blockchain: %w", err)
	}

	record.BlockHeight = height

	// Mark the draw verified at creation time: a draw is only trustworthy if
	// its recorded proof verifies against the persisted public key (binding
	// it to the seed) AND the recorded winners are exactly what the
	// deterministic selection produces from the VRF output. This makes the
	// audit surface honest (previously Verified stayed permanently false).
	if pk, derr := lottery.DecodePublicKey(publicKey); derr == nil {
		if ok, _ := uc.service.VerifyDraw(record, pk); ok {
			expected := lottery.SelectWinners(output, participants, req.WinnerCount)
			if sameStringSet(expected, winners) {
				record.Verified = true
			}
		}
	}

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

// VerifyLotteryRequest carries the id of the draw to re-verify.
type VerifyLotteryRequest struct {
	ID string
}

// VerifyLotteryResponse reports whether a stored draw re-verifies.
type VerifyLotteryResponse struct {
	ID     string `json:"id"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
}

// VerifyLotteryUseCase re-verifies a persisted draw: it loads the record, and
// when the stored VRF public key is present it re-runs the same proof/key
// verification and winner-set check that creation performed. Records created
// before this feature (no public key) fall back to the deterministic
// winner-set check so they do not hard-fail.
type VerifyLotteryUseCase struct {
	repo    lottery.Repository
	service lottery.Service
}

// NewVerifyLotteryUseCase builds the re-verification use case.
func NewVerifyLotteryUseCase(repo lottery.Repository, service lottery.Service) *VerifyLotteryUseCase {
	return &VerifyLotteryUseCase{repo: repo, service: service}
}

// Execute re-verifies the draw identified by req.ID.
func (uc *VerifyLotteryUseCase) Execute(req VerifyLotteryRequest) (*VerifyLotteryResponse, error) {
	record, err := uc.repo.GetByID(req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lottery: %w", err)
	}
	if record == nil {
		return nil, lottery.ErrNotFound
	}

	// Decode the VRF output for the deterministic winner-set check.
	vrfOutput, err := hex.DecodeString(record.VRFOutput)
	if err != nil {
		return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "vrf output is not valid hex"}, nil
	}
	if _, err := hex.DecodeString(record.VRFProof); err != nil {
		return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "vrf proof is not valid hex"}, nil
	}

	// Optional key-bound proof verification (draws created before this
	// feature have no stored key and are checked deterministically only).
	if record.VRFPublicKey != "" {
		pk, derr := lottery.DecodePublicKey(record.VRFPublicKey)
		if derr != nil {
			return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "stored public key is malformed"}, nil
		}
		ok, verr := uc.service.VerifyDraw(record, pk)
		if verr != nil || !ok {
			return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "VRF proof does not verify against the stored key"}, nil
		}
	}

	// A draw with no winners is corrupt by construction: CreateLotteryUseCase
	// (and the CLI/API/TUI paths behind it) always stores at least one winner
	// because ValidateWinnerCount rejects count<=0, so an empty winners slice
	// can only come from a partial/corrupt write or a hand-crafted record.
	// Without this guard, SelectWinners(output, roster, 0) returns an empty
	// set, sameStringSet([], []) is vacuously true, and the record reports
	// valid:true — masking the corruption (TASK-249, ISS-249).
	if len(record.Winners) == 0 {
		return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "record has no winners"}, nil
	}

	// The winners recorded must be exactly what SelectWinners produces from
	// the stored output for this roster and winner count.
	expected := lottery.SelectWinners(vrfOutput, record.Participants, len(record.Winners))
	if !sameStringSet(expected, record.Winners) {
		return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "stored winners do not match the VRF output"}, nil
	}
	if len(record.Winners) != len(record.WinnerAddresses) {
		return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: "winner/address slice length mismatch"}, nil
	}

	return &VerifyLotteryResponse{ID: record.ID, Valid: true}, nil
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[string]int, len(a))
	for _, s := range a {
		count[s]++
	}
	for _, s := range b {
		count[s]--
		if count[s] < 0 {
			return false
		}
	}
	return true
}
