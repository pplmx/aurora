package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

// TestApplySourceHeaders guards the functional contract that persisted source
// request headers are actually applied to the outgoing request (previously the
// Headers field was silently ignored — only the transport defaults were set).
// This test exercises the pure helper without any loopback networking.
func TestApplySourceHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/data", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	err = applySourceHeaders(req, `{"Authorization": "Bearer token123", "X-Custom": "value"}`)
	if err != nil {
		t.Fatalf("applySourceHeaders failed: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer token123")
	}
	if got := req.Header.Get("X-Custom"); got != "value" {
		t.Errorf("X-Custom = %q, want %q", got, "value")
	}
}

func TestApplySourceHeaders_Empty(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/data", nil)
	if err := applySourceHeaders(req, ""); err != nil {
		t.Fatalf("empty headers should be a no-op, got error: %v", err)
	}
	if err := applySourceHeaders(req, "   "); err != nil {
		t.Fatalf("whitespace headers should be a no-op, got error: %v", err)
	}
}

func TestApplySourceHeaders_InvalidJSON(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/data", nil)
	err := applySourceHeaders(req, "not-json")
	if err == nil {
		t.Fatal("expected an error for invalid headers JSON")
	}
	// Invalid header config must surface as an invalid-source error, not a
	// silent drop of the operator's intent.
	if !errors.Is(err, oracle.ErrInvalidSource) {
		t.Errorf("expected an error wrapping oracle.ErrInvalidSource, got %v", err)
	}
}
