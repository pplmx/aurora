package deskapp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/ledger"
)

// --- Dashboard / ledger --------------------------------------------------

func (s *nativeServices) Dashboard() (nativeui.DashboardStats, error) {
	stats := nativeui.DashboardStats{}

	// The dashboard refreshes while the window is open. Exact SQL aggregates
	// avoid decoding every draw, candidate, session and feed into Go rows just
	// to take their lengths.
	lotteries, err := s.lotteryRepo.Count()
	if err != nil {
		return stats, err
	}
	stats.Lotteries = lotteries

	votingCounts, err := s.votingRepo.Counts()
	if err != nil {
		return stats, err
	}
	stats.Candidates = votingCounts.Candidates
	stats.Votes = votingCounts.Votes
	stats.Sessions = votingCounts.Sessions

	oracleSources, err := s.oracleRepo.CountSources()
	if err != nil {
		return stats, err
	}
	stats.Oracle = oracleSources

	if s.chain != nil && s.chain.VerifyIntegrity().Valid {
		stats.Integrity = "verified"
	} else {
		stats.Integrity = "broken"
	}
	if s.chain != nil {
		// Len includes the genesis block, which is why the audit page shows the
		// same number as its block count: the dashboard counts chain entries,
		// not records the modules wrote.
		stats.Blocks = s.chain.Len()
	}
	return stats, nil
}

func (s *nativeServices) LedgerStatus() (nativeui.LedgerStatus, error) {
	if s.chain == nil {
		return nativeui.LedgerStatus{}, fmt.Errorf("blockchain not initialised")
	}
	return nativeui.LedgerStatus{
		// Len takes the chain's read lock; reading the Blocks slice directly
		// raced with a concurrent AddBlock on a busy API.
		Valid:  s.chain.VerifyIntegrity().Valid,
		Height: s.chain.Len() - 1,
		Blocks: s.chain.Len(),
	}, nil
}

// LedgerBlocks returns up to limit recorded audit blocks, newest first (limit
// <= 0 for all), with the height each block actually has. The payload is each
// module's own JSON, so the module is read from its "module" or "type" field
// and the summary is the payload shortened for a cell.
func (s *nativeServices) LedgerBlocks(limit int) ([]nativeui.BlockRow, error) {
	page, err := s.LedgerBlocksPage("", limit, 0)
	return page.Rows, err
}

// LedgerBlocksPage reads one newest-first page of audit entries and counts all
// entries matching the same query. A non-empty search is evaluated against the
// whole chain while only the requested page is retained.
func (s *nativeServices) LedgerBlocksPage(search string, limit, offset int) (nativeui.LedgerBlockPage, error) {
	if s.chain == nil {
		return nativeui.LedgerBlockPage{}, fmt.Errorf("blockchain not initialised")
	}
	search = strings.TrimSpace(search)
	if search == "" {
		entries := s.chain.BlockEntriesPage(limit, offset)
		total := s.chain.Len() - 1
		if total < 0 {
			total = 0
		}
		return nativeui.LedgerBlockPage{Rows: ledgerBlockRows(entries), Total: total}, nil
	}

	query := strings.ToLower(search)
	entries, total := s.chain.BlockEntriesPageMatching(limit, offset, func(entry ledger.BlockEntry) bool {
		module, text := blockFields(entry.Data)
		return strings.Contains(strings.ToLower(module), query) ||
			strings.Contains(strings.ToLower(text), query) ||
			strings.Contains(strconv.FormatInt(entry.Height, 10), query)
	})
	return nativeui.LedgerBlockPage{Rows: ledgerBlockRows(entries), Total: total}, nil
}

func ledgerBlockRows(entries []ledger.BlockEntry) []nativeui.BlockRow {
	rows := make([]nativeui.BlockRow, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, ledgerBlockRow(entry))
	}
	return rows
}

func ledgerBlockRow(entry ledger.BlockEntry) nativeui.BlockRow {
	module, text := blockFields(entry.Data)
	return nativeui.BlockRow{
		Height:  entry.Height,
		Module:  module,
		Summary: clipBlockText(text),
	}
}

// blockModule names the module a recorded payload belongs to.
//
// Two payload shapes are on the chain. Lottery and voting blocks are JSON and
// name themselves; token and NFT blocks are `verb|field|field` with raw
// public-key bytes, so they are recognised by their verb. Rendering the raw
// bytes printed mojibake in the ledger table (measured).
func blockModule(payload string) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		return "—"
	}
	for _, key := range []string{"module", "type", "kind"} {
		if raw, ok := fields[key]; ok {
			var name string
			if json.Unmarshal(raw, &name) == nil && name != "" {
				return name
			}
		}
	}
	// A lottery record names itself by its fields.
	if _, ok := fields["participants"]; ok {
		return "lottery"
	}
	// Oracle observations are JSON too, but identify their source rather than
	// naming a module.
	if _, ok := fields["source_id"]; ok {
		return "oracle"
	}
	return "—"
}

// blockModuleFor names the module of any payload shape, including the ones
// that are not JSON. It is what the ledger's Module column shows.
func blockModuleFor(payload string) string {
	module, _ := blockFields(payload)
	return module
}

// blockFields derives the rendered module and the full flattened search text
// in one pass. The table clips the text, but search must still reach a payload
// whose distinctive value lies beyond the visible prefix.
func blockFields(payload string) (module, text string) {
	text = blockText(payload)
	if name := blockModule(payload); name != "—" {
		return name, text
	}
	switch {
	case strings.HasPrefix(text, "token "):
		return "token", text
	case strings.HasPrefix(text, "nft "):
		return "nft", text
	case strings.HasPrefix(text, "voting "):
		return "voting", text
	}
	return "—", text
}

// blockSummary renders a payload for a table cell: JSON is compacted, and the
// pipe-delimited token/NFT payload is decoded field by field so its binary key
// bytes read as base64 rather than as unprintable bytes.
func blockSummary(payload string) string {
	_, text := blockFields(payload)
	return clipBlockText(text)
}

func blockText(payload string) string {
	text := decodePipePayload(payload)
	return strings.Join(strings.Fields(text), " ")
}

// clipBlockText shortens display text without splitting a UTF-8 code point.
func clipBlockText(text string) string {
	const keep = 96
	runes := []rune(text)
	if len(runes) <= keep {
		return text
	}
	return string(runes[:keep]) + "…"
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
