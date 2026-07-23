// Package lottery provides VRF-based transparent lottery functionality.
// It implements verifiable random function (VRF) to ensure fair and
// transparent winner selection that can be verified on-chain.
package lottery

import "errors"

// ErrNotFound is returned when a lottery record is not found in the
// repository.
var ErrNotFound = errors.New("lottery record not found")

// Validation errors for lottery participant names, seeds, and winner counts.
// Using sentinels allows API handlers to map these to 400 Bad Request via
// errors.Is, rather than falling through to 500 Internal Server Error.
var (
	ErrEmptyParticipantName      = errors.New("participant name cannot be empty")
	ErrParticipantNameTooLong    = errors.New("participant name too long")
	ErrInvalidParticipantName    = errors.New("participant name contains invalid characters")
	ErrSeedTooShort              = errors.New("seed too short")
	ErrSeedTooLong               = errors.New("seed too long")
	ErrNoParticipants            = errors.New("at least one participant required")
	ErrTooManyParticipants       = errors.New("too many participants")
	ErrDuplicateParticipant      = errors.New("duplicate participant")
	ErrWinnerCountNotPositive    = errors.New("winner count must be positive")
	ErrTooManyWinners            = errors.New("too many winners")
	ErrWinnersExceedParticipants = errors.New("winner count cannot exceed participants")
)
