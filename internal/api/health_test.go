package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLivenessHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	LivenessHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestHealthResponse(t *testing.T) {
	resp := HealthResponse{
		Status: "ok",
		Checks: map[string]string{"database": "ok"},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "ok", parsed["status"])
}

func TestHealthResponseWithChecks(t *testing.T) {
	resp := HealthResponse{
		Status: "ok",
		Checks: map[string]string{
			"database": "ok",
			"cache":    "ok",
		},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed HealthResponse
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "ok", parsed.Status)
	assert.Equal(t, "ok", parsed.Checks["database"])
	assert.Equal(t, "ok", parsed.Checks["cache"])
}

func TestHealthResponseEmptyChecks(t *testing.T) {
	resp := HealthResponse{
		Status: "healthy",
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed HealthResponse
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", parsed.Status)
	assert.Nil(t, parsed.Checks)
}
