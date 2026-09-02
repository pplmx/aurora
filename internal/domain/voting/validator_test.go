package voting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests pin the free-text length bounds on voting write inputs
// (TASK-271, ISS-267): the write endpoints previously stored unbounded user
// strings while the token surface capped name/symbol, so a key-holding caller
// could grow rows and list/detail responses without bound.

func TestValidateVoterName(t *testing.T) {
	require.ErrorIs(t, ValidateVoterName(""), ErrVoterNameRequired)
	require.ErrorIs(t, ValidateVoterName("   "), ErrVoterNameRequired)
	require.NoError(t, ValidateVoterName("Alice"))
	require.ErrorIs(t, ValidateVoterName(strings.Repeat("a", MaxVoterNameLength+1)), ErrVoterNameTooLong)
	require.NoError(t, ValidateVoterName(strings.Repeat("a", MaxVoterNameLength)))
}

func TestValidateCandidateName(t *testing.T) {
	require.ErrorIs(t, ValidateCandidateName(""), ErrCandidateNameRequired)
	require.ErrorIs(t, ValidateCandidateName("   "), ErrCandidateNameRequired)
	require.NoError(t, ValidateCandidateName("Bob"))
	require.ErrorIs(t, ValidateCandidateName(strings.Repeat("b", MaxCandidateNameLength+1)), ErrCandidateNameTooLong)
}

func TestValidateCandidateParty(t *testing.T) {
	// Party is optional: empty is fine; only the bound applies.
	require.NoError(t, ValidateCandidateParty(""))
	require.NoError(t, ValidateCandidateParty("Green"))
	require.ErrorIs(t, ValidateCandidateParty(strings.Repeat("p", MaxCandidatePartyLength+1)), ErrCandidatePartyTooLong)
}

func TestValidateCandidateProgram(t *testing.T) {
	require.NoError(t, ValidateCandidateProgram(""))
	require.NoError(t, ValidateCandidateProgram("Vote for me"))
	require.ErrorIs(t, ValidateCandidateProgram(strings.Repeat("m", MaxCandidateProgramLength+1)), ErrCandidateProgramTooLong)
}

func TestValidateSessionTitle(t *testing.T) {
	require.ErrorIs(t, ValidateSessionTitle(""), ErrSessionTitleRequired)
	require.ErrorIs(t, ValidateSessionTitle("  "), ErrSessionTitleRequired)
	require.NoError(t, ValidateSessionTitle("Board Election"))
	require.ErrorIs(t, ValidateSessionTitle(strings.Repeat("t", MaxSessionTitleLength+1)), ErrSessionTitleTooLong)
}

func TestValidateSessionDescription(t *testing.T) {
	require.NoError(t, ValidateSessionDescription(""))
	require.NoError(t, ValidateSessionDescription("Annual board election"))
	require.ErrorIs(t, ValidateSessionDescription(strings.Repeat("d", MaxSessionDescriptionLen+1)), ErrSessionDescriptionTooLong)
}
