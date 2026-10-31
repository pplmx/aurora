package deskapp

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/require"
)

// sessionStatusRepo records what the adapter asked to persist. The embedded
// interface keeps it to the two methods the status path calls.
type sessionStatusRepo struct {
	voting.TransactableRepository
	sessions map[string]*voting.Session
	updated  []string
}

func (r *sessionStatusRepo) GetSession(id string) (*voting.Session, error) {
	if s, ok := r.sessions[id]; ok {
		return s, nil
	}
	return nil, repoerr.ErrNotFound
}

func (r *sessionStatusRepo) UpdateSession(s *voting.Session) error {
	r.updated = append(r.updated, s.ID+"="+s.Status)
	return nil
}

// TestSessionStatusIsReadThenWritten pins the two steps the Session Controls
// take. The read comes first so a missing id is a not-found rather than a
// silent no-op: a session that was never stored must not report success and
// leave the button looking like it worked.
func TestSessionStatusIsReadThenWritten(t *testing.T) {
	repo := &sessionStatusRepo{sessions: map[string]*voting.Session{
		"s1": {ID: "s1", Title: "Budget", Status: voting.SessionStatusDraft},
	}}
	services := &nativeServices{votingRepo: repo}

	require.NoError(t, services.StartSession("s1"))
	require.NoError(t, services.EndSession("s1"))
	require.Equal(t, []string{
		"s1=" + voting.SessionStatusActive,
		"s1=" + voting.SessionStatusEnded,
	}, repo.updated, "start activates, end closes, and each persists once")
}

// TestSessionStatusOfAMissingSessionFails is the half that matters: without the
// read, UpdateSession would accept any id and the window would claim a session
// was started that does not exist.
func TestSessionStatusOfAMissingSessionFails(t *testing.T) {
	repo := &sessionStatusRepo{sessions: map[string]*voting.Session{}}
	services := &nativeServices{votingRepo: repo}

	require.Error(t, services.StartSession("nope"))
	require.Error(t, services.EndSession("nope"))
	require.Empty(t, repo.updated, "nothing may be written for a session that was not found")
}
