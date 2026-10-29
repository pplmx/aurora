package deskapp

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// The ledger's record language.
//
// Every module writes its own payload shape: lottery and oracle records are
// JSON that can be decoded field by field, while token, NFT and voting records
// are a `verb|field|field` line carrying base64 or raw key bytes. Rendering
// those verbatim was the old ledger: a table of JSON blobs and pipe-delimited
// key bytes, which is an audit trail that nobody can read.
//
// decodeBlockRecord turns each shape into the same small vocabulary: which
// module wrote it, what it did, the one object it names, and the facts worth
// reading. The page then draws that vocabulary as a record instead of a cell,
// and the exact payload stays available underneath for the auditor who needs
// the bytes rather than the sentence.

// recordFact is one labelled value read out of a payload. Fields whose value
// is empty are dropped by the decoder, so a record never shows a label with
// nothing beside it.
type recordFact struct {
	label string
	value string
}

// decodeBlockRecord maps a raw payload to the record the ledger page shows.
//
// height is passed through so an unknown payload still reads as a real chain
// entry rather than as an unexplained blob, and timestamp is the block's own
// time, the only clock every payload shape shares.
func decodeBlockRecord(height, timestamp int64, payload string) nativeui.BlockRow {
	row := nativeui.BlockRow{
		Height:    height,
		Timestamp: timestamp,
		Payload:   payload,
	}
	if module, action, subject, facts := parseBlockPayload(payload); module != "" {
		row.Module = module
		row.Action = action
		row.Subject = subject
		for _, fact := range facts {
			row.Facts = append(row.Facts, nativeui.BlockFact{Label: fact.label, Value: fact.value})
		}
	}
	// An unrecognised payload keeps its module empty rather than taking the
	// em dash the module column used to print for "unknown": the dash is a
	// display placeholder, and a module named "—" would claim the payload
	// identified itself when it did not. The view shows the stored text under
	// no module instead.
	row.Summary = clipBlockText(blockText(payload))
	return row
}

// parseBlockPayload dispatches on the two shapes the chain actually holds.
func parseBlockPayload(payload string) (module, action, subject string, facts []recordFact) {
	if strings.HasPrefix(strings.TrimSpace(payload), "{") {
		return parseJSONRecord(payload)
	}
	return parsePipeRecord(payload)
}

// parseJSONRecord reads the JSON shapes: a lottery draw, an oracle observation
// and a voting block that names its module.
func parseJSONRecord(payload string) (module, action, subject string, facts []recordFact) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return "", "", "", nil
	}
	// Lottery records name themselves by their participants and winners.
	if _, ok := fields["participants"]; ok {
		return lotteryRecord(fields)
	}
	if _, ok := fields["source_id"]; ok {
		return oracleRecord(fields)
	}
	named, _ := blockFields(payload)
	switch named {
	case "lottery":
		return lotteryRecord(fields)
	case "voting":
		action, subject, facts = votingRecord(fields)
		return "voting", action, subject, facts
	}
	return "", "", "", nil
}

// lotteryRecord describes a stored draw: how many winners came out of how many
// participants, and who they were. The seed is the auditable input, so it
// follows the outcome rather than leading the record.
func lotteryRecord(fields map[string]json.RawMessage) (module, action, subject string, facts []recordFact) {
	participants := jsonStringSlice(fields["participants"])
	winners := jsonStringSlice(fields["winners"])
	subject = strings.Join(winners, ", ")
	if subject == "" {
		subject = jsonString(fields["id"])
	}
	facts = appendFact(facts, "Winners", subject)
	facts = appendFact(facts, "Participants", strconv.Itoa(len(participants)))
	facts = appendFact(facts, "Seed", jsonString(fields["seed"]))
	return "lottery", "Draw", subject, facts
}

// oracleRecord describes one fetched observation: which feed produced it and
// what value it returned. The value is the subject because it is the thing an
// auditor is checking.
func oracleRecord(fields map[string]json.RawMessage) (module, action, subject string, facts []recordFact) {
	value := jsonString(fields["value"])
	source := jsonString(fields["source_id"])
	facts = appendFact(facts, "Value", value)
	facts = appendFact(facts, "Source", source)
	return "oracle", "Observation", value, facts
}

// votingRecord describes a JSON voting block that names a module and describes
// its action. Cast ballots are a pipe payload; this shape reaches the chain
// from the API's own audit writer.
func votingRecord(fields map[string]json.RawMessage) (action, subject string, facts []recordFact) {
	description := jsonString(fields["description"])
	subject = description
	if subject == "" {
		subject = jsonString(fields["session_id"])
	}
	facts = appendFact(facts, "Candidate", jsonString(fields["candidate_id"]))
	facts = appendFact(facts, "Session", jsonString(fields["session_id"]))
	return "Ballot", subject, facts
}

// parsePipeRecord reads the `verb|field|field` shapes. The verbs are the ones
// the token, NFT and voting services actually write; see
// internal/domain/{token,nft} and internal/app/votingapp.
func parsePipeRecord(payload string) (module, action, subject string, facts []recordFact) {
	if !strings.Contains(payload, "|") {
		return "", "", "", nil
	}
	parts := strings.Split(payload, "|")
	// The NFT services write `<operation-uuid>|<verb>` for a transfer or a
	// burn. Without this branch those records rendered as a bare UUID followed
	// by a word, which named no module at all.
	if len(parts) == 2 && isUUID(parts[0]) {
		switch parts[1] {
		case "transfer", "burn":
			return "nft", titleAction(parts[1]), "", nil
		}
	}
	fields := readableFields(parts[1:])
	verb := parts[0]
	switch verb {
	case "mint", "transfer", "transfer_from", "transferfrom", "burn", "approve":
		return tokenRecord(verb, fields)
	case "mint_nft", "transfer_nft", "burn_nft":
		return nftRecord(verb, fields)
	case "vote":
		return voteRecord(fields)
	}
	// An NFT's own mint payload has no verb: `<id>|<name>|<owner key>`, the
	// id being a UUID rather than a token symbol.
	if len(parts) >= 3 && isUUID(parts[0]) {
		all := readableFields(parts)
		subject := all[1]
		facts = appendFact(facts, "Name", subject)
		facts = appendFact(facts, "Owner", all[2])
		return "nft", "Mint", subject, facts
	}
	return "", "", "", nil
}

// tokenRecord reads a token write. The pipe shape is `<verb>|<token>|<from>
// [|<to>]`: mint carries only the recipient, transfer carries both ends.
func tokenRecord(verb string, fields []string) (module, action, subject string, facts []recordFact) {
	action = titleAction(verb)
	if len(fields) == 0 {
		return "token", action, "", nil
	}
	subject = fields[0]
	facts = appendFact(facts, "Token", subject)
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
	return "token", action, subject, facts
}

// nftRecord reads an NFT write; its first readable field is the asset's name.
func nftRecord(verb string, fields []string) (module, action, subject string, facts []recordFact) {
	action = titleAction(strings.TrimSuffix(verb, "_nft"))
	if len(fields) > 0 {
		subject = fields[0]
	}
	facts = appendFact(facts, "NFT", subject)
	return "nft", action, subject, facts
}

// voteRecord reads a cast ballot: who voted, for whom, in which session.
func voteRecord(fields []string) (module, action, subject string, facts []recordFact) {
	subject = fieldAt(fields, 1)
	facts = appendFact(facts, "Candidate", subject)
	facts = appendFact(facts, "Voter", fieldAt(fields, 0))
	facts = appendFact(facts, "Session", fieldAt(fields, 2))
	return "voting", "Ballot", subject, facts
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

// fieldAt reads one pipe field, tolerating a short payload rather than
// indexing past its end.
func fieldAt(fields []string, index int) string {
	if index < 0 || index >= len(fields) {
		return ""
	}
	return fields[index]
}

// appendFact adds a labelled value, skipping an empty one so a record never
// shows a label with nothing beside it.
func appendFact(facts []recordFact, label, value string) []recordFact {
	value = strings.TrimSpace(value)
	if value == "" {
		return facts
	}
	return append(facts, recordFact{label: label, value: value})
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

// recordSearchText is what the adapter's search compares against. It includes
// the decoded sentences so a query for "transfer" or a winner's name finds the
// record even though the raw payload spells neither that way.
func recordSearchText(row nativeui.BlockRow) string {
	parts := []string{
		row.Module, row.Action, row.Subject, row.Summary, row.Payload,
		strconv.FormatInt(row.Height, 10),
	}
	for _, fact := range row.Facts {
		parts = append(parts, fact.Label, fact.Value)
	}
	return strings.Join(parts, " ")
}

// recordSearchMatches reports whether a record answers a search query. The
// query is matched case-insensitively against the record's decoded text, its
// raw payload and its height.
func recordSearchMatches(row nativeui.BlockRow, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(recordSearchText(row)), query)
}

// decodePipePayload turns a `verb|field|field` audit payload into readable
// text. Fields that are not printable are shown as base64; anything that does
// not look like that shape is returned unchanged.
func decodePipePayload(payload string) string {
	if !strings.Contains(payload, "|") {
		return payload
	}
	parts := strings.Split(payload, "|")
	if len(parts) < 2 {
		return payload
	}
	switch parts[0] {
	case "mint", "transfer", "transfer_from", "transferfrom", "approve", "burn":
		return "token " + strings.Join(readableFields(parts[1:]), " ")
	case "mint_nft", "transfer_nft", "burn_nft":
		return "nft " + strings.Join(readableFields(parts[1:]), " ")
	case "vote":
		return "voting " + strings.Join(readableFields(parts[1:]), " ")
	}
	// An NFT's own mint payload has no verb: `<id>|<name>|<owner key>`, the
	// id being a UUID rather than a token symbol.
	if len(parts) >= 3 && isUUID(parts[0]) {
		return "nft " + strings.Join(readableFields(parts), " ")
	}
	return payload
}

// readableFields renders each pipe field as text, or as base64 when it carries
// bytes a terminal cannot show.
func readableFields(fields []string) []string {
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

// isUUID reports whether s is a canonical 8-4-4-4-12 UUID.
func isUUID(s string) bool {
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
