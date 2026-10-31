package records

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseReadsEveryShape pins the vocabulary each payload shape produces. The
// window's ledger page and the CLI's record list both render it, so a shape that
// stops decoding would regress two surfaces at once.
func TestParseReadsEveryShape(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    Record
		ok      bool
	}{
		{
			name:    "lottery draw",
			payload: `{"id":"abc","seed":"s","participants":["A","B"],"winners":["B"],"verified":true}`,
			want:    Record{Module: "lottery", Action: "Draw", Subject: "B", Facts: []Fact{{"Winners", "B"}, {"Participants", "2"}, {"Seed", "s"}}},
			ok:      true,
		},
		{
			name:    "oracle observation",
			payload: `{"source_id":"src-1","value":"42","timestamp":1}`,
			want:    Record{Module: "oracle", Action: "Observation", Subject: "42", Facts: []Fact{{"Value", "42"}, {"Source", "src-1"}}},
			ok:      true,
		},
		{
			name:    "voting block",
			payload: `{"module":"voting","candidate_id":"c1","session_id":"s1","description":"Cast ballot"}`,
			want:    Record{Module: "voting", Action: "Ballot", Subject: "Cast ballot", Facts: []Fact{{"Candidate", "c1"}, {"Session", "s1"}}},
			ok:      true,
		},
		{
			name:    "token transfer",
			payload: "transfer|GLD|alice|bob",
			want:    Record{Module: "token", Action: "Transfer", Subject: "GLD", Facts: []Fact{{"Token", "GLD"}, {"From", "alice"}, {"To", "bob"}}},
			ok:      true,
		},
		{
			name:    "nft transfer",
			payload: "0ba3647f-d955-4e53-8feb-a3932b00c75d|transfer",
			want:    Record{Module: "nft", Action: "Transfer"},
			ok:      true,
		},
		{
			name:    "nft mint",
			payload: "0ba3647f-d955-4e53-8feb-a3932b00c75d|Star|ownerkey",
			want:    Record{Module: "nft", Action: "Mint", Subject: "Star", Facts: []Fact{{"Name", "Star"}, {"Owner", "ownerkey"}}},
			ok:      true,
		},
		{
			name:    "cast ballot",
			payload: "vote|voterpk|cand-1|session-1",
			want:    Record{Module: "voting", Action: "Ballot", Subject: "cand-1", Facts: []Fact{{"Candidate", "cand-1"}, {"Voter", "voterpk"}, {"Session", "session-1"}}},
			ok:      true,
		},
		{
			name:    "unknown payload",
			payload: "plain text the chain stores as-is",
			ok:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Parse(tc.payload)
			require.Equal(t, tc.ok, ok)
			if !tc.ok {
				return
			}
			assert.Equal(t, tc.want.Module, got.Module)
			assert.Equal(t, tc.want.Action, got.Action)
			assert.Equal(t, tc.want.Subject, got.Subject)
			assert.Equal(t, tc.want.Facts, got.Facts)
		})
	}
}

// TestReadableFieldsKeepKeyMaterialPrintable pins the rule that makes an audit
// row readable: raw key bytes become base64 rather than control characters.
func TestReadableFieldsKeepKeyMaterialPrintable(t *testing.T) {
	raw := string([]byte{0x00, 0x01, 0x02, 0xff})
	got := ReadableFields([]string{"GLD", raw})
	require.Len(t, got, 2)
	assert.Equal(t, "GLD", got[0])
	assert.NotContains(t, got[1], "\x00")
	assert.NotEmpty(t, got[1])
}

// TestParseReadsModuleNamedBlocks pins the synthetic blocks the API's audit
// writer stores: they name their module instead of carrying its fields, and the
// ledger must still label them rather than present an unknown payload. The
// deskapp suite caught this case disappearing during the extraction.
func TestParseReadsModuleNamedBlocks(t *testing.T) {
	lottery, ok := Parse(`{"module":"lottery"}`)
	require.True(t, ok)
	assert.Equal(t, "lottery", lottery.Module)
	assert.Equal(t, "Draw", lottery.Action)

	voting, ok := Parse(`{"module":"voting","session_id":"s1"}`)
	require.True(t, ok)
	assert.Equal(t, "voting", voting.Module)
	assert.Equal(t, "Ballot", voting.Action)
	assert.Equal(t, "s1", voting.Subject)
}

// TestRecordTextRendersOneLine pins the sentence the CLI prints and every
// search compares: module · action · subject, then the facts.
func TestRecordTextRendersOneLine(t *testing.T) {
	record, ok := Parse("transfer|GLD|alice|bob")
	require.True(t, ok)
	assert.Equal(t, "token · Transfer · GLD   (Token: GLD; From: alice; To: bob)", record.Text())
}

// TestParseReadsThePipeVerbs covers every verb the services write, so a new
// forbidden verb or a renamed action shows up here rather than in a surface.
func TestParseReadsThePipeVerbs(t *testing.T) {
	cases := []struct {
		payload string
		module  string
		action  string
		subject string
	}{
		{"mint_nft|Star|owner", "nft", "Mint", "Star"},
		{"transfer_nft|Star|alice|bob", "nft", "Transfer", "Star"},
		{"burn_nft|Star|alice", "nft", "Burn", "Star"},
		{"mint|GLD|alice", "token", "Mint", "GLD"},
		{"burn|GLD|alice", "token", "Burn", "GLD"},
		{"approve|GLD|alice|bob", "token", "Approve", "GLD"},
		{"transfer_from|GLD|alice|bob", "token", "Transfer from", "GLD"},
		{"transferfrom|GLD|alice|bob", "token", "Transfer from", "GLD"},
	}
	for _, tc := range cases {
		record, ok := Parse(tc.payload)
		require.True(t, ok, tc.payload)
		assert.Equal(t, tc.module, record.Module, tc.payload)
		assert.Equal(t, tc.action, record.Action, tc.payload)
		assert.Equal(t, tc.subject, record.Subject, tc.payload)
	}
}

// TestParseRejectsMalformedPayloads keeps an unknown shape from claiming a
// module: the caller then shows the stored text instead of a guess.
func TestParseRejectsMalformedPayloads(t *testing.T) {
	for _, payload := range []string{"", "{not json", "mint_nft", "|", "plain text"} {
		if _, ok := Parse(payload); ok {
			t.Errorf("%q must not parse as a known record", payload)
		}
	}
}

// TestIsUUID pins the canonical check the pipe parser uses to recognise the
// NFT's verb-less mint payload.
func TestIsUUID(t *testing.T) {
	assert.True(t, IsUUID("5c43a418-ca80-46ba-af63-cc7893d1a5e3"))
	assert.False(t, IsUUID("5c43a418-ca80-46ba-af63-cc7893d1a5e"), "too short")
	assert.False(t, IsUUID("5c43a418xca80-46ba-af63-cc7893d1a5e3"), "bad separator")
	assert.False(t, IsUUID("5c43a418-ca80-46ba-af63-cc7893d1a5eg"), "non-hex digit")
}
