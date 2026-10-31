// Reading a session's tally: the counting rule lives here, apart from the
// session lifecycle that produces the window and the ballot path that fills it.
package votingapp

import (
	"errors"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
)

// GetResultsUseCase returns the per-candidate tally for a session from its
// declared candidate roster, plus the total vote count (v1.23 Voting Results
// API). Votes are counted from the session's claim-winning ballots, not from
// the candidate's global lifetime vote_count: a candidate reused in another
// election must not carry those votes into this result (ISS-299).
type GetResultsUseCase struct {
	repo voting.Repository
}

func NewGetResultsUseCase(repo voting.Repository) *GetResultsUseCase {
	return &GetResultsUseCase{repo: repo}
}

func (uc *GetResultsUseCase) Execute(sessionID string) (*ResultsResponse, error) {
	session, err := uc.repo.GetSession(sessionID)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return nil, voting.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, voting.ErrSessionNotFound
	}

	candidates := make([]CandidateResult, 0, len(session.Candidates))
	total := 0
	for _, cid := range session.Candidates {
		c, err := uc.repo.GetCandidate(cid)
		if err != nil {
			if errors.Is(err, repoerr.ErrNotFound) {
				// A candidate that has since been deleted still counts as 0
				// (same as the c == nil branch for in-memory fakes below).
				candidates = append(candidates, CandidateResult{ID: cid})
				continue
			}
			// A REAL read failure must abort the report: swallowing it here
			// silently under-counted the candidate and the total (L3, ISS-168).
			return nil, fmt.Errorf("failed to read candidate %s: %w", cid, err)
		}
		if c == nil {
			// A candidate that has since been deleted still counts as 0.
			candidates = append(candidates, CandidateResult{ID: cid})
			continue
		}
		voteCount, err := uc.repo.CountVotesBySession(session.ID, cid)
		if err != nil {
			return nil, fmt.Errorf("failed to count votes for candidate %s: %w", cid, err)
		}
		candidates = append(candidates, CandidateResult{
			ID:        c.ID,
			Name:      c.Name,
			Party:     c.Party,
			VoteCount: voteCount,
		})
		total += voteCount
	}

	return &ResultsResponse{
		SessionID:  session.ID,
		TotalVotes: total,
		Candidates: candidates,
	}, nil
}

// containsString reports whether list contains target (small lists; linear scan).
func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
