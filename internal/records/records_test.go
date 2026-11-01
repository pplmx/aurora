package records

import (
	"encoding/base64"
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
			want:    Record{Module: "nft", Action: "Mint", Subject: "Star", Facts: []Fact{{"Name", "Star"}, {"Holder", "ownerkey"}}},
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

// TestReadableFieldsEncodeEveryUnprintableByte covers the characters a
// terminal interprets rather than shows. The old test had NUL and high bytes,
// which fail the UTF-8 check on their own; DEL and the ESC sequence are valid
// UTF-8 and were only caught by isPrintable, so a regression there would have
// reached the CLI and the ledger card as a colour change or a swallowed line.
func TestReadableFieldsEncodeEveryUnprintableByte(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"delete", "abc\x7fdef"},
		{"escape sequence", "\x1b[31mred\x1b[0m"},
		{"bell", "ring\x07"},
		{"carriage return", "over\rwrite"},
		{"newline", "line1\nline2"},
		{"tab", "a\tb"},
		{"unit separator", "a\x1fb"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ReadableFields([]string{tc.in})
			require.Len(t, got, 1)
			assert.NotEqual(t, tc.in, got[0], "an unprintable field must not be rendered as-is")
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(tc.in)), got[0],
				"the field is preserved exactly, just encoded")
			assert.True(t, IsPrintable(got[0]), "what reaches the terminal is printable")
		})
	}
}

// TestReadableFieldsLeaveOrdinaryTextAlone is the other half: printable text,
// including non-ASCII, must not be encoded. An audit row whose amounts and names
// were base64 would be unreadable, which is the failure the encoding exists to
// prevent in the first place.
func TestReadableFieldsLeaveOrdinaryTextAlone(t *testing.T) {
	in := []string{"GLD", "价格 100", "0xdeadbeef", "", "  spaced  "}
	assert.Equal(t, in, ReadableFields(in))
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

// TestParseKeepsControlCharactersOutOfJSONPayloads is the JSON twin of
// TestReadableFieldsEncodeEveryUnprintableByte. The pipe payloads were always
// passed through ReadableFields; the JSON ones were not, so a winner renamed to
// an escape sequence reached the CLI table and the ledger card as a live colour
// change, and an embedded newline split the row it was supposed to label.
func TestParseKeepsControlCharactersOutOfJSONPayloads(t *testing.T) {
	// A draw whose winning name carries a terminal escape and whose seed rings
	// the bell. Both are valid JSON strings.
	payload := `{"participants":["alice","bob"],"winners":["evil\u001b[31mRED\u001b[0m"],` +
		`"seed":"\u0007bell","id":"abc"}`

	record, ok := Parse(payload)
	require.True(t, ok, "the payload is a lottery record")

	for _, field := range []struct{ name, value string }{
		{"Subject", record.Subject},
		{"Winners", factValue(record, "Winners")},
		{"Seed", factValue(record, "Seed")},
	} {
		assert.NotContainsf(t, field.value, "\x1b", "%s must not carry an escape sequence", field.name)
		assert.NotContainsf(t, field.value, "\x07", "%s must not carry a bell", field.name)
		assert.Truef(t, IsPrintable(field.value), "%s must be printable", field.name)
	}

	// The rendered sentence is what a table row prints.
	assert.True(t, IsPrintable(record.Text()), "the whole rendered line must be printable")

	// An embedded newline is encoded too: it would otherwise break the row.
	nl, ok := Parse(`{"participants":["a"],"winners":["line1\nline2"],"seed":"s"}`)
	require.True(t, ok)
	assert.NotContains(t, nl.Subject, "\n", "a newline in a name must not reach the table")
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("line1\nline2")), nl.Subject,
		"the name is preserved exactly, just encoded")
}

// factValue returns one fact's value, or "" when the record has no such fact.
func factValue(record Record, label string) string {
	for _, fact := range record.Facts {
		if fact.Label == label {
			return fact.Value
		}
	}
	return ""
}

// TestParseLeavesOrdinaryJSONTextAlone is the other half: plain names and seeds
// must not be encoded. A ledger whose winner names were base64 would be
// unreadable, which is the failure the encoding exists to prevent.
func TestParseLeavesOrdinaryJSONTextAlone(t *testing.T) {
	record, ok := Parse(`{"participants":["alice","bob"],"winners":["Alice","Bob"],` +
		`"seed":"seed-2026","id":"abc"}`)
	require.True(t, ok)
	assert.Equal(t, "Alice, Bob", record.Subject)
	assert.Equal(t, "seed-2026", factValue(record, "Seed"))
	assert.Equal(t, "Alice, Bob", factValue(record, "Winners"))
}

// TestReadableTextEncodesAnUnrecognisedPayload covers the fallback the display
// paths use. A payload matching no known shape has no fields to inspect, so the
// whole value has to be tested: returning it unchanged let a forged ledger row
// carry an escape sequence into the CLI table and the window's ledger card.
func TestReadableTextEncodesAnUnrecognisedPayload(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"escape sequence", "raw audit \x1b[31mRED\x1b[0m text"},
		{"embedded newline", "line1\nline2"},
		{"bell", "ring\x07"},
		{"carriage return", "over\rwrite"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ReadableText(tc.in)
			assert.NotEqual(t, tc.in, got, "an unprintable payload must not print as-is")
			assert.True(t, IsPrintable(got), "what reaches the terminal is printable")
			assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(tc.in)), got,
				"the payload is preserved exactly, just encoded")
		})
	}

	// Printable text, including Chinese, passes through untouched: an audit
	// trail encoded wholesale would be unreadable.
	for _, plain := range []string{"lottery module block", "价格 100", "a b  c"} {
		assert.Equal(t, plain, ReadableText(plain), "printable text must not be encoded")
	}
}

// TestParseRejectsEveryUnrecognisedShape keeps the fallback reachable for the
// reason it exists: these payloads match nothing, so Parse must say so rather
// than inventing a record.
func TestParseRejectsEveryUnrecognisedShape(t *testing.T) {
	for _, payload := range []string{
		"raw audit text with no separators",
		"unknownverb|a|b",
		`{"unrelated":true}`,
		`not json at all`,
	} {
		_, ok := Parse(payload)
		assert.Falsef(t, ok, "%q must not parse as a record", payload)
	}
}

// TestTextLResolvesEveryWord covers the localized rendering: each vocabulary
// word goes through the resolver, the subject and the fact values do not, and an
// action that resolves to the module's own name is dropped rather than repeated.
func TestTextLResolvesEveryWord(t *testing.T) {
	record := Record{
		Module: "lottery", Action: "Draw", Subject: "Alice",
		Facts: []Fact{{Label: "Winners", Value: "Alice"}, {Label: "Seed", Value: "s"}},
	}
	resolve := func(kind, word string) string {
		switch kind + ":" + word {
		case "module:lottery":
			return "抽奖"
		case "action:Draw":
			return "抽奖"
		case "fact:Winners":
			return "中奖者"
		case "fact:Seed":
			return "种子"
		}
		return word
	}
	style := Style{Divider: " · ", Label: "：", Fact: "；", Open: "（", Close: "）"}

	require.Equal(t, "抽奖 · Alice（中奖者：Alice；种子：s）", record.TextL(resolve, style))

	// A nil resolver keeps the English rendering, so a caller that has no
	// catalog yet prints the sentence rather than losing the record.
	require.Equal(t, record.Text(), record.TextL(nil, DefaultStyle))
	// The default style is what Text has always written.
	require.Equal(t, "lottery · Draw · Alice   (Winners: Alice; Seed: s)", record.TextIn(DefaultStyle))
}
