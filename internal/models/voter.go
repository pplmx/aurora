package models

import (
	"crypto/ed25519"
)

// Voter is a struct that represents a voter in the voting system
type Voter struct {
	Token     ed25519.PublicKey // The token of the voter
	Candidate string            // The candidate that the voter voted for
	Signature []byte            // The signature of the vote
}

// GetToken is a method that returns the token of the voter
func (v *Voter) GetToken() ed25519.PublicKey {
	return v.Token
}

// GetCandidate is a method that returns the candidate that the voter voted for
func (v *Voter) GetCandidate() string {
	return v.Candidate
}

// GetSignature is a method that returns the signature of the vote
func (v *Voter) GetSignature() []byte {
	return v.Signature
}

// VerifyVote is a method that verifies if the vote is valid by checking the signature with the token
func (v *Voter) VerifyVote() bool {
	return ed25519.Verify(v.Token, []byte(v.Candidate), v.Signature)
}

// Register is a function that registers a new voter by generating and returning an ed25519 key pair
func Register() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}
	return publicKey, privateKey, nil
}
