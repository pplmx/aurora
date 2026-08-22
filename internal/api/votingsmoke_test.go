package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
)

// realMigrationsDir resolves the repository migrations/ directory from this
// source file, independent of the process cwd (resetForAPITest chdirs into a
// temp dir).
func realMigrationsDirSmoke() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", assert.AnError
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")), nil
}

// TestVotingSmoke_FullFlowOverHTTP boots the REAL server wiring (NewServer +
// router + SQLite via a temp DB) and drives the complete voting domain over
// the HTTP API, mirroring exactly what web/voting.html does:
//
//	register voter -> register candidate -> create session -> list sessions
//	-> cast signed vote -> tally reflects the vote.
//
// This is the v1.8 "boot the real API + Web UI and drive voting end-to-end"
// item. It guards the API contract the browser page depends on against
// regressions that unit tests with mock repos cannot catch (real SQLite, real
// Ed25519 signing, real tx manager).
func TestVotingSmoke_FullFlowOverHTTP(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "smoke-test-key"
	viper.Set("api.key", apiKey)

	// NewServer opens ./data/aurora.db relative to the cwd, which
	// resetForAPITest redirected to a temp dir. It does NOT run migrations,
	// so apply the real checkout migrations first (mirrors `aurora migrate
	// up` that the CLI runs before serving).
	dbPath := blockchain.DBPath()
	require.NotEmpty(t, dbPath, "DBPath should resolve to a writable temp data dir")
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
	base := "/api/v1/voting"

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, base+path,
			bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, base+path, nil)
		req.Header.Set("X-API-Key", apiKey)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	// Protects against an unauthenticated regression: without the key the
	// API must reject.
	unauth := httptest.NewRequest(http.MethodGet, base+"/candidates", nil)
	unauthRR := httptest.NewRecorder()
	router.ServeHTTP(unauthRR, unauth)
	assert.Equal(t, http.StatusUnauthorized, unauthRR.Code, "candidates requires API key")

	// 1. Register a voter (returns keypair).
	rr := post("/register/voter", `{"name":"Alice"}`)
	require.Equal(t, http.StatusOK, rr.Code, "register voter body: %s", rr.Body.String())
	var voter struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &voter))
	require.NotEmpty(t, voter.PublicKey)
	require.NotEmpty(t, voter.PrivateKey)

	// 2. Register a candidate.
	rr = post("/register/candidate", `{"name":"Bob","party":"Party A","program":"Platform"}`)
	require.Equal(t, http.StatusOK, rr.Code, "register candidate body: %s", rr.Body.String())
	var candidate struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &candidate))
	require.NotEmpty(t, candidate.ID)
	assert.Equal(t, "Bob", candidate.Name)

	// 3. Create a session spanning now, with Bob as the roster.
	now := time.Now().Unix()
	rr = post("/session", `{"title":"Election","description":"Smoke","candidate_ids":["`+
		candidate.ID+`"],"start_time":`+itoa(now-60)+`,"end_time":`+itoa(now+3600)+`}`)
	require.Equal(t, http.StatusOK, rr.Code, "create session body: %s", rr.Body.String())
	var session struct {
		ID     string   `json:"id"`
		Status string   `json:"status"`
		Cands  []string `json:"candidates"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &session))
	require.NotEmpty(t, session.ID)
	assert.Contains(t, session.Cands, candidate.ID)

	// 4. List sessions returns the new session.
	rr = get("/sessions")
	require.Equal(t, http.StatusOK, rr.Code)
	var sessions []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &sessions))
	assert.True(t, len(sessions) >= 1, "sessions should be listable")
	found := false
	for _, s := range sessions {
		if s.ID == session.ID {
			found = true
		}
	}
	assert.True(t, found, "newly created session must appear in list")

	// 5. Cast a signed vote.
	voteBody := `{"voter_public_key":"` + voter.PublicKey +
		`","candidate_id":"` + candidate.ID +
		`","private_key":"` + voter.PrivateKey +
		`","session_id":"` + session.ID + `"}`
	rr = post("/vote", voteBody)
	require.Equal(t, http.StatusOK, rr.Code, "cast vote body: %s", rr.Body.String())
	var vote struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &vote))
	assert.NotEmpty(t, vote.ID)

	// 6. Tally reflects the vote via the candidates endpoint.
	rr = get("/candidates")
	require.Equal(t, http.StatusOK, rr.Code)
	var cands []struct {
		ID        string `json:"id"`
		VoteCount int    `json:"vote_count"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &cands))
	for _, c := range cands {
		if c.ID == candidate.ID {
			assert.Equal(t, 1, c.VoteCount, "candidate tally should include the cast vote")
		}
	}

	// 7. Double-voting the same voter must be rejected (already-voted).
	rr = post("/vote", voteBody)
	assert.NotEqual(t, http.StatusOK, rr.Code, "same voter must not vote twice")

	// 8. Session lifecycle over the real API (v1.10): register a second voter,
	// end the session, confirm a NEW vote is rejected while "ended", then start
	// the session and confirm the locked-out voter can now vote.
	rr = post("/register/voter", `{"name":"Carol"}`)
	require.Equal(t, http.StatusOK, rr.Code, "register second voter body: %s", rr.Body.String())
	var voter2 struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &voter2))

	vote2Body := `{"voter_public_key":"` + voter2.PublicKey +
		`","candidate_id":"` + candidate.ID +
		`","private_key":"` + voter2.PrivateKey +
		`","session_id":"` + session.ID + `"}`

	// End the session and verify status flips to "ended".
	rr = post("/session/"+session.ID+"/end", "")
	require.Equal(t, http.StatusOK, rr.Code, "end session body: %s", rr.Body.String())
	var ended struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ended))
	assert.Equal(t, "ended", ended.Status)

	// A NEW vote from a fresh voter must be rejected on an ended session.
	rr = post("/vote", vote2Body)
	assert.NotEqual(t, http.StatusOK, rr.Code, "vote must be rejected on ended session")

	// Start it again: status -> active and the locked-out voter can now vote.
	rr = post("/session/"+session.ID+"/start", "")
	require.Equal(t, http.StatusOK, rr.Code, "start session body: %s", rr.Body.String())
	var started struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &started))
	assert.Equal(t, "active", started.Status)

	rr = post("/vote", vote2Body)
	require.Equal(t, http.StatusOK, rr.Code, "vote after restart body: %s", rr.Body.String())
}

func itoa(n int64) string {
	b, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(b)
}
