package voting

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

type Service interface {
	SignVote(message string, privateKey []byte) (string, error)
	VerifyVote(voterPK, message, signature string) bool
	CountVotes(candidates []Candidate) map[string]int
}

type Ed25519Service struct{}

func NewEd25519Service() *Ed25519Service {
	return &Ed25519Service{}
}

func (s *Ed25519Service) SignVote(message string, privateKey []byte) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size")
	}

	signature := ed25519.Sign(privateKey, []byte(message))
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *Ed25519Service) VerifyVote(voterPK, message, signature string) bool {
	pubBytes, err := base64.StdEncoding.DecodeString(voterPK)
	if err != nil {
		return false
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	return ed25519.Verify(pubBytes, []byte(message), sigBytes)
}

func (s *Ed25519Service) CountVotes(candidates []Candidate) map[string]int {
	results := make(map[string]int)
	for _, c := range candidates {
		results[c.ID] = c.VoteCount
	}
	return results
}
