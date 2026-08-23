package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
)

// TestLotterySmoke_CreateThenVerifyOverHTTP boots the real server wiring
// (NewServer + router + SQLite via a temp DB) and drives the v1.31 audit flow
// end-to-end over HTTP: create a lottery (comma-separated participants +
// winner_count, the contract the web page sends) then verify it via
// GET /api/v1/lottery/{id}/verify, which must report valid.
func TestLotterySmoke_CreateThenVerifyOverHTTP(t *testing.T) {
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

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	create := post("/api/v1/lottery/create",
		`{"participants":"Alice,Bob,Charlie","seed":"e2e-lottery-seed","winner_count":2}`)
	require.Equal(t, http.StatusOK, create.Code, "create body: %s", create.Body.String())
	var created struct {
		ID      string   `json:"id"`
		Winners []string `json:"winners"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	require.NotEmpty(t, created.ID)
	require.Len(t, created.Winners, 2)

	verify := get("/api/v1/lottery/" + created.ID + "/verify")
	require.Equal(t, http.StatusOK, verify.Code, "verify body: %s", verify.Body.String())
	var report struct {
		Valid bool `json:"valid"`
	}
	require.NoError(t, json.Unmarshal(verify.Body.Bytes(), &report))
	require.True(t, report.Valid, "a freshly created lottery must verify as valid")
}
