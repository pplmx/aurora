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
