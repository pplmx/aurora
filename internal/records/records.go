// Package records decodes the payloads the modules write onto the shared
// ledger into one small vocabulary: which module wrote the record, what it did,
// the object it names and the facts worth reading.
//
// The chain stores heterogeneous payloads: lottery and oracle records are JSON,
// while token, NFT and voting records are a `verb|field|field` line carrying
// base64 or raw key bytes. Rendering those verbatim produced an audit trail of
// JSON blobs and pipe-delimited bytes; every representation layer (the window's
// ledger page, the CLI's record list) renders this vocabulary instead, and the
// exact payload stays available for the auditor who needs the bytes.
package records

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Fact is one labelled value read out of a payload. Fields whose value is empty
// are dropped, so a record never shows a label with nothing beside it.
type Fact struct {
	Label string
	Value string
}

// Record is one decoded ledger payload. Module is empty when the payload does
// not match any shape this package knows.
type Record struct {
	Module  string
	Action  string
	Subject string
	Facts   []Fact
}

// Parse maps a raw payload to the record's vocabulary. It reports false when
// the payload matches no known shape, so a caller can keep the stored text as
// the only description instead of inventing one.
func Parse(payload string) (Record, bool) {
	if strings.HasPrefix(strings.TrimSpace(payload), "{") {
		return parseJSONPayload(payload)
	}
	return parsePipePayload(payload)
}

// parseJSONPayload reads the JSON shapes: a lottery draw, an oracle observation
// and a voting block that names its module.
func parseJSONPayload(payload string) (Record, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return Record{}, false
	}
	// Lottery records name themselves by their participants and winners.
	if _, ok := fields["participants"]; ok {
		return lotteryRecord(fields), true
	}
	// Oracle observations identify their source rather than naming a module.
	if _, ok := fields["source_id"]; ok {
		return oracleRecord(fields), true
	}
	// A block written by the API's audit writer names its module instead of
	// carrying the module's own fields.
	for _, key := range []string{"module", "type", "kind"} {
		switch jsonString(fields[key]) {
		case "lottery":
			return lotteryRecord(fields), true
		case "voting":
			return votingRecord(fields), true
		}
	}
	return Record{}, false
}

// lotteryRecord describes a stored draw: how many winners came out of how many
// participants, and who they were. The seed is the auditable input, so it
// follows the outcome rather than leading the record.
func lotteryRecord(fields map[string]json.RawMessage) Record {
	participants := jsonStringSlice(fields["participants"])
	winners := jsonStringSlice(fields["winners"])
	subject := strings.Join(winners, ", ")
	if subject == "" {
		subject = jsonString(fields["id"])
	}
	facts := appendFact(nil, "Winners", subject)
	facts = appendFact(facts, "Participants", strconv.Itoa(len(participants)))
	facts = appendFact(facts, "Seed", jsonString(fields["seed"]))
	return Record{Module: "lottery", Action: "Draw", Subject: subject, Facts: facts}
}

// oracleRecord describes one fetched observation: which feed produced it and
// what value it returned. The value is the subject because it is the thing an
// auditor is checking.
func oracleRecord(fields map[string]json.RawMessage) Record {
	value := jsonString(fields["value"])
	facts := appendFact(nil, "Value", value)
	facts = appendFact(facts, "Source", jsonString(fields["source_id"]))
	return Record{Module: "oracle", Action: "Observation", Subject: value, Facts: facts}
}

// votingRecord describes a JSON voting block that names a module and describes
// its action. Cast ballots are a pipe payload; this shape reaches the chain
// from the API's own audit writer.
func votingRecord(fields map[string]json.RawMessage) Record {
	subject := jsonString(fields["description"])
	if subject == "" {
		subject = jsonString(fields["session_id"])
	}
	facts := appendFact(nil, "Candidate", jsonString(fields["candidate_id"]))
	facts = appendFact(facts, "Session", jsonString(fields["session_id"]))
	return Record{Module: "voting", Action: "Ballot", Subject: subject, Facts: facts}
}

// parsePipePayload reads the `verb|field|field` shapes. The verbs are the ones
// the token, NFT and voting services actually write; see
// internal/domain/{token,nft} and internal/app/votingapp.
func parsePipePayload(payload string) (Record, bool) {
	if !strings.Contains(payload, "|") {
		return Record{}, false
	}
	parts := strings.Split(payload, "|")
	// The NFT services write `<operation-uuid>|<verb>` for a transfer or a
	// burn. Without this branch those records rendered as a bare UUID followed
	// by a word, which named no module at all.
	if len(parts) == 2 && IsUUID(parts[0]) {
		switch parts[1] {
		case "transfer", "burn":
			return Record{Module: "nft", Action: titleAction(parts[1])}, true
		}
	}
	fields := ReadableFields(parts[1:])
	verb := parts[0]
	switch verb {
	case "mint", "transfer", "transfer_from", "transferfrom", "burn", "approve":
		return tokenRecord(verb, fields), true
	case "mint_nft", "transfer_nft", "burn_nft":
		return nftRecord(verb, fields), true
	case "vote":
		return voteRecord(fields), true
	}
	// An NFT's own mint payload has no verb: `<id>|<name>|<owner key>`, the
	// id being a UUID rather than a token symbol.
	if len(parts) >= 3 && IsUUID(parts[0]) {
		all := ReadableFields(parts)
		subject := fieldAt(all, 1)
		facts := appendFact(nil, "Name", subject)
		facts = appendFact(facts, "Owner", fieldAt(all, 2))
		return Record{Module: "nft", Action: "Mint", Subject: subject, Facts: facts}, true
	}
	return Record{}, false
}

// tokenRecord reads a token write. The pipe shape is `<verb>|<token>|<from>
// [|<to>]`: mint carries only the recipient, transfer carries both ends.
func tokenRecord(verb string, fields []string) Record {
	action := titleAction(verb)
	record := Record{Module: "token", Action: action}
	if len(fields) == 0 {
		return record
	}
	record.Subject = fields[0]
	facts := appendFact(nil, "Token", record.Subject)
	switch verb {
	case "mint":
		facts = appendFact(facts, "To", fieldAt(fields, 1))
	case "burn":
		facts = appendFact(facts, "From", fieldAt(fields, 1))
	case "transfer", "transfer_from", "transferfrom":
		facts = appendFact(facts, "From", fieldAt(fields, 1))
		facts = appendFact(facts, "To", fieldAt(fields, 2))
	case "approve":
		facts = appendFact(facts, "Owner", fieldAt(fields, 1))
		facts = appendFact(facts, "Spender", fieldAt(fields, 2))
	}
	record.Facts = facts
	return record
}

// nftRecord reads an NFT write; its first readable field is the asset's name.
func nftRecord(verb string, fields []string) Record {
	record := Record{Module: "nft", Action: titleAction(strings.TrimSuffix(verb, "_nft"))}
	if len(fields) > 0 {
		record.Subject = fields[0]
	}
	record.Facts = appendFact(nil, "NFT", record.Subject)
	return record
}

// voteRecord reads a cast ballot: who voted, for whom, in which session.
func voteRecord(fields []string) Record {
	subject := fieldAt(fields, 1)
	facts := appendFact(nil, "Candidate", subject)
	facts = appendFact(facts, "Voter", fieldAt(fields, 0))
	facts = appendFact(facts, "Session", fieldAt(fields, 2))
	return Record{Module: "voting", Action: "Ballot", Subject: subject, Facts: facts}
}

// titleAction turns a payload verb into the word a record shows. The verbs are
// spelled the way the services write them; the record language is the
// human-readable form.
func titleAction(verb string) string {
	switch verb {
	case "mint":
		return "Mint"
	case "transfer":
		return "Transfer"
	case "transfer_from", "transferfrom":
		return "Transfer from"
	case "burn":
		return "Burn"
	case "approve":
		return "Approve"
	}
	if verb == "" {
		return ""
	}
	return strings.ToUpper(verb[:1]) + verb[1:]
}

// fieldAt reads one pipe field, tolerating a short payload rather than indexing
// past its end.
func fieldAt(fields []string, index int) string {
	if index < 0 || index >= len(fields) {
		return ""
	}
	return fields[index]
}

// appendFact adds a labelled value, skipping an empty one.
func appendFact(facts []Fact, label, value string) []Fact {
	value = strings.TrimSpace(value)
	if value == "" {
		return facts
	}
	return append(facts, Fact{Label: label, Value: value})
}

// jsonString reads one string field, returning "" when the field is absent or
// holds another JSON type.
func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return ""
}

// jsonStringSlice reads a JSON array of strings, tolerating either a missing
// field or a different shape.
func jsonStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if json.Unmarshal(raw, &values) == nil {
		return values
	}
	return nil
}

// ReadableFields renders each pipe field as text, or as base64 when it carries
// bytes a terminal cannot show. Display helpers use it too, so a payload never
// appears as raw control characters.
func ReadableFields(fields []string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if utf8.ValidString(f) && isPrintable(f) {
			out = append(out, f)
			continue
		}
		out = append(out, base64.StdEncoding.EncodeToString([]byte(f)))
	}
	return out
}

// IsUUID reports whether s is a canonical 8-4-4-4-12 UUID.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return false
			}
		}
	}
	return true
}

// isPrintable reports whether every rune in s has a visible form.
func isPrintable(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
