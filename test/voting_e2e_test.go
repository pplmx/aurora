package test

import (
	"encoding/base64"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/voting"
)

func TestVotingE2E_FullFlow(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)
	chain := blockchain.InitBlockChain()

	c1, err := voting.RegisterCandidate("张三", "党A", "纲领A")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := voting.RegisterCandidate("李四", "党B", "纲领B")
	if err != nil {
		t.Fatal(err)
	}

	v1Pub, v1Priv, err := voting.RegisterVoter("投票人1")
	if err != nil {
		t.Fatal(err)
	}
	v2Pub, v2Priv, err := voting.RegisterVoter("投票人2")
	if err != nil {
		t.Fatal(err)
	}

	session, err := voting.CreateSession("测试选举", "描述", []string{c1.ID, c2.ID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := voting.StartSession(session.ID); err != nil {
		t.Fatal(err)
	}

	_, err = voting.CastVote(v1Pub, c1.ID, v1Priv, chain)
	if err != nil {
		t.Fatal(err)
	}

	_, err = voting.CastVote(v2Pub, c2.ID, v2Priv, chain)
	if err != nil {
		t.Fatal(err)
	}

	if err := voting.EndSession(session.ID); err != nil {
		t.Fatal(err)
	}

	results, err := voting.GetSessionResults(session.ID)
	if err != nil {
		t.Fatal(err)
	}

	if results[c1.ID] != 1 {
		t.Errorf("c1 votes = %v, want 1", results[c1.ID])
	}
	if results[c2.ID] != 1 {
		t.Errorf("c2 votes = %v, want 1", results[c2.ID])
	}

	t.Log("E2E test passed!")
}

func TestVotingE2E_MultipleVoters(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)
	chain := blockchain.InitBlockChain()

	c1, err := voting.RegisterCandidate("候选人A", "党派A", "纲领A")
	if err != nil {
		t.Fatal(err)
	}

	var voters []struct {
		pub  []byte
		priv []byte
	}
	for i := 0; i < 5; i++ {
		pub, priv, err := voting.RegisterVoter(string(rune('A' + i)))
		if err != nil {
			t.Fatal(err)
		}
		voters = append(voters, struct {
			pub  []byte
			priv []byte
		}{pub, priv})
	}

	session, err := voting.CreateSession("多人投票测试", "测试", []string{c1.ID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := voting.StartSession(session.ID); err != nil {
		t.Fatal(err)
	}

	for _, v := range voters {
		_, err := voting.CastVote(v.pub, c1.ID, v.priv, chain)
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := voting.EndSession(session.ID); err != nil {
		t.Fatal(err)
	}

	results, err := voting.GetSessionResults(session.ID)
	if err != nil {
		t.Fatal(err)
	}

	if results[c1.ID] != 5 {
		t.Errorf("c1 votes = %v, want 5", results[c1.ID])
	}
}

func TestVotingE2E_VerifySignature(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)
	chain := blockchain.InitBlockChain()

	c1, err := voting.RegisterCandidate("候选人X", "党派X", "纲领X")
	if err != nil {
		t.Fatal(err)
	}

	vPub, vPriv, err := voting.RegisterVoter("验证者")
	if err != nil {
		t.Fatal(err)
	}

	session, err := voting.CreateSession("签名验证测试", "测试", []string{c1.ID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := voting.StartSession(session.ID); err != nil {
		t.Fatal(err)
	}

	record, err := voting.CastVote(vPub, c1.ID, vPriv, chain)
	if err != nil {
		t.Fatal(err)
	}

	if !voting.VerifyVoteRecord(record) {
		t.Error("Vote record signature verification failed")
	}

	votes, err := voting.GetVotesByCandidate(c1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(votes) != 1 {
		t.Errorf("Expected 1 vote, got %d", len(votes))
	}

	if !voting.VerifyVoteRecord(votes[0]) {
		t.Error("Retrieved vote record signature verification failed")
	}
}

func TestVotingE2E_DuplicateVotePrevention(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)
	chain := blockchain.InitBlockChain()

	c1, err := voting.RegisterCandidate("候选人Y", "党派Y", "纲领Y")
	if err != nil {
		t.Fatal(err)
	}

	vPub, vPriv, err := voting.RegisterVoter("重复投票测试")
	if err != nil {
		t.Fatal(err)
	}

	session, err := voting.CreateSession("防重复测试", "测试", []string{c1.ID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := voting.StartSession(session.ID); err != nil {
		t.Fatal(err)
	}

	_, err = voting.CastVote(vPub, c1.ID, vPriv, chain)
	if err != nil {
		t.Fatal(err)
	}

	_, err = voting.CastVote(vPub, c1.ID, vPriv, chain)
	if err == nil {
		t.Error("Expected error for duplicate vote, got nil")
	}

	canVote, err := voting.CanVote(base64.StdEncoding.EncodeToString(vPub))
	if err != nil {
		t.Fatal(err)
	}
	if canVote {
		t.Error("Voter should not be able to vote twice")
	}
}

func TestVotingE2E_CandidateManagement(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)

	c1, err := voting.RegisterCandidate("候选人1", "党派1", "纲领1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := voting.RegisterCandidate("候选人2", "党派2", "纲领2")
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := voting.ListCandidates()
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(candidates))
	}

	gotC1, err := voting.GetCandidate(c1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotC1.Name != "候选人1" {
		t.Errorf("Expected candidate name '候选人1', got '%s'", gotC1.Name)
	}

	if err := voting.DeleteCandidate(c2.ID); err != nil {
		t.Fatal(err)
	}

	candidates, err = voting.ListCandidates()
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Errorf("Expected 1 candidate after delete, got %d", len(candidates))
	}
}

func TestVotingE2E_SessionLifecycle(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)

	c1, err := voting.RegisterCandidate("候选人Z", "党派Z", "纲领Z")
	if err != nil {
		t.Fatal(err)
	}

	session, err := voting.CreateSession("生命周期测试", "测试描述", []string{c1.ID}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != "draft" {
		t.Errorf("Initial status should be 'draft', got '%s'", session.Status)
	}

	sess, err := voting.GetSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != "draft" {
		t.Error("Session should be in draft status")
	}

	if err := voting.StartSession(session.ID); err != nil {
		t.Fatal(err)
	}

	sess, err = voting.GetSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != "active" {
		t.Errorf("Status should be 'active', got '%s'", sess.Status)
	}

	if err := voting.EndSession(session.ID); err != nil {
		t.Fatal(err)
	}

	sess, err = voting.GetSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != "ended" {
		t.Errorf("Status should be 'ended', got '%s'", sess.Status)
	}

	sessions, err := voting.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestVotingE2E_VoterManagement(t *testing.T) {
	storage := voting.NewInMemoryStorage()
	voting.InitVoting(storage)

	vPub, _, err := voting.RegisterVoter("选民A")
	if err != nil {
		t.Fatal(err)
	}

	voters, err := voting.ListVoters()
	if err != nil {
		t.Fatal(err)
	}
	if len(voters) != 1 {
		t.Errorf("Expected 1 voter, got %d", len(voters))
	}

	pkStr := base64.StdEncoding.EncodeToString(vPub)
	voter, err := voting.GetVoter(pkStr)
	if err != nil {
		t.Fatal(err)
	}
	if voter.Name != "选民A" {
		t.Errorf("Expected voter name '选民A', got '%s'", voter.Name)
	}

	canVote, err := voting.CanVote(pkStr)
	if err != nil {
		t.Fatal(err)
	}
	if !canVote {
		t.Error("New voter should be able to vote")
	}
}
