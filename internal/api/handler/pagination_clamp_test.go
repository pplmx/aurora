package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/ledger"
)

// TestOutOfRangePagingIsClampedNotRejected covers the documented behaviour that
// an over-limit page request is silently reduced rather than refused, and that
// the response says which value took effect.
//
// The clamp is deliberate — a caller asking for 500 rows gets 100 instead of an
// error — but "silently" is only acceptable when the caller can find out. The
// envelope endpoints echo the effective limit; the bare-array endpoints cannot,
// which is why docs/api.md tells a client to keep its own request inside the
// ceiling or to compare the returned count with it.
func TestOutOfRangePagingIsClampedNotRejected(t *testing.T) {
	mkPage := func(query string) BlockPage {
		t.Helper()
		chain := fakeIntegrity{
			report: &ledger.IntegrityReport{Valid: true},
			entries: []ledger.BlockEntry{
				{Height: 1, Data: "a"}, {Height: 2, Data: "b"},
			},
		}
		router := chi.NewRouter()
		NewBlockchainHandler(chain).Routes(router)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?"+query, nil))
		require.Equal(t, http.StatusOK, rec.Code, "an over-limit page must not be an error")
		var page BlockPage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		return page
	}

	// An over-limit request answers 200 and reports the value it applied.
	page := mkPage("limit=9999")
	assert.Equal(t, maxBlockPageLimit, page.Limit,
		"the response must echo the limit it applied, not the one that was asked for")

	page = mkPage("offset=999999")
	assert.Equal(t, maxBlockPageOffset, page.Offset,
		"the response must echo the offset it applied")

	// A negative value is treated as unset rather than as an error.
	page = mkPage("limit=-5&offset=-5")
	assert.Equal(t, defaultBlockPageLimit, page.Limit, "a negative limit falls back to the default")
	assert.Equal(t, 0, page.Offset, "a negative offset falls back to zero")

	// Garbage is unset too.
	page = mkPage("limit=abc&offset=xyz")
	assert.Equal(t, defaultBlockPageLimit, page.Limit)
	assert.Equal(t, 0, page.Offset)

	// A value inside the range is used as given, so the echo is not just the
	// constant in every case.
	page = mkPage("limit=1&offset=1")
	assert.Equal(t, 1, page.Limit)
	assert.Equal(t, 1, page.Offset)
}
