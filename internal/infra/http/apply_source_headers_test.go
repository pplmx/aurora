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

// TestApplySourceHeaders_RejectsCRLF pins TASK-239/ISS-237: Go JSON escape
// sequences ("\r\n") decode to literal CR/LF bytes, so a header name or value
// could smuggle extra header lines toward the upstream provider. Modern
// net/http refuses to write such a request (a confusing runtime fetch error),
// so applySourceHeaders must reject CR/LF up front as an invalid source.
func TestApplySourceHeaders_RejectsCRLF(t *testing.T) {
	for _, tc := range []struct {
		name, json string
	}{
		{"value CRLF", `{"X-Test": "v\r\nInjected: 1"}`},
		{"name CRLF", `{"X-Bad\r\nX-Injected": "v"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "https://example.com/data", nil)
			err := applySourceHeaders(req, tc.json)
			if !errors.Is(err, oracle.ErrInvalidSource) {
				t.Fatalf("CRLF header must be rejected as invalid source, got %v", err)
			}
		})
	}
}
