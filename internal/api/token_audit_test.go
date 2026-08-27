package api

// Regression for ISS-080 (v1.73): the API server's event bus never subscribed
// an audit handler, so every token audit event the service published was
// silently dropped and GET /api/v1/token/history was always empty on the
// production HTTP path (the CLI wiring in app/wire.go subscribed it; cmd/api
// did not). Boots the real server wiring and asserts a transfer appears in
// history — proving the subscription exists end to end.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
)

func TestTokenAudit_TransferAppearsInHistoryOverHTTP(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "smoke-token-audit"
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

	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	ownerPub, ownerPriv := newKeyPairB64(t)
	recipientPub, _ := newKeyPairB64(t)

	// Create funded token.
	rr := request(http.MethodPost, "/api/v1/token/create", fmt.Sprintf(
		`{"name":"AUD","symbol":"AUD","total_supply":"1000","owner":"%s"}`, ownerPub))
	require.Equal(t, http.StatusOK, rr.Code, "create body: %s", rr.Body.String())

	// Transfer 10 owner -> recipient.
	rr = request(http.MethodPost, "/api/v1/token/transfer", fmt.Sprintf(
		`{"token_id":"AUD","from":"%s","to":"%s","amount":"10","private_key":"%s"}`,
		ownerPub, recipientPub, ownerPriv))
	require.Equal(t, http.StatusOK, rr.Code, "transfer body: %s", rr.Body.String())

	// History must show that one transfer (pre-fix it was always empty).
	u := "/api/v1/token/history?token_id=AUD&owner=" + url.QueryEscape(ownerPub) + "&limit=20"
	rr = request(http.MethodGet, u, "")
	require.Equal(t, http.StatusOK, rr.Code, "history body: %s", rr.Body.String())

	// /api/v1/token/history returns the bare array like every other list
	// endpoint (TASK-093, ISS-086) — not the old {"transfers":[...]} wrapper
	// the web UI could never consume.
	var resp []struct {
		ID          string `json:"id"`
		From        string `json:"from"`
		To          string `json:"to"`
		Amount      string `json:"amount"`
		BlockHeight int64  `json:"block_height"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp, 1, "the committed transfer must be persisted in history on the HTTP path")
	require.Equal(t, "10", resp[0].Amount)
	require.Equal(t, ownerPub, resp[0].From)
	require.Equal(t, recipientPub, resp[0].To)
	require.Positive(t, resp[0].BlockHeight)
}

// TestTokenCreate_DecimalsOverHTTP locks the v1.79 fix that the optional
// "decimals" JSON field on POST /api/v1/token/create is honored (TASK-099,
// ISS-091) — pre-fix every create silently stored the default of 8. An
// omitted field keeps the default.
func TestTokenCreate_DecimalsOverHTTP(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "smoke-token-decimals"
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

	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	ownerPub, _ := newKeyPairB64(t)

	// Explicit decimals=6 must be honored in the create response and `info`.
	rr := request(http.MethodPost, "/api/v1/token/create", fmt.Sprintf(
		`{"name":"GFX","symbol":"GFX","total_supply":"1000","owner":"%s","decimals":6}`, ownerPub))
	require.Equal(t, http.StatusOK, rr.Code, "create body: %s", rr.Body.String())
	var create struct {
		Decimals int8 `json:"decimals"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &create))
	require.Equal(t, int8(6), create.Decimals, "create response must report decimals 6")

	rr = request(http.MethodGet, "/api/v1/token/info?token_id=GFX", "")
	require.Equal(t, http.StatusOK, rr.Code, "info body: %s", rr.Body.String())
	var info struct {
		Decimals int8 `json:"decimals"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &info))
	require.Equal(t, int8(6), info.Decimals, "info must report the persisted decimals 6")

	// Omitted decimals field keeps the default of 8.
	rr = request(http.MethodPost, "/api/v1/token/create", fmt.Sprintf(
		`{"name":"DEF","symbol":"DEF","total_supply":"1000","owner":"%s"}`, ownerPub))
	require.Equal(t, http.StatusOK, rr.Code, "create body: %s", rr.Body.String())
	rr = request(http.MethodGet, "/api/v1/token/info?token_id=DEF", "")
	require.Equal(t, http.StatusOK, rr.Code, "info body: %s", rr.Body.String())
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &info))
	require.Equal(t, int8(8), info.Decimals, "omitted decimals must fall back to 8")
}
