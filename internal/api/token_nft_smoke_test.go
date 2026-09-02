package api

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
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

// TestTokenNFT_Smoke_ApproveAllowanceAndNFTKeyBinding boots the REAL server
// wiring (NewServer + router + SQLite via a temp DB, migrations applied) and
// exercises the v1.9/v1.10 parity + security surfaces over HTTP:
//
//	token: create -> approve -> allowance reflects it  (and approve with a
//	       mismatched key is rejected)
//	nft:   mint -> transfer with a forged (mismatched) private key is rejected
//
// This is the v1.11 "E2E coverage for token approve/allowance + NFT key-binding
// rejection" item.
func TestTokenNFT_Smoke_ApproveAllowanceAndNFTKeyBinding(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "smoke-token-nft"
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
	require.NotNil(t, srv)
	t.Cleanup(func() { _ = srv.Close() })
	router := srv.Router()

	request := func(method, base string, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, base, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	// ---------- TOKEN: create, approve, allowance ----------
	ownerPub, ownerPriv := newKeyPairB64(t)

	// create token
	rr := request(http.MethodPost, "/api/v1/token/create", fmt.Sprintf(
		`{"name":"MTK","symbol":"MTK","total_supply":"1000000","owner":"%s"}`, ownerPub))
	require.Equal(t, http.StatusOK, rr.Code, "create token body: %s", rr.Body.String())
	var tok struct {
		ID          string `json:"id"`
		Owner       string `json:"owner"`
		TotalSupply string `json:"total_supply"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tok))
	require.NotEmpty(t, tok.ID)
	require.Equal(t, ownerPub, tok.Owner)
	require.Equal(t, "1000000", tok.TotalSupply)

	spenderPub, spenderPriv := newKeyPairB64(t)

	// approve a correct allowance with the owner's key
	rr = request(http.MethodPost, "/api/v1/token/approve", fmt.Sprintf(
		`{"token_id":"%s","owner":"%s","spender":"%s","amount":"250","private_key":"%s"}`,
		tok.ID, ownerPub, spenderPub, ownerPriv))
	require.Equal(t, http.StatusOK, rr.Code, "approve body: %s", rr.Body.String())
	var appr struct {
		ID     string `json:"id"`
		Owner  string `json:"owner"`
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &appr))
	require.NotEmpty(t, appr.ID)
	require.Equal(t, ownerPub, appr.Owner)
	require.Equal(t, "250", appr.Amount)

	// allowance reflects it
	rr = request(http.MethodGet, fmt.Sprintf("/api/v1/token/allowance?token_id=%s&owner=%s&spender=%s",
		url.QueryEscape(tok.ID), url.QueryEscape(ownerPub), url.QueryEscape(spenderPub)), "")
	require.Equal(t, http.StatusOK, rr.Code, "allowance body: %s", rr.Body.String())
	var allow struct {
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &allow))
	require.Equal(t, "250", allow.Amount)

	// ---------- TOKEN: transfer_from (spender spends the allowance, v1.38) ----------
	toPub, _ := newKeyPairB64(t)
	rr = request(http.MethodPost, "/api/v1/token/transfer_from", fmt.Sprintf(
		`{"token_id":"%s","owner":"%s","to":"%s","amount":"100","spender":"%s","spender_key":"%s"}`,
		tok.ID, ownerPub, toPub, spenderPub, spenderPriv))
	require.Equal(t, http.StatusOK, rr.Code, "transfer_from body: %s", rr.Body.String())

	// The allowance must be drawn down (250 - 100 = 150).
	rr = request(http.MethodGet, fmt.Sprintf("/api/v1/token/allowance?token_id=%s&owner=%s&spender=%s",
		url.QueryEscape(tok.ID), url.QueryEscape(ownerPub), url.QueryEscape(spenderPub)), "")
	require.Equal(t, http.StatusOK, rr.Code)
	var allowAfter struct {
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &allowAfter))
	require.Equal(t, "150", allowAfter.Amount)

	// The recipient must hold the transferred 100.
	rr = request(http.MethodGet, fmt.Sprintf("/api/v1/token/balance?token_id=%s&owner=%s",
		url.QueryEscape(tok.ID), url.QueryEscape(toPub)), "")
	require.Equal(t, http.StatusOK, rr.Code, "to-balance body: %s", rr.Body.String())
	var toBal struct {
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &toBal))
	require.Equal(t, "100", toBal.Amount)

	// ---------- Read paths on a NONEXISTENT token must 404 (ISS-250) ----------
	// balance/allowance/history previously read a typo'd token_id back as a
	// legitimate zero balance / empty history (`200 {"amount":"0"}` / `200 []`)
	// while /token/info 404'd — the shared requireToken guard now puts the
	// read surfaces on the same unknown-resource->404 contract.
	for _, path := range []string{
		"/api/v1/token/balance?token_id=MTK_TYPO&owner=" + url.QueryEscape(ownerPub),
		"/api/v1/token/allowance?token_id=MTK_TYPO&owner=" + url.QueryEscape(ownerPub) + "&spender=" + url.QueryEscape(spenderPub),
		"/api/v1/token/history?token_id=MTK_TYPO&owner=" + url.QueryEscape(ownerPub),
	} {
		rr = request(http.MethodGet, path, "")
		require.Equal(t, http.StatusNotFound, rr.Code,
			"GET %s on a nonexistent token must 404, got %d: %s", path, rr.Code, rr.Body.String())
		require.Contains(t, rr.Body.String(), "TOKEN_NOT_FOUND",
			"GET %s must return the TOKEN_NOT_FOUND code, body: %s", path, rr.Body.String())
	}

	// approve with a MISMATCHED key must be rejected (401)
	_, attackerPriv := newKeyPairB64(t)
	rr = request(http.MethodPost, "/api/v1/token/approve", fmt.Sprintf(
		`{"token_id":"%s","owner":"%s","spender":"%s","amount":"250","private_key":"%s"}`,
		tok.ID, ownerPub, spenderPub, attackerPriv))
	require.Equal(t, http.StatusUnauthorized, rr.Code,
		"approve with mismatched key must be rejected, got %d: %s", rr.Code, rr.Body.String())

	// ---------- NFT: mint then forged transfer is rejected ----------
	creatorPub, creatorPriv := newKeyPairB64(t)
	recipientPub, _ := newKeyPairB64(t)

	rr = request(http.MethodPost, "/api/v1/nft/mint", fmt.Sprintf(
		`{"name":"Aurora NFT","description":"smoke","creator":"%s"}`, creatorPub))
	require.Equal(t, http.StatusOK, rr.Code, "mint body: %s", rr.Body.String())
	var nft struct {
		ID    string `json:"id"`
		Owner string `json:"owner"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &nft))
	require.NotEmpty(t, nft.ID)

	// Forge a transfer: use the victim's public key as `from` but a DIFFERENT
	// private key -> must be rejected (401 KEY_MISMATCH).
	_, forgerPriv := newKeyPairB64(t)
	rr = request(http.MethodPost, "/api/v1/nft/transfer", fmt.Sprintf(
		`{"nft_id":"%s","from":"%s","to":"%s","private_key":"%s"}`,
		nft.ID, creatorPub, recipientPub, forgerPriv))
	require.Equal(t, http.StatusUnauthorized, rr.Code,
		"forged NFT transfer must be rejected, got %d: %s", rr.Code, rr.Body.String())

	// A legitimate transfer with the real owner key succeeds.
	rr = request(http.MethodPost, "/api/v1/nft/transfer", fmt.Sprintf(
		`{"nft_id":"%s","from":"%s","to":"%s","private_key":"%s"}`,
		nft.ID, creatorPub, recipientPub, creatorPriv))
	require.Equal(t, http.StatusOK, rr.Code, "legit transfer body: %s", rr.Body.String())
}

// newKeyPairB64 generates an Ed25519 keypair and returns base64-encoded public
// and private keys (matching how the CLI/API pass keys around).
func newKeyPairB64(t *testing.T) (pubB64, privB64 string) {
	t.Helper()
	pub, priv, err := randomKeyPair()
	require.NoError(t, err)
	return encB64(pub), encB64(priv)
}

func randomKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func encB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
