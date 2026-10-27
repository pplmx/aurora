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

type fakeIntegrity struct{ report *ledger.IntegrityReport }

func (f fakeIntegrity) VerifyIntegrity() *ledger.IntegrityReport { return f.report }
