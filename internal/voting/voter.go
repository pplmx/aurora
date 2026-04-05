package voting

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Voter is a struct that represents a voter in the voting system
type Voter struct {
	PublicKey    string `json:"public_key"`
	Name         string `json:"name"`
	HasVoted     bool   `json:"has_voted"`
	VoteHash     string `json:"vote_hash"`
	RegisteredAt int64  `json:"registered_at"`
	Token        ed25519.PublicKey
	Candidate    string
	Signature    []byte
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
	if v.Token == nil || v.Signature == nil {
		return false
	}
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

var voterStorage Storage

func SetVoterStorage(s Storage) {
	voterStorage = s
}

func RegisterVoter(name string) (publicKey []byte, privateKey []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	voter := &Voter{
		PublicKey:    base64.StdEncoding.EncodeToString(pub),
		Name:         name,
		HasVoted:     false,
		RegisteredAt: time.Now().Unix(),
	}

	dbVoter := &DBVoter{
		PublicKey:    voter.PublicKey,
		Name:         voter.Name,
		HasVoted:     voter.HasVoted,
		VoteHash:     voter.VoteHash,
		RegisteredAt: voter.RegisteredAt,
	}

	if err := voterStorage.SaveVoter(dbVoter); err != nil {
		return nil, nil, err
	}

	return pub, priv, nil
}

func GetVoter(publicKey string) (*Voter, error) {
	dbVoter, err := voterStorage.GetVoter(publicKey)
	if err != nil {
		return nil, err
	}
	if dbVoter == nil {
		return nil, nil
	}
	return &Voter{
		PublicKey:    dbVoter.PublicKey,
		Name:         dbVoter.Name,
		HasVoted:     dbVoter.HasVoted,
		VoteHash:     dbVoter.VoteHash,
		RegisteredAt: dbVoter.RegisteredAt,
	}, nil
}

func ListVoters() ([]*Voter, error) {
	dbVoters, err := voterStorage.ListVoters()
	if err != nil {
		return nil, err
	}
	voters := make([]*Voter, len(dbVoters))
	for i, dbVoter := range dbVoters {
		voters[i] = &Voter{
			PublicKey:    dbVoter.PublicKey,
			Name:         dbVoter.Name,
			HasVoted:     dbVoter.HasVoted,
			VoteHash:     dbVoter.VoteHash,
			RegisteredAt: dbVoter.RegisteredAt,
		}
	}
	return voters, nil
}

func CanVote(publicKey string) (bool, error) {
	voter, err := voterStorage.GetVoter(publicKey)
	if err != nil {
		return false, err
	}
	if voter == nil {
		return false, nil
	}
	return !voter.HasVoted, nil
}

func MarkVoted(publicKey, voteHash string) error {
	voter, err := voterStorage.GetVoter(publicKey)
	if err != nil {
		return err
	}
	voter.HasVoted = true
	voter.VoteHash = voteHash
	return voterStorage.UpdateVoter(voter)
}
