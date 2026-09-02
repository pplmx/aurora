package voting

import "strings"

// Length bounds for the free-text fields voting write endpoints accept.
// Voting previously persisted unbounded user strings into SQLite TEXT
// columns while the token surface capped name/symbol (validator.go) — a
// key-holding caller could grow rows and list/detail responses without
// bound. These caps mirror the token validator and are enforced at the
// shared domain edge so REST/CLI/web all inherit them (TASK-271, ISS-267).
const (
	MaxVoterNameLength        = 100
	MaxCandidateNameLength    = 100
	MaxCandidatePartyLength   = 100
	MaxCandidateProgramLength = 1000
	MaxSessionTitleLength     = 200
	MaxSessionDescriptionLen  = 1000
)

// ValidateVoterName rejects empty (required) and over-length names.
func ValidateVoterName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrVoterNameRequired
	}
	if len(name) > MaxVoterNameLength {
		return ErrVoterNameTooLong
	}
	return nil
}

// ValidateCandidateName rejects empty (required) and over-length names.
func ValidateCandidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrCandidateNameRequired
	}
	if len(name) > MaxCandidateNameLength {
		return ErrCandidateNameTooLong
	}
	return nil
}

// ValidateCandidateParty rejects over-length party names. Party is optional
// (may be empty), so only the bound applies.
func ValidateCandidateParty(party string) error {
	if len(party) > MaxCandidatePartyLength {
		return ErrCandidatePartyTooLong
	}
	return nil
}

// ValidateCandidateProgram rejects over-length programs. Program is optional
// (may be empty), so only the bound applies.
func ValidateCandidateProgram(program string) error {
	if len(program) > MaxCandidateProgramLength {
		return ErrCandidateProgramTooLong
	}
	return nil
}

// ValidateSessionTitle rejects empty (required) and over-length titles.
func ValidateSessionTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return ErrSessionTitleRequired
	}
	if len(title) > MaxSessionTitleLength {
		return ErrSessionTitleTooLong
	}
	return nil
}

// ValidateSessionDescription rejects over-length descriptions. Description
// is optional (may be empty), so only the bound applies.
func ValidateSessionDescription(description string) error {
	if len(description) > MaxSessionDescriptionLen {
		return ErrSessionDescriptionTooLong
	}
	return nil
}
