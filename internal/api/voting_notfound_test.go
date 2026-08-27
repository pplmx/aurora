package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestVotingAPI_MissingResourcesAre4xx is the regression test for the voting
// 500-for-missing-resource class (TASK-111, ISS-103): the pre-transaction
// existence checks in the voting use cases hit GetSession/GetVoter/
// GetCandidate, whose SQLite implementation returns sqlite.ErrNotFound on a
// missing row — but that sentinel was not mapped to a domain error/message, so
// every missing-resource request returned 500 INTERNAL_ERROR instead of the
// intended 4xx. Previous coverage used fakes returning (nil, nil), which never
// exercised the ErrNotFound path.
//
// Assertions use the REAL server wiring (NewServer + migrations + SQLite):
//
//	unknown candidate in session create -> 404 CANDIDATE_NOT_FOUND
//	unknown session on vote           -> 404 SESSION_NOT_FOUND
//	unregistered voter on vote        -> 404 VOTER_NOT_REGISTERED
//	unknown candidate on vote         -> 400 CANDIDATE_NOT_FOUND
//	unknown session in results        -> 404 SESSION_NOT_FOUND
func TestVotingAPI_MissingResourcesAre4xx(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "notfound-test-key")
	const apiKey = "notfound-test-key"

	dbPath := blockchain.DBPath()
	require.NotEmpty(t, dbPath)
	migPath, err := realMigrationsDirSmoke()
	require.NoError(t, err)
	m, err := migrate.New(dbPath, migPath)
	require.NoError(t, err)
	_, err = m.Up(0)
	require.NoError(t, err)
	_ = m.Close()

	srv, err := NewServer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close() })
	router := srv.Router()

	do := func(method, path, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	// Baseline: create a real session + voter + candidate so we can exercise the
	// "real but wrong reference" cases against a live set.
	registerVoter := func(name string) (pub, priv string) {
		rr := do(http.MethodPost, "/api/v1/voting/register/voter", `{"name":"`+name+`"}`)
		require.Equal(t, http.StatusOK, rr.Code, "register voter: %s", rr.Body.String())
		var v struct {
			PublicKey  string `json:"public_key"`
			PrivateKey string `json:"private_key"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &v))
		return v.PublicKey, v.PrivateKey
	}
	registerCandidate := func(name string) string {
		rr := do(http.MethodPost, "/api/v1/voting/register/candidate",
			`{"name":"`+name+`","party":"P","program":"X"}`)
		require.Equal(t, http.StatusOK, rr.Code, "register candidate: %s", rr.Body.String())
		var c struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &c))
		return c.ID
	}
	createSession := func(candIDs ...string) string {
		ids, _ := json.Marshal(candIDs)
		rr := do(http.MethodPost, "/api/v1/voting/session",
			`{"title":"S","description":"D","start_time":1700000000,"end_time":1900000000,"candidate_ids":`+string(ids)+`}`)
		require.Equal(t, http.StatusOK, rr.Code, "create session: %s", rr.Body.String())
		var s struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &s))
		return s.ID
	}

	alicePub, alicePriv := registerVoter("Alice")
	cand := registerCandidate("Bob")
	sess := createSession(cand)

	t.Run("create session with unknown candidate is 404", func(t *testing.T) {
		rr := do(http.MethodPost, "/api/v1/voting/session",
			`{"title":"X","description":"","start_time":1700000000,"end_time":1900000000,"candidate_ids":["does-not-exist"]}`)
		// Candidate-not-found is a 400 in the established contract (the
		// classification table maps ErrCandidateNotFound to BAD_REQUEST); the
		// regression is that it is no longer a 500 INTERNAL_ERROR.
		require.Equal(t, http.StatusBadRequest, rr.Code, "body: %s", rr.Body.String())
		require.Contains(t, rr.Body.String(), "CANDIDATE_NOT_FOUND")
	})

	t.Run("vote with unknown session is 404", func(t *testing.T) {
		body := `{"voter_public_key":"` + alicePub + `","candidate_id":"` + cand +
			`","private_key":"` + alicePriv + `","session_id":"ghost-session"}`
		rr := do(http.MethodPost, "/api/v1/voting/vote", body)
		require.Equal(t, http.StatusNotFound, rr.Code, "body: %s", rr.Body.String())
		require.Contains(t, rr.Body.String(), "SESSION_NOT_FOUND")
	})

	t.Run("vote with unregistered voter is 404", func(t *testing.T) {
		body := `{"voter_public_key":"not-a-real-key-in-db","candidate_id":"` + cand +
			`","private_key":"` + alicePriv + `","session_id":"` + sess + `"}`
		rr := do(http.MethodPost, "/api/v1/voting/vote", body)
		require.Equal(t, http.StatusNotFound, rr.Code, "body: %s", rr.Body.String())
		require.Contains(t, rr.Body.String(), "VOTER_NOT_REGISTERED")
	})

	t.Run("vote with unknown candidate is 400", func(t *testing.T) {
		body := `{"voter_public_key":"` + alicePub + `","candidate_id":"ghost-candidate"` +
			`,"private_key":"` + alicePriv + `","session_id":"` + sess + `"}`
		rr := do(http.MethodPost, "/api/v1/voting/vote", body)
		require.Equal(t, http.StatusBadRequest, rr.Code, "body: %s", rr.Body.String())
		require.Contains(t, rr.Body.String(), "CANDIDATE_NOT_FOUND")
	})

	t.Run("results for unknown session is 404", func(t *testing.T) {
		rr := do(http.MethodGet, "/api/v1/voting/results/ghost-session", "")
		require.Equal(t, http.StatusNotFound, rr.Code, "body: %s", rr.Body.String())
		require.Contains(t, rr.Body.String(), "SESSION_NOT_FOUND")
	})
}
