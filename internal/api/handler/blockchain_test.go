package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/ledger"
	"github.com/stretchr/testify/require"
)

// TestBlockchainVerifyRoute pins the audit surface's only endpoint: it answers
// the chain's own integrity report as JSON, so a client can distinguish a
// broken chain from a gateway error.
func TestBlockchainVerifyRoute(t *testing.T) {
	want := &ledger.IntegrityReport{Valid: true, Length: 2, FirstBrokenIndex: -1}
	router := chi.NewRouter()
	NewBlockchainHandler(fakeIntegrity{report: want}).Routes(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/verify", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var report ledger.IntegrityReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	require.Equal(t, want.Valid, report.Valid)
	require.Equal(t, want.Length, report.Length)

	// The route is a GET; another method is a routing error, not a verification.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/verify", nil))
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

type fakeIntegrity struct {
	report  *ledger.IntegrityReport
	entries []ledger.BlockEntry
}

func (f fakeIntegrity) VerifyIntegrity() *ledger.IntegrityReport { return f.report }

// Len counts the genesis block, which is why the endpoint's total is Len-1.
func (f fakeIntegrity) Len() int { return len(f.entries) + 1 }

func (f fakeIntegrity) BlockEntriesPage(limit, offset int) []ledger.BlockEntry {
	return f.page(limit, offset, func(ledger.BlockEntry) bool { return true })
}

func (f fakeIntegrity) BlockEntriesPageMatching(limit, offset int, match func(ledger.BlockEntry) bool) ([]ledger.BlockEntry, int) {
	total := 0
	for _, entry := range f.entries {
		if match(entry) {
			total++
		}
	}
	return f.page(limit, offset, match), total
}

func (f fakeIntegrity) page(limit, offset int, match func(ledger.BlockEntry) bool) []ledger.BlockEntry {
	matches := make([]ledger.BlockEntry, 0, len(f.entries))
	for _, entry := range f.entries {
		if match(entry) {
			matches = append(matches, entry)
		}
	}
	if offset >= len(matches) {
		return nil
	}
	page := matches[offset:]
	if limit > 0 && len(page) > limit {
		page = page[:limit]
	}
	return page
}

// TestBlockchainBlocksRoute pins the audit page: newest-first entries with the
// exact payload, an envelope carrying the total, and paging that does not
// mistake heights for entry positions.
// TestBlockchainBlocksSearchReachesTheDecodedRecord: q finds a word the decoded
// sentence shows even though the stored payload spells something else. The CLI's
// -s and the window's ledger search compare the same sentence.
func TestBlockchainBlocksSearchReachesTheDecodedRecord(t *testing.T) {
	chain := fakeIntegrity{
		report: &ledger.IntegrityReport{Valid: true},
		entries: []ledger.BlockEntry{
			{Height: 1, Data: "transfer|GLD|alice|bob"},
			{Height: 2, Data: "plain payload"},
		},
	}
	router := chi.NewRouter()
	NewBlockchainHandler(chain).Routes(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?q=token", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var page BlockPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Equal(t, 1, page.Total, "the decoded module name is searchable")
	require.Len(t, page.Blocks, 1)
	require.Equal(t, int64(1), page.Blocks[0].Height)
	require.Equal(t, "transfer|GLD|alice|bob", page.Blocks[0].Payload,
		"the document still carries the exact payload")
}

func TestBlockchainBlocksRoute(t *testing.T) {
	chain := fakeIntegrity{
		report: &ledger.IntegrityReport{Valid: true},
		entries: []ledger.BlockEntry{
			{Height: 2, Timestamp: 1_700_000_002, Data: `{"module":"nft","name":"Star"}`},
			{Height: 1, Timestamp: 1_700_000_001, Data: "seed|draw|2"},
		},
	}
	router := chi.NewRouter()
	NewBlockchainHandler(chain).Routes(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?limit=1&offset=1", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	requireSnakeCaseJSON(t, rec.Body.Bytes())

	var page BlockPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Equal(t, 2, page.Total, "total excludes the genesis block")
	require.Equal(t, 1, page.Limit)
	require.Equal(t, 1, page.Offset)
	require.Len(t, page.Blocks, 1)
	require.Equal(t, int64(1), page.Blocks[0].Height)
	require.Equal(t, "seed|draw|2", page.Blocks[0].Payload)

	// An unparsable page size falls back to the default instead of failing.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?limit=abc", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"limit":20`)

	// q filters on the recorded text and the total counts matches.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?q=STAR", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var filtered BlockPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &filtered))
	require.Equal(t, 1, filtered.Total)
	require.Len(t, filtered.Blocks, 1)
	require.Equal(t, int64(2), filtered.Blocks[0].Height)
}
