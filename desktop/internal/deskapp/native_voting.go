package deskapp

import (
	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/votingapp"
)

// --- Voting --------------------------------------------------------------

func (s *nativeServices) RegisterVoter(name string) (nativeui.VoterRow, error) {
	uc := votingapp.NewRegisterVoterUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.RegisterVoterRequest{Name: name})
	if err != nil {
		return nativeui.VoterRow{}, err
	}
	return nativeui.VoterRow{PublicKey: resp.PublicKey, Name: resp.Name, PrivateKey: resp.PrivateKey}, nil
}

func (s *nativeServices) RegisterCandidate(name, party, program string) (nativeui.CandidateRow, error) {
	uc := votingapp.NewRegisterCandidateUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.RegisterCandidateRequest{Name: name, Party: party, Program: program})
	if err != nil {
		return nativeui.CandidateRow{}, err
	}
	return nativeui.CandidateRow{ID: resp.ID, Name: resp.Name, Party: resp.Party, Program: resp.Program}, nil
}

func (s *nativeServices) CreateSession(title, description string, candidateIDs []string, start, end int64) (nativeui.SessionRow, error) {
	uc := votingapp.NewCreateSessionUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.CreateSessionRequest{
		Title: title, Description: description, CandidateIDs: candidateIDs,
		StartTime: start, EndTime: end,
	})
	if err != nil {
		return nativeui.SessionRow{}, err
	}
	return nativeui.SessionRow{ID: resp.ID, Title: resp.Title, Status: resp.Status}, nil
}

func (s *nativeServices) CastVote(sessionID, candidateID, voterPub, voterPriv string) error {
	uc := votingapp.NewCastVoteUseCase(s.votingRepo, s.votingSvc, s.txManager)
	_, err := uc.Execute(votingapp.CastVoteRequest{
		VoterPublicKey: voterPub, CandidateID: candidateID,
		SessionID: sessionID, PrivateKey: voterPriv,
	})
	return err
}

func (s *nativeServices) SessionResults(sessionID string) ([]nativeui.ResultRow, error) {
	uc := votingapp.NewGetResultsUseCase(s.votingRepo)
	resp, err := uc.Execute(sessionID)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.ResultRow, 0, len(resp.Candidates))
	for _, r := range resp.Candidates {
		rows = append(rows, nativeui.ResultRow{CandidateID: r.ID, Name: r.Name, Votes: r.VoteCount})
	}
	return rows, nil
}

func (s *nativeServices) Candidates() ([]nativeui.CandidateRow, error) {
	uc := votingapp.NewGetCandidatesUseCase(s.votingRepo)
	resp, err := uc.Execute()
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.CandidateRow, 0, len(resp))
	for _, c := range resp {
		rows = append(rows, nativeui.CandidateRow{ID: c.ID, Name: c.Name, Party: c.Party, Program: c.Program, VoteCount: c.VoteCount})
	}
	return rows, nil
}

func (s *nativeServices) Sessions() ([]nativeui.SessionRow, error) {
	uc := votingapp.NewListSessionsUseCase(s.votingRepo)
	resp, err := uc.Execute()
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.SessionRow, 0, len(resp))
	for _, sess := range resp {
		rows = append(rows, sessionRow(sess))
	}
	return rows, nil
}

// sessionRow maps a session for the desktop. Description is part of what the
// Create Session form collected and what the API returns; dropping it left the
// window unable to show why a session exists.
func sessionRow(sess *votingapp.SessionResponse) nativeui.SessionRow {
	return nativeui.SessionRow{
		ID: sess.ID, Title: sess.Title, Description: sess.Description,
		Status: sess.Status, Candidates: sess.Candidates,
		StartTime: sess.StartTime, EndTime: sess.EndTime,
	}
}

func (s *nativeServices) StartSession(id string) error {
	return s.setSessionStatus(id, "active")
}

func (s *nativeServices) EndSession(id string) error {
	return s.setSessionStatus(id, "ended")
}

// setSessionStatus mirrors the REST handler's two steps: read the session so a
// missing id is a not-found rather than a silent no-op, then persist the new
// status.
func (s *nativeServices) setSessionStatus(id, status string) error {
	session, err := votingapp.NewGetSessionUseCase(s.votingRepo).Execute(id)
	if err != nil {
		return err
	}
	session.Status = status
	return s.votingRepo.UpdateSession(session)
}

// GenerateKeyPair makes the Ed25519 key pair a token owner or NFT creator
// needs. The CLI generates one when the flag is omitted; without this the
// window could not create a token at all without an external key generator.
