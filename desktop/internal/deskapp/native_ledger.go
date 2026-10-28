package deskapp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// --- Dashboard / ledger --------------------------------------------------

func (s *nativeServices) Dashboard() (nativeui.DashboardStats, error) {
	stats := nativeui.DashboardStats{}

	records, err := s.lotteryRepo.GetAll()
	if err != nil {
		return stats, err
	}
	stats.Lotteries = len(records)

	cands, err := s.votingRepo.ListCandidates(0, 0)
	if err != nil {
		return stats, err
	}
	stats.Candidates = len(cands)
	votes := 0
	for _, c := range cands {
		votes += c.VoteCount
	}
	stats.Votes = votes

	sessions, err := s.votingRepo.ListSessions(0, 0)
	if err != nil {
		return stats, err
	}
	stats.Sessions = len(sessions)

	sources, err := s.oracleRepo.ListSources(0, 0)
	if err != nil {
		return stats, err
	}
	stats.Oracle = len(sources)

	if s.chain != nil && s.chain.VerifyIntegrity().Valid {
		stats.Integrity = "verified"
	} else {
		stats.Integrity = "broken"
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

// LedgerBlocks returns the recorded audit blocks, newest first. The payload is
// each module's own JSON, so the module is read from its "module" or
// "type" field and the summary is the payload shortened for a cell.
// LedgerBlocks returns up to limit recorded audit blocks, newest first (limit
// <= 0 for all), with the height each block actually has. The payload is each
// module's own JSON, so the module is read from its "module" or "type" field
// and the summary is the payload shortened for a cell.
func (s *nativeServices) LedgerBlocks(limit int) ([]nativeui.BlockRow, error) {
	return s.LedgerBlocksPage(limit, 0)
}

// LedgerBlocksPage reads one page of audit entries; offset is in entries
// (newest first), so a long ledger does not build every row to show the next
// page.
func (s *nativeServices) LedgerBlocksPage(limit, offset int) ([]nativeui.BlockRow, error) {
	if s.chain == nil {
		return nil, fmt.Errorf("blockchain not initialised")
	}
	entries := s.chain.BlockEntriesPage(limit, offset)
	rows := make([]nativeui.BlockRow, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, nativeui.BlockRow{
			Height:  entry.Height,
			Module:  blockModuleFor(entry.Data),
			Summary: blockSummary(entry.Data),
		})
	}
	return rows, nil
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
	return "—"
}

// blockModuleFor names the module of any payload shape, including the ones
// that are not JSON. It is what the ledger's Module column shows.
func blockModuleFor(payload string) string {
	if name := blockModule(payload); name != "—" {
		return name
	}
	switch {
	case strings.HasPrefix(blockSummary(payload), "token "):
		return "token"
	case strings.HasPrefix(blockSummary(payload), "nft "):
		return "nft"
	}
	return "—"
}

// blockSummary renders a payload for a table cell: JSON is compacted, and the
// pipe-delimited token/NFT payload is decoded field by field so its binary key
// bytes read as base64 rather than as unprintable bytes.
func blockSummary(payload string) string {
	const keep = 96
	text := decodePipePayload(payload)
	flat := strings.Join(strings.Fields(text), " ")
	if len(flat) <= keep {
		return flat
	}
	return flat[:keep] + "…"
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
	case "mint", "transfer", "transfer_from", "approve", "burn":
		return "token " + strings.Join(readableFields(parts[1:]), " ")
	case "mint_nft", "transfer_nft", "burn_nft":
		return "nft " + strings.Join(readableFields(parts[1:]), " ")
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
