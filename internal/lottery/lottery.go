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
	// Only allow alphanumeric, Chinese characters, and common symbols
	validNameRegex = regexp.MustCompile(`^[\p{L}\p{N}\s\-_]+$`)
)

type LotteryRecord struct {
	ID              string   `json:"id"`
	Seed            string   `json:"seed"`
	Participants    []string `json:"participants"`
	Winners         []string `json:"winners"`
	WinnerAddresses []string `json:"winner_addresses"`
	VRFProof        string   `json:"vrf_proof"`
	VRFOutput       string   `json:"vrf_output"`
	BlockHeight     int64    `json:"block_height"`
	Timestamp       int64    `json:"timestamp"`
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
	idHash := sha256.Sum256([]byte(seed))
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

func (r *LotteryRecord) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *LotteryRecord) GetID() string {
	return r.ID
}

func ValidateParticipantName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("participant name cannot be empty")
	}
	if len(name) > MaxParticipantNameLength {
		return fmt.Errorf("participant name too long (max %d chars)", MaxParticipantNameLength)
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("participant name contains invalid characters")
	}
	return nil
}

func ValidateSeed(seed string) error {
	seed = strings.TrimSpace(seed)
	if len(seed) < MinSeedLength {
		return fmt.Errorf("seed too short (min %d chars)", MinSeedLength)
	}
	if len(seed) > MaxSeedLength {
		return fmt.Errorf("seed too long (max %d chars)", MaxSeedLength)
	}
	return nil
}

func ValidateParticipants(participants []string) error {
	if len(participants) == 0 {
		return fmt.Errorf("at least one participant required")
	}
	if len(participants) > MaxParticipants {
		return fmt.Errorf("too many participants (max %d)", MaxParticipants)
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, p := range participants {
		p = strings.TrimSpace(p)
		if err := ValidateParticipantName(p); err != nil {
			return fmt.Errorf("invalid participant: %w", err)
		}
		if seen[p] {
			return fmt.Errorf("duplicate participant: %s", p)
		}
		seen[p] = true
	}
	return nil
}

func ValidateWinnerCount(count, participantCount int) error {
	if count <= 0 {
		return fmt.Errorf("winner count must be positive")
	}
	if count > MaxWinners {
		return fmt.Errorf("too many winners (max %d)", MaxWinners)
	}
	if count > participantCount {
		return fmt.Errorf("winner count (%d) cannot exceed participants (%d)", count, participantCount)
	}
	return nil
}

func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove control characters
	s = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(s, "")
	return s
}
