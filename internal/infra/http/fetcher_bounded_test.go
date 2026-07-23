package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

// TestReadBounded proves the body-size cap rejects oversized payloads
// without buffering them. Without readBounded, a malicious oracle
// source could OOM the process by streaming a multi-GB body that
// io.ReadAll would happily buffer into RAM.
func TestReadBounded(t *testing.T) {
	t.Run("UnderLimit", func(t *testing.T) {
		got, err := readBounded(strings.NewReader("hello"), 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("got %q, want %q", string(got), "hello")
		}
	})

	t.Run("ExactlyAtLimit", func(t *testing.T) {
		// Exactly max bytes is allowed (>= not >).
		body := strings.Repeat("x", 100)
		got, err := readBounded(strings.NewReader(body), 100)
		if err != nil {
			t.Fatalf("unexpected error at exact limit: %v", err)
		}
		if len(got) != 100 {
			t.Errorf("got %d bytes, want 100", len(got))
		}
	})

	t.Run("OverLimit", func(t *testing.T) {
		body := strings.Repeat("x", 101)
		_, err := readBounded(strings.NewReader(body), 100)
		if !errors.Is(err, ErrResponseTooLarge) {
			t.Errorf("got %v, want ErrResponseTooLarge", err)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		got, err := readBounded(strings.NewReader(""), 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(got))
		}
	})
}

// TestFetcher_Get_OversizedResponseRejected proves the bounded read is
// wired into the public Get path. The server streams a body larger
// than maxResponseBytes; Get must return ErrResponseTooLarge without
// buffering the whole response.
func TestFetcher_Get_OversizedResponseRejected(t *testing.T) {
	// Build a body that is definitely larger than maxResponseBytes,
	// but stream it in small chunks so we don't allocate 10+ MiB in
	// the test process.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		chunk := make([]byte, 64*1024)
		for i := range chunk {
			chunk[i] = 'x'
		}
		// Write maxResponseBytes + a bit more so the bounded reader
		// must trip.
		writes := (maxResponseBytes / len(chunk)) + 2
		for i := 0; i < writes; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	fetcher := NewFetcher()
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("expected error for oversized response, got nil")
	}
	if !errors.Is(err, ErrResponseTooLarge) &&
		!strings.Contains(err.Error(), "response body exceeds maximum size") {
		t.Errorf("expected ErrResponseTooLarge, got: %v", err)
	}
}

// TestFetcher_FetchData_RejectsOversized proves the FetchData path used
// by oracle ingestion also enforces the size cap, so a hostile source
// can never write a multi-GB row into oracle_data.
func TestFetcher_FetchData_RejectsOversized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate payload lazily to avoid wasting test memory.
		_, _ = w.Write([]byte(`{"big":"`))
		chunk := strings.Repeat("x", 1024)
		for i := 0; i < maxResponseBytes/1024+1; i++ {
			_, _ = w.Write([]byte(chunk))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		_, _ = w.Write([]byte(`"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	_, err := fetcher.FetchData(&oracle.DataSource{
		ID:   "test-source",
		URL:  server.URL,
		Type: "json",
	})
	if err == nil {
		t.Fatal("expected error for oversized response, got nil")
	}
	if !strings.Contains(err.Error(), "response body exceeds maximum size") {
		t.Errorf("expected ErrResponseTooLarge in error chain, got: %v", err)
	}
}
