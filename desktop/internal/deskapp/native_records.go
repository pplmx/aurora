package deskapp

import (
	"strconv"
	"strings"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/records"
)

// The ledger's record language.
//
// Every module writes its own payload shape: lottery and oracle records are
// JSON that can be decoded field by field, while token, NFT and voting records
// are a `verb|field|field` line carrying base64 or raw key bytes. Rendering
// those verbatim was the old ledger: a table of JSON blobs and pipe-delimited
// key bytes, which is an audit trail that nobody can read.
//
// internal/records owns that vocabulary — which module wrote a payload, what it
// did, the one object it names and the facts worth reading — so the CLI's record
// list and this page describe the same payload the same way. This file maps the
// vocabulary onto the view and keeps the display-only helpers: the clipped
// summary and the search text.

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
	if record, ok := records.Parse(payload); ok {
		row.Module, row.Action, row.Subject = record.Module, record.Action, record.Subject
		for _, fact := range record.Facts {
			row.Facts = append(row.Facts, nativeui.BlockFact{Label: fact.Label, Value: fact.Value})
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
		return "token " + strings.Join(records.ReadableFields(parts[1:]), " ")
	case "mint_nft", "transfer_nft", "burn_nft":
		return "nft " + strings.Join(records.ReadableFields(parts[1:]), " ")
	case "vote":
		return "voting " + strings.Join(records.ReadableFields(parts[1:]), " ")
	}
	// An NFT's own mint payload has no verb: `<id>|<name>|<owner key>`, the
	// id being a UUID rather than a token symbol.
	if len(parts) >= 3 && records.IsUUID(parts[0]) {
		return "nft " + strings.Join(records.ReadableFields(parts), " ")
	}
	return payload
}
