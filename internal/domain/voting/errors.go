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

	// Validation sentinels. These let API handlers classify client input
	// errors as 400s instead of falling through to 500.
	ErrVoterNameRequired     = errors.New("voter name is required")
	ErrCandidateNameRequired = errors.New("candidate name is required")
	ErrSessionTitleRequired  = errors.New("session title is required")
	ErrInvalidSessionTime    = errors.New("session end time must be after start time")
	ErrCandidatesRequired    = errors.New("at least one candidate is required")
)
