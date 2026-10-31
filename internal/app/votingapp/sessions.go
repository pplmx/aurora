// Sessions and their tally: opening a governance window, listing and reading
// one, and counting the ballots it accepted.
package votingapp

import (
	"errors"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
)

type CreateSessionUseCase struct {
	repo voting.Repository
}

func NewCreateSessionUseCase(repo voting.Repository) *CreateSessionUseCase {
	return &CreateSessionUseCase{repo: repo}
}

func (uc *CreateSessionUseCase) Execute(req CreateSessionRequest) (*SessionResponse, error) {
	if err := voting.ValidateSessionTitle(req.Title); err != nil {
		return nil, err
	}
	if err := voting.ValidateSessionDescription(req.Description); err != nil {
		return nil, err
	}
	if len(req.CandidateIDs) == 0 {
		return nil, voting.ErrCandidatesRequired
	}
	if req.EndTime <= req.StartTime {
		return nil, voting.ErrInvalidSessionTime
	}
	// Reject duplicate candidate IDs before the existence scan: results sum each
	// roster candidate's full vote_count per entry, so a roster with the same
	// candidate twice would double-count the tally and render the candidate
	// twice. Reachable from both the API (candidate_ids: ["c1","c1"]) and the
	// CLI (-c c1 -c c1) (TASK-122, ISS-114).
	seen := make(map[string]bool, len(req.CandidateIDs))
	for _, id := range req.CandidateIDs {
		if seen[id] {
			return nil, voting.ErrDuplicateCandidate
		}
		seen[id] = true
	}
	// The session stores candidate IDs as references; reject dangling
	// references up front so a session never points at candidates that do
	// not exist.
	for _, id := range req.CandidateIDs {
		c, err := uc.repo.GetCandidate(id)
		if err != nil {
			if errors.Is(err, repoerr.ErrNotFound) {
				return nil, voting.ErrCandidateNotFound
			}
			return nil, fmt.Errorf("failed to get candidate: %w", err)
		}
		if c == nil {
			return nil, voting.ErrCandidateNotFound
		}
	}

	session := voting.NewSession(req.Title, req.Description, req.CandidateIDs, req.StartTime, req.EndTime)

	if err := uc.repo.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// The same shape the list and single-session endpoints return: a client
	// that created the session must see its window and created_at too.
	return SessionResponseOf(session), nil
}

// ListSessionsUseCase returns a page of sessions, exposing the sessions
// surface that CreateSession writes into but which previously had no read path
// (the API could only fetch a single session by id).
type ListSessionsUseCase struct {
	repo voting.Repository
}

func NewListSessionsUseCase(repo voting.Repository) *ListSessionsUseCase {
	return &ListSessionsUseCase{repo: repo}
}

func (uc *ListSessionsUseCase) Execute(req *ListSessionsRequest) ([]*SessionResponse, error) {
	if req == nil {
		req = &ListSessionsRequest{}
	}
	sessions, err := uc.repo.ListSessions(req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	responses := make([]*SessionResponse, len(sessions))
	for i, s := range sessions {
		responses[i] = SessionResponseOf(s)
	}
	return responses, nil
}

// SessionResponseOf renders a stored session for the API. The domain struct
// cannot do this itself: its exported fields serialize as PascalCase (ID,
// Title, StartTime), so the single-session endpoint returned a different shape
// from the list until it went through this mapping.
func SessionResponseOf(s *voting.Session) *SessionResponse {
	return &SessionResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		Status:      s.Status,
		Candidates:  s.Candidates,
		StartTime:   s.StartTime,
		EndTime:     s.EndTime,
		CreatedAt:   s.CreatedAt,
	}
}

// GetSessionUseCase returns one voting session by id, mapping the sqlite
// repo's ErrNotFound sentinel to the domain voting.ErrSessionNotFound so the
// API layer classifies a missing session as 404 (SESSION_NOT_FOUND) while a
// genuine DB failure stays an unclassified 500. The handler previously called
// h.repo.GetSession directly and folded every error — including real server
// faults — into a 404 "not found" (TASK-171, ISS-166), which disguised an
// outage as a missing resource and misdirected clients.
type GetSessionUseCase struct {
	repo voting.Repository
}

func NewGetSessionUseCase(repo voting.Repository) *GetSessionUseCase {
	return &GetSessionUseCase{repo: repo}
}

func (uc *GetSessionUseCase) Execute(id string) (*voting.Session, error) {
	session, err := uc.repo.GetSession(id)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return nil, voting.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		// In-memory fakes signal missing rows with (nil, nil); normalize.
		return nil, voting.ErrSessionNotFound
	}
	return session, nil
}
