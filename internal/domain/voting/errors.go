package voting

import "errors"

// Domain errors for the voting system. These sentinel errors allow
// API handlers and callers to classify failures by type via errors.Is.
var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionNotStarted  = errors.New("voting session has not started yet")
	ErrSessionEnded       = errors.New("voting session has ended")
	ErrVoterNotRegistered = errors.New("voter not registered")
	ErrCandidateNotFound  = errors.New("candidate not found")
	ErrAlreadyVoted       = errors.New("voter has already voted")
)
