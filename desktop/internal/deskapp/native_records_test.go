package deskapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// The record decoder is what turns the chain's payload shapes into the ledger's
// record language. These tests pin each shape against the bytes the services
// actually write, so a new field or verb that reaches the chain shows up here
// before it reaches an auditor as an unreadable blob.

// TestDecodeRecordReadsADraw checks a lottery record: the winners are the
// subject, and the participant count and seed are the auditable facts.
func TestDecodeRecordReadsADraw(t *testing.T) {
	const payload = `{"id":"draw-1","block_height":7,"seed":"seed-abc",` +
		`"participants":["Alice","Bob","Carol"],"winners":["Alice","Carol"],` +
		`"vrf_proof":"cHJvb2Y=","vrf_output":"b3V0","vrf_public_key":"cGs=",` +
		`"timestamp":1700000000,"verified":true}`

	row := decodeBlockRecord(7, 1700000001, payload)
	assert.Equal(t, int64(7), row.Height)
	assert.Equal(t, int64(1700000001), row.Timestamp)
	assert.Equal(t, "lottery", row.Module)
	assert.Equal(t, "Draw", row.Action)
	assert.Equal(t, "Alice, Carol", row.Subject)
	assert.Equal(t, payload, row.Payload, "the exact payload must survive decoding")

	facts := factMap(row)
	assert.Equal(t, "Alice, Carol", facts["Winners"])
	assert.Equal(t, "3", facts["Participants"])
	assert.Equal(t, "seed-abc", facts["Seed"])
}

// TestDecodeRecordReadsAnObservation checks an oracle reading: the value is
// the subject, and the source is the fact that says where it came from.
func TestDecodeRecordReadsAnObservation(t *testing.T) {
	const payload = `{"id":"obs-1","source_id":"source-1","value":"68421.50",` +
		`"raw_response":"{}","timestamp":1700000000,"block_height":4}`

	row := decodeBlockRecord(4, 1700000002, payload)
	assert.Equal(t, "oracle", row.Module)
	assert.Equal(t, "Observation", row.Action)
	assert.Equal(t, "68421.50", row.Subject)

	facts := factMap(row)
	assert.Equal(t, "68421.50", facts["Value"])
	assert.Equal(t, "source-1", facts["Source"])
}

// TestDecodeRecordReadsATokenWrite checks the token verbs. Each one names a
// token and the accounts it touches; the record shows them as labelled facts
// rather than as pipe-delimited base64.
func TestDecodeRecordReadsATokenWrite(t *testing.T) {
	cases := []struct {
		payload string
		action  string
		facts   map[string]string
	}{
		{
			payload: "mint|GLD|owner",
			action:  "Mint",
			facts:   map[string]string{"Token": "GLD", "To": "owner"},
		},
		{
			payload: "transfer|GLD|alice|bob",
			action:  "Transfer",
			facts:   map[string]string{"Token": "GLD", "From": "alice", "To": "bob"},
		},
		{
			payload: "transferfrom|GLD|alice|bob",
			action:  "Transfer from",
			facts:   map[string]string{"Token": "GLD", "From": "alice", "To": "bob"},
		},
		{
			payload: "burn|GLD|alice",
			action:  "Burn",
			facts:   map[string]string{"Token": "GLD", "From": "alice"},
		},
		{
			payload: "approve|GLD|alice|bob",
			action:  "Approve",
			facts:   map[string]string{"Token": "GLD", "Owner": "alice", "Spender": "bob"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.payload, func(t *testing.T) {
			row := decodeBlockRecord(9, 0, tc.payload)
			assert.Equal(t, "token", row.Module)
			assert.Equal(t, tc.action, row.Action)
			assert.Equal(t, "GLD", row.Subject)
			facts := factMap(row)
			for label, want := range tc.facts {
				assert.Equalf(t, want, facts[label], "fact %q", label)
			}
		})
	}
}

// TestDecodeRecordReadsAnNFTWrite covers the shape the old decoder dropped: a
// transfer or burn is `<operation-uuid>|<verb>`, so the ledger used to print a
// bare UUID followed by a word. It is an NFT write, and the verb is the action.
func TestDecodeRecordReadsAnNFTWrite(t *testing.T) {
	const opID = "5c43a418-ca80-46ba-af63-cc7893d1a5e3"
	for _, verb := range []string{"transfer", "burn"} {
		t.Run(verb, func(t *testing.T) {
			row := decodeBlockRecord(6, 0, opID+"|"+verb)
			assert.Equal(t, "nft", row.Module, "an operation uuid must not become the module")
			assert.NotEmpty(t, row.Action, "the verb must become the action")
		})
	}
}

// TestDecodeRecordReadsAnNFTMint checks the verbless NFT mint payload:
// `<id>|<name>|<owner key>`, where the id is a UUID rather than a symbol.
func TestDecodeRecordReadsAnNFTMint(t *testing.T) {
	row := decodeBlockRecord(3, 0, "5c43a418-ca80-46ba-af63-cc7893d1a5e3|First Art|owner-key")
	assert.Equal(t, "nft", row.Module)
	assert.Equal(t, "Mint", row.Action)
	assert.Equal(t, "First Art", row.Subject)
	facts := factMap(row)
	assert.Equal(t, "First Art", facts["Name"])
	assert.Equal(t, "owner-key", facts["Owner"])
}

// TestDecodeRecordReadsAVote checks a cast ballot: the candidate is the
// subject, and the voter and session are the facts that place it.
func TestDecodeRecordReadsAVote(t *testing.T) {
	row := decodeBlockRecord(5, 0, "vote|dGVzdC1rZXk=|candidate-1|session-1")
	assert.Equal(t, "voting", row.Module)
	assert.Equal(t, "Ballot", row.Action)
	assert.Equal(t, "candidate-1", row.Subject)
	facts := factMap(row)
	assert.Equal(t, "dGVzdC1rZXk=", facts["Voter"])
	assert.Equal(t, "session-1", facts["Session"])
}

// TestDecodeRecordKeepsBinaryKeyBytesOut checks that an unprintable field is
// shown as base64. The exact payload keeps its bytes on purpose — it is the
// evidence — so only what the record renders is asserted printable.
func TestDecodeRecordKeepsBinaryKeyBytesOut(t *testing.T) {
	row := decodeBlockRecord(2, 0, "mint|AUR|\x00\x01\x02binary")
	require.NotEmpty(t, row.Facts)
	readable := row.Summary
	for _, fact := range row.Facts {
		readable += " " + fact.Value
	}
	assert.NotContains(t, readable, "\x00", "raw bytes must not reach the rendered record")
	assert.Contains(t, readable, "AAECYmluYXJ5", "a binary field is shown as base64")
}

// TestDecodeRecordKeepsAnUnknownPayloadReadable checks the fallback: a payload
// the decoder does not recognise still reads as its stored text rather than as
// an empty record, and the exact bytes are still available.
func TestDecodeRecordKeepsAnUnknownPayloadReadable(t *testing.T) {
	const payload = "some future module wrote this"
	row := decodeBlockRecord(11, 0, payload)
	assert.Equal(t, int64(11), row.Height)
	assert.Empty(t, row.Module, "an unknown payload must not be given an invented module")
	assert.Equal(t, payload, row.Payload)
	assert.Equal(t, payload, row.Summary)
}

// TestRecordSearchReachesDecodedText keeps the ledger's search useful: an
// operator looking for a winner's name or the word "transfer" must find the
// entry even though the raw payload spells neither that way.
func TestRecordSearchReachesDecodedText(t *testing.T) {
	row := decodeBlockRecord(7, 1700000001,
		`{"participants":["Alice","Bob"],"winners":["Alice"],"seed":"s"}`)
	assert.True(t, recordSearchMatches(row, "alice"), "a winner's name must be searchable")
	assert.True(t, recordSearchMatches(row, "draw"), "the decoded action must be searchable")
	assert.True(t, recordSearchMatches(row, "7"), "the height must be searchable")
	assert.True(t, recordSearchMatches(row, "PARTICIPANTS"), "search is case-insensitive")
	assert.False(t, recordSearchMatches(row, "nothing-like-this"))

	empty := decodeBlockRecord(1, 0, "")
	assert.True(t, recordSearchMatches(empty, ""), "an empty query matches everything")
}

// factMap turns a record's facts into a label map for assertions.
func factMap(row nativeui.BlockRow) map[string]string {
	out := make(map[string]string, len(row.Facts))
	for _, fact := range row.Facts {
		out[fact.Label] = fact.Value
	}
	return out
}
