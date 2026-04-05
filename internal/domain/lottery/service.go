package lottery

import (
	"encoding/hex"

	"filippo.io/edwards25519"
)

type Service interface {
	DrawWinners(participants []string, seed string, count int) ([]string, []string, []byte, []byte, error)
	VerifyDraw(record *LotteryRecord, publicKey *edwards25519.Point) (bool, error)
}

type lotteryService struct{}

func NewService() Service {
	return &lotteryService{}
}

func (s *lotteryService) DrawWinners(participants []string, seed string, count int) ([]string, []string, []byte, []byte, error) {
	if err := ValidateParticipants(participants); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := ValidateSeed(seed); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := ValidateWinnerCount(count, len(participants)); err != nil {
		return nil, nil, nil, nil, err
	}

	pk, sk, err := GenerateKeyPair()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	output, proof, err := VRFProve(sk, []byte(seed))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	winners := SelectWinners(output, participants, count)

	winnerAddrs := make([]string, len(winners))
	for i, w := range winners {
		winnerAddrs[i] = NameToAddress(w)
	}

	_ = pk
	return winners, winnerAddrs, output, proof, nil
}

func (s *lotteryService) VerifyDraw(record *LotteryRecord, publicKey *edwards25519.Point) (bool, error) {
	if record == nil {
		return false, nil
	}

	output, err := hex.DecodeString(record.VRFOutput)
	if err != nil {
		return false, err
	}

	proof, err := hex.DecodeString(record.VRFProof)
	if err != nil {
		return false, err
	}

	valid := VRFVerify(publicKey, []byte(record.Seed), output, proof)
	return valid, nil
}
