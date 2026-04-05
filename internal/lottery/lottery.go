package lottery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"
)

type LotteryRecord struct {
	ID           string   `json:"id"`
	Seed         string   `json:"seed"`
	Participants []string `json:"participants"`
	Winners      []string `json:"winners"`
	WinnerAddrs  []string `json:"winner_addrs"`
	VRFProof     string   `json:"vrf_proof"`
	VRFOutput    string   `json:"vrf_output"`
	BlockHeight  int64    `json:"block_height"`
	Timestamp    int64    `json:"timestamp"`
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
		ID:           id,
		Seed:         seed,
		Participants: participants,
		Winners:      winners,
		WinnerAddrs:  winnerAddrs,
		VRFProof:     hex.EncodeToString(proof),
		VRFOutput:    hex.EncodeToString(output),
		BlockHeight:  blockHeight,
		Timestamp:    time.Now().Unix(),
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
