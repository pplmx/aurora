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
	// ErrCandidateNotInSession guards ballot integrity: a vote must target a
	// candidate that is part of the session's candidate roster. Without this,
	// a caller could name any registered candidate (even one in a different
	// election) and inflate that candidate's tally.
	ErrCandidateNotInSession = errors.New("candidate is not part of this session")
	ErrAlreadyVoted          = errors.New("voter has already voted")
	// ErrInvalidSignature guards ballot authenticity: the private key presented
	// must correspond to the registered voter's public key. Without this, a
	// caller who knows a voter's public key could forge that voter's ballot
	// with a random private key.
	ErrInvalidSignature = errors.New("invalid vote signature")
	// ErrInvalidBase64 is returned when a base64-encoded key field cannot be
	// decoded at all — a client input error (HTTP 400), not a server fault
	// (TASK-095, ISS-089).
	ErrInvalidBase64 = errors.New("invalid base64 encoding")

	// Validation sentinels. These let API handlers classify client input
	// errors as 400s instead of falling through to 500.
	ErrVoterNameRequired     = errors.New("voter name is required")
	ErrCandidateNameRequired = errors.New("candidate name is required")
	ErrSessionTitleRequired  = errors.New("session title is required")
	ErrInvalidSessionTime    = errors.New("session end time must be after start time")
	ErrCandidatesRequired    = errors.New("at least one candidate is required")
)
