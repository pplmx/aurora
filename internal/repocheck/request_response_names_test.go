package repocheck

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/app/votingapp"
)

// TestTheReferenceNamesTheRenamedRequestFields covers the two places where a
// request key and the response field that supplies it are spelled differently:
//
//	response.candidates         → request.candidate_ids
//	response.public_key         → request.voter_public_key
//
// Both were documented as JSON samples that hinted at the mapping without
// stating it. A client that reads an object and posts it back — the obvious way
// to write "edit this session" — sends the wrong key and gets an empty roster
// or an empty public key, and the error names neither field.
//
// The check marshals real DTOs, so a rename on either side fails here rather
// than leaving the paragraph wrong.
func TestTheReferenceNamesTheRenamedRequestFields(t *testing.T) {
	// The response side, from the types the handlers actually write.
	session, err := json.Marshal(votingapp.SessionResponse{
		ID: "s1", Title: "T", Candidates: []string{"c1"},
	})
	require.NoError(t, err)
	var sessionKeys map[string]any
	require.NoError(t, json.Unmarshal(session, &sessionKeys))
	assert.Contains(t, sessionKeys, "candidates",
		"the session response must still call the roster \"candidates\": %s", session)
	assert.NotContains(t, sessionKeys, "candidate_ids")

	voter, err := json.Marshal(votingapp.VoterResponse{ID: "v1", Name: "N", PublicKey: "pk"})
	require.NoError(t, err)
	var voterKeys map[string]any
	require.NoError(t, json.Unmarshal(voter, &voterKeys))
	assert.Contains(t, voterKeys, "public_key",
		"the voter response must still call the key \"public_key\": %s", voter)
	assert.NotContains(t, voterKeys, "voter_public_key")

	// The request side: the handler's anonymous struct, read from source because
	// it is not exported as a type.
	voting := readRepoFile(t, "internal", "api", "handler", "voting.go")
	for _, want := range []string{`json:"candidate_ids"`, `json:"voter_public_key"`} {
		assert.Containsf(t, voting, want,
			"the vote and session handlers must still accept %s", want)
	}
	// And they must not accept the response spelling, which would make the
	// paragraph wrong in the other direction.
	assert.NotContains(t, voting, `json:"candidates"`,
		"a session request must not accept the response's field name")

	// The reference has to state both mappings.
	doc := readRepoFile(t, "docs", "api.md")
	require.True(t, strings.Contains(doc, "请求字段是 `candidate_ids`"),
		"docs/api.md must name the request spelling of the roster")
	require.True(t, strings.Contains(doc, "**`voter_public_key`**"),
		"docs/api.md must name the vote request's key field")
	assert.Regexp(t, regexp.MustCompile(`public_key.*voter_public_key|voter_public_key.*public_key`),
		strings.ReplaceAll(doc, "\n", " "),
		"docs/api.md must show the two spellings together")
}
