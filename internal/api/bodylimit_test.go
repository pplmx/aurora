package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
)

// TestBodyLimit_RejectsOversizedPayloadOverHTTP boots the real server wiring
// (NewServer + router, rate limiting off by default) and proves the BodyLimit
// middleware caps the JSON body at the handler boundary: an oversized
// `participants` payload is rejected with 413 BODY_TOO_LARGE, while a normal
// small body reaches the validator and fails with its own status (400), not
// with 413 — so legitimate requests are unaffected by the cap (v1.71,
// ISS-077).
func TestBodyLimit_RejectsOversizedPayloadOverHTTP(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "smoke-lottery"
	viper.Set("api.key", apiKey)

	dbPath := blockchain.DBPath()
	require.NotEmpty(t, dbPath)
	migPath, err := realMigrationsDirSmoke()
	require.NoError(t, err)
	m, err := migrate.New(dbPath, migPath)
	require.NoError(t, err, "migrator should init")
	_, err = m.Up(0)
	require.NoError(t, err, "applying real migrations must not fail")
	_ = m.Close()

	srv, err := NewServer()
	require.NoError(t, err, "NewServer should boot against a temp DB")
	t.Cleanup(func() { _ = srv.Close() })
	router := srv.Router()

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	// A small, well-formed-but-invalid body must reach the validator: 400
	// from the domain validation, never a 413 from the body cap.
	small := post(`{"participants":"","seed":"s","winner_count":0}`)
	require.Equal(t, http.StatusBadRequest, small.Code, "small body must not trip the body cap; got %d: %s", small.Code, small.Body.String())

	// An oversized body (way past MaxRequestBody) is cut off by the cap and
	// surfaces as 413 BODY_TOO_LARGE.
	big := post(`{"participants":"` + strings.Repeat("x", 5<<20) + `","seed":"s","winner_count":1}`)
	require.Equal(t, http.StatusRequestEntityTooLarge, big.Code, "oversized body must get 413; got %d: %s", big.Code, big.Body.String())
	require.Contains(t, big.Body.String(), "BODY_TOO_LARGE")
}
