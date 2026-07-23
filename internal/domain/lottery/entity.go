// Package lottery provides VRF-based transparent lottery functionality.
// It implements verifiable random function (VRF) to ensure fair and
// transparent winner selection that can be verified on-chain.
package lottery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const (
	MaxParticipants          = 10000
	MaxWinners               = 100
	MinSeedLength            = 3
	MaxSeedLength            = 256
	MaxParticipantNameLength = 100
)

var (
	validNameRegex = regexp.MustCompile(`^[\p{L}\p{N}\s\-_]+$`)
)

type LotteryRecord struct {
	ID              string   `json:"id"`
	BlockHeight     int64    `json:"block_height"`
	Seed            string   `json:"seed"`
	Participants    []string `json:"participants"`
	Winners         []string `json:"winners"`
	WinnerAddresses []string `json:"winner_addresses"`
	VRFProof        string   `json:"vrf_proof"`
	VRFOutput       string   `json:"vrf_output"`
	Timestamp       int64    `json:"timestamp"`
	Verified        bool     `json:"verified"`
}

func (r *LotteryRecord) Validate() error {
	if err := ValidateSeed(r.Seed); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	if err := ValidateParticipants(r.Participants); err != nil {
		return fmt.Errorf("participants: %w", err)
	}
	if err := ValidateWinnerCount(len(r.Winners), len(r.Participants)); err != nil {
		return fmt.Errorf("winners: %w", err)
	}
	return nil
}

func (r *LotteryRecord) GetWinners() []string {
	return r.Winners
}

func (r *LotteryRecord) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *LotteryRecord) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), r)
}

func SelectWinners(output []byte, participants []string, count int) []string {
	if len(participants) == 0 {
		return []string{}
	}

	if count >= len(participants) {
		result := make([]string, len(participants))
		copy(result, participants)
		return result
	}

	winners := make([]string, 0, count)
	used := make(map[int]bool)

	current := make([]byte, len(output))
	copy(current, output)

	for len(winners) < count && len(used) < len(participants) {
		num := new(big.Int).SetBytes(current)
		mod := big.NewInt(int64(len(participants)))
		idx := int(num.Mod(num, mod).Int64())

		if !used[idx] {
			used[idx] = true
			winners = append(winners, participants[idx])
		}

		hash := sha256.Sum256(current)
		current = hash[:]
	}

	return winners
}

func CreateLotteryRecord(
	seed string,
	participants []string,
	winners []string,
	winnerAddrs []string,
	output []byte,
	proof []byte,
	blockHeight int64,
) *LotteryRecord {
	// Build a record ID that is unique per draw. The VRF `output` is the
	// critical disambiguator: it is derived from a freshly-generated
	// keypair inside DrawWinners, so two records produced by different
	// draws (even with the same seed and participants) will have distinct
	// outputs and therefore distinct IDs.
	//
	// Why this matters: the repository uses INSERT OR REPLACE keyed on
	// the ID, so two records with the same ID would silently overwrite
	// one another — unacceptable for an audit trail.
	idSrc := make([]byte, 0, len(seed)+len(output))
	idSrc = append(idSrc, []byte(seed)...)
	idSrc = append(idSrc, output...)
	idHash := sha256.Sum256(idSrc)
	id := hex.EncodeToString(idHash[:])[:16]

	return &LotteryRecord{
		ID:              id,
		Seed:            seed,
		Participants:    participants,
		Winners:         winners,
		WinnerAddresses: winnerAddrs,
		VRFProof:        hex.EncodeToString(proof),
		VRFOutput:       hex.EncodeToString(output),
		BlockHeight:     blockHeight,
		Timestamp:       time.Now().Unix(),
	}
}

func ValidateParticipantName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyParticipantName
	}
	if len(name) > MaxParticipantNameLength {
		return ErrParticipantNameTooLong
	}
	if !validNameRegex.MatchString(name) {
		return ErrInvalidParticipantName
	}
	return nil
}

func ValidateSeed(seed string) error {
	seed = strings.TrimSpace(seed)
	if len(seed) < MinSeedLength {
		return ErrSeedTooShort
	}
	if len(seed) > MaxSeedLength {
		return ErrSeedTooLong
	}
	return nil
}

func ValidateParticipants(participants []string) error {
	if len(participants) == 0 {
		return ErrNoParticipants
	}
	if len(participants) > MaxParticipants {
		return ErrTooManyParticipants
	}

	seen := make(map[string]bool)
	for _, p := range participants {
		p = strings.TrimSpace(p)
		if err := ValidateParticipantName(p); err != nil {
			return fmt.Errorf("invalid participant: %w", err)
		}
		if seen[p] {
			return fmt.Errorf("duplicate participant: %w", ErrDuplicateParticipant)
		}
		seen[p] = true
	}
	return nil
}

func ValidateWinnerCount(count, participantCount int) error {
	if count <= 0 {
		return ErrWinnerCountNotPositive
	}
	if count > MaxWinners {
		return ErrTooManyWinners
	}
	if count > participantCount {
		return ErrWinnersExceedParticipants
	}
	return nil
}

func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(s, "")
	return s
}

func NameToAddress(name string) string {
	h := sha256.Sum256([]byte(name))
	return "0x" + hex.EncodeToString(h[:20])
}
