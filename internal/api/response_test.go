package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := NewSuccessResponse(data)

	assert.Nil(t, resp.Error)
	assert.Equal(t, data, resp.Data)
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("TEST_CODE", "test message")

	assert.Nil(t, resp.Data)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "TEST_CODE", resp.Error.Code)
	assert.Equal(t, "test message", resp.Error.Message)
}

func TestResponse_JSONSerialization(t *testing.T) {
	resp := NewSuccessResponse(map[string]int{"count": 42})

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)

	data, ok := parsed["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(42), data["count"])
}

func TestResponse_JSONSerialization_Error(t *testing.T) {
	resp := NewErrorResponse("ERR_CODE", "error message")

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)

	errorInfo, ok := parsed["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "ERR_CODE", errorInfo["code"])
	assert.Equal(t, "error message", errorInfo["message"])
}
