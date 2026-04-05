package voting

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
)

func TestCastVote(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{
		PublicKey:    voterPKStr,
		Name:         "Test Voter",
		HasVoted:     false,
		RegisteredAt: 0,
	}
	if err := voterStorage.SaveVoter(dbVoter); err != nil {
		t.Fatalf("failed to register voter: %v", err)
	}

	candidate, err := RegisterCandidate("Candidate A", "Party A", "Program A")
	if err != nil {
		t.Fatalf("failed to register candidate: %v", err)
	}

	record, err := CastVote(pub, candidate.ID, priv, chain)
	if err != nil {
		t.Fatalf("failed to cast vote: %v", err)
	}

	if record == nil {
		t.Fatal("vote record is nil")
	}

	if record.VoterPK != voterPKStr {
		t.Errorf("expected voter pk %s, got %s", voterPKStr, record.VoterPK)
	}

	if record.CandidateID != candidate.ID {
		t.Errorf("expected candidate id %s, got %s", candidate.ID, record.CandidateID)
	}

	if record.Signature == "" {
		t.Error("signature should not be empty")
	}

	if record.BlockHeight <= 0 {
		t.Errorf("block height should be > 0, got %d", record.BlockHeight)
	}

	t.Logf("Vote record: %+v", record)
}

func TestVerifyVoteRecord(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{
		PublicKey:    voterPKStr,
		Name:         "Test Voter",
		HasVoted:     false,
		RegisteredAt: 0,
	}
	if err := voterStorage.SaveVoter(dbVoter); err != nil {
		t.Fatalf("failed to register voter: %v", err)
	}

	candidate, err := RegisterCandidate("Candidate B", "Party B", "Program B")
	if err != nil {
		t.Fatalf("failed to register candidate: %v", err)
	}

	record, err := CastVote(pub, candidate.ID, priv, chain)
	if err != nil {
		t.Fatalf("failed to cast vote: %v", err)
	}

	if !VerifyVoteRecord(record) {
		t.Error("vote verification failed")
	}

	record.Signature = "invalid"
	if VerifyVoteRecord(record) {
		t.Error("vote with invalid signature should not verify")
	}
}

func TestGetVote(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{
		PublicKey:    voterPKStr,
		Name:         "Test Voter",
		HasVoted:     false,
		RegisteredAt: 0,
	}
	if err := voterStorage.SaveVoter(dbVoter); err != nil {
		t.Fatalf("failed to register voter: %v", err)
	}

	candidate, err := RegisterCandidate("Candidate C", "Party C", "Program C")
	if err != nil {
		t.Fatalf("failed to register candidate: %v", err)
	}

	record, err := CastVote(pub, candidate.ID, priv, chain)
	if err != nil {
		t.Fatalf("failed to cast vote: %v", err)
	}

	fetched, err := GetVote(record.ID)
	if err != nil {
		t.Fatalf("failed to get vote: %v", err)
	}

	if fetched.ID != record.ID {
		t.Errorf("expected id %s, got %s", record.ID, fetched.ID)
	}
}

func TestGetVotesByCandidate(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub1, priv1, _ := ed25519.GenerateKey(nil)
	pub2, priv2, _ := ed25519.GenerateKey(nil)

	voterPK1 := base64.StdEncoding.EncodeToString(pub1)
	voterPK2 := base64.StdEncoding.EncodeToString(pub2)

	dbVoter1 := &DBVoter{PublicKey: voterPK1, Name: "Voter 1", HasVoted: false, RegisteredAt: 0}
	dbVoter2 := &DBVoter{PublicKey: voterPK2, Name: "Voter 2", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter1)
	voterStorage.SaveVoter(dbVoter2)

	candidate, _ := RegisterCandidate("Candidate D", "Party D", "Program D")

	CastVote(pub1, candidate.ID, priv1, chain)
	CastVote(pub2, candidate.ID, priv2, chain)

	votes, err := GetVotesByCandidate(candidate.ID)
	if err != nil {
		t.Fatalf("failed to get votes: %v", err)
	}

	if len(votes) != 2 {
		t.Errorf("expected 2 votes, got %d", len(votes))
	}
}

func TestCountVotes(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub1, priv1, _ := ed25519.GenerateKey(nil)
	pub2, priv2, _ := ed25519.GenerateKey(nil)

	voterPK1 := base64.StdEncoding.EncodeToString(pub1)
	voterPK2 := base64.StdEncoding.EncodeToString(pub2)

	dbVoter1 := &DBVoter{PublicKey: voterPK1, Name: "Voter 1", HasVoted: false, RegisteredAt: 0}
	dbVoter2 := &DBVoter{PublicKey: voterPK2, Name: "Voter 2", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter1)
	voterStorage.SaveVoter(dbVoter2)

	candidate, _ := RegisterCandidate("Candidate E", "Party E", "Program E")

	CastVote(pub1, candidate.ID, priv1, chain)
	CastVote(pub2, candidate.ID, priv2, chain)

	count, err := CountVotes(candidate.ID)
	if err != nil {
		t.Fatalf("failed to count votes: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 votes, got %d", count)
	}
}

func TestVoteRecordToJSON(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, _ := ed25519.GenerateKey(nil)

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{PublicKey: voterPKStr, Name: "Test Voter", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter)

	candidate, _ := RegisterCandidate("Candidate F", "Party F", "Program F")

	record, _ := CastVote(pub, candidate.ID, priv, chain)

	jsonStr, err := record.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON string should not be empty")
	}

	t.Logf("JSON: %s", jsonStr)
}

func TestDuplicateVote(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, _ := ed25519.GenerateKey(nil)

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{PublicKey: voterPKStr, Name: "Test Voter", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter)

	candidate, _ := RegisterCandidate("Candidate G", "Party G", "Program G")

	_, err := CastVote(pub, candidate.ID, priv, chain)
	if err != nil {
		t.Fatalf("first vote failed: %v", err)
	}

	_, err = CastVote(pub, candidate.ID, priv, chain)
	if err == nil {
		t.Error("duplicate vote should fail")
	}

	if err.Error() != "already voted" {
		t.Errorf("expected 'already voted' error, got '%s'", err.Error())
	}
}

func TestVoteNotRegisteredVoter(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, _, _ := ed25519.GenerateKey(nil)
	_, priv, _ := ed25519.GenerateKey(nil)

	candidate, _ := RegisterCandidate("Candidate H", "Party H", "Program H")

	_, err := CastVote(pub, candidate.ID, priv, chain)
	if err == nil {
		t.Error("vote from non-registered voter should fail")
	}

	if err.Error() != "voter not registered" {
		t.Errorf("expected 'voter not registered' error, got '%s'", err.Error())
	}
}

func TestVoteNonExistentCandidate(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, _ := ed25519.GenerateKey(nil)

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{PublicKey: voterPKStr, Name: "Test Voter", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter)

	_, err := CastVote(pub, "non-existent-candidate", priv, chain)
	if err == nil {
		t.Error("vote for non-existent candidate should fail")
	}

	if err.Error() != "candidate not found" {
		t.Errorf("expected 'candidate not found' error, got '%s'", err.Error())
	}
}

func TestVoteWithSQLiteStorage(t *testing.T) {
	tmpFile := "/tmp/voting_test.db"
	os.Remove(tmpFile)

	storage, err := NewSQLiteStorage(tmpFile)
	if err != nil {
		t.Fatalf("failed to create SQLite storage: %v", err)
	}
	defer os.Remove(tmpFile)
	defer storage.Close()

	SetCandidateStorage(storage)
	SetVoterStorage(storage)
	SetVoteStorage(storage)

	chain := blockchain.InitBlockChain()

	pub, priv, _ := ed25519.GenerateKey(nil)

	voterPKStr := base64.StdEncoding.EncodeToString(pub)
	dbVoter := &DBVoter{PublicKey: voterPKStr, Name: "SQLite Voter", HasVoted: false, RegisteredAt: 0}
	voterStorage.SaveVoter(dbVoter)

	candidate, _ := RegisterCandidate("SQLite Candidate", "Party", "Program")

	record, err := CastVote(pub, candidate.ID, priv, chain)
	if err != nil {
		t.Fatalf("failed to cast vote: %v", err)
	}

	fetched, err := GetVote(record.ID)
	if err != nil {
		t.Fatalf("failed to get vote: %v", err)
	}

	if fetched.ID != record.ID {
		t.Errorf("expected id %s, got %s", record.ID, fetched.ID)
	}
}
