package lotteryapp

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
)

type transactionalLotteryRepository interface {
	lottery.Repository
	WithTx(tx *sql.Tx) lottery.Repository
	WithTransaction(fn func(tx *sql.Tx) error) error
}

type Create struct {
	lotteryRepo lottery.Repository
	blockRepo   interface {
		AddLotteryRecord(data string) (int64, error)
	}
	service lottery.Service
}

func NewCreate(
	lotteryRepo lottery.Repository,
	blockRepo interface {
		AddLotteryRecord(data string) (int64, error)
	},
) *Create {
	return &Create{
		lotteryRepo: lotteryRepo,
		blockRepo:   blockRepo,
		service:     lottery.NewService(),
	}
}

func (uc *Create) saveWithBlock(jsonData string, record *lottery.LotteryRecord) error {
	txRepo, repoOK := uc.lotteryRepo.(transactionalLotteryRepository)
	writer, chainOK := uc.blockRepo.(blockchain.LotteryRecordTransactionWriter)
	if repoOK && chainOK {
		_, err := blockchain.AppendRecordWithHeight(writer, txRepo, jsonData, func(tx *sql.Tx, height int64) error {
			record.BlockHeight = height
			if err := txRepo.WithTx(tx).Save(record); err != nil {
				return fmt.Errorf("failed to save to repository: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to persist lottery draw: %w", err)
		}
		return nil
	}

	height, err := uc.blockRepo.AddLotteryRecord(jsonData)
	if err != nil {
		return fmt.Errorf("failed to add to blockchain: %w", err)
	}
	record.BlockHeight = height
	if err := uc.lotteryRepo.Save(record); err != nil {
		return fmt.Errorf("failed to save to repository: %w", err)
	}
	return nil
}

func (uc *Create) Execute(req CreateLotteryRequest) (*LotteryResponse, error) {
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

	// Mark the draw verified at creation time: a draw is only trustworthy if
	// its recorded proof verifies against the persisted public key (binding
	// it to the seed) AND the recorded winners are exactly what the
	// deterministic selection produces from the VRF output. This makes the
	// audit surface honest (previously Verified stayed permanently false).
	// The verification runs BEFORE the on-chain JSON is serialized so the
	// immutable chain copy carries the same verified state the DB, REST and
	// CLI report — otherwise the chain record always said verified:false
	// while every other surface said true (ISS-256).
	if pk, derr := lottery.DecodePublicKey(publicKey); derr == nil {
		if ok, _ := uc.service.VerifyDraw(record, pk); ok {
			expected := lottery.SelectWinners(output, participants, req.WinnerCount)
			if sameStringSet(expected, winners) {
				record.Verified = true
			}
		}
	}

	jsonData, err := record.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize record: %w", err)
	}

	if err := uc.saveWithBlock(jsonData, record); err != nil {
		return nil, err
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

// Verify re-verifies a persisted draw through VerifyRecord, the
// same verifier used by the CLI, desktop shell and importer. Keeping one
// implementation prevents REST from accepting evidence another surface rejects.
type Verify struct {
	repo lottery.Repository
}

// NewVerify builds the re-verification use case.
func NewVerify(repo lottery.Repository) *Verify {
	return &Verify{repo: repo}
}

// Execute re-verifies the draw identified by req.ID.
func (uc *Verify) Execute(req VerifyLotteryRequest) (*VerifyLotteryResponse, error) {
	record, err := uc.repo.GetByID(req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lottery: %w", err)
	}
	if record == nil {
		return nil, lottery.ErrNotFound
	}

	report := VerifyRecord(record)
	if report.IntegrityOK {
		return &VerifyLotteryResponse{ID: record.ID, Valid: true}, nil
	}
	for _, check := range report.Checks {
		if check.Status == VerifyFail {
			return &VerifyLotteryResponse{ID: record.ID, Valid: false, Reason: check.Message}, nil
		}
	}
	return &VerifyLotteryResponse{
		ID: record.ID, Valid: false, Reason: "lottery record failed integrity check",
	}, nil
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
