package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVotingHandler_RegisterVoter_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_RegisterCandidate_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_CreateSession_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/session", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_Vote_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/vote", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Vote(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_Routes(t *testing.T) {
	handler := NewVotingHandler(nil)
	assert.NotNil(t, handler)
}