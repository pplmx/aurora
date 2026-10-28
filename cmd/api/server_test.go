package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewServer_SetsHTTPTimeouts locks the v1.60 hardening: the API HTTP
// server must configure non-zero connection timeouts so slowloris / slow-body
// / idle-keep-alive connections cannot hold handlers open indefinitely
// (ISS-065, TASK-073). No socket is bound — we only assert the configured
// http.Server fields.
func TestNewServer_SetsHTTPTimeouts(t *testing.T) {
	srv := newServer("127.0.0.1:8080", http.NewServeMux())

	if srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout = %v, want > 0 (slowloris protection)", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Errorf("ReadTimeout = %v, want > 0 (slow-body protection)", srv.ReadTimeout)
	}
	if srv.WriteTimeout <= 0 {
		t.Errorf("WriteTimeout = %v, want > 0", srv.WriteTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Errorf("IdleTimeout = %v, want > 0 (keep-alive reaping)", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != maxHeaderBytes {
		t.Errorf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, maxHeaderBytes)
	}
	if srv.Addr == "" {
		t.Error("Addr must be propagated")
	}
	_ = time.Second // keep time import intent explicit via a bounded check below
	if readHeaderTimeout < 1*time.Second || readTimeout < 1*time.Second {
		t.Error("configured timeouts are unreasonably small")
	}
}

// TestNewServer_RejectsOversizedRequestHeaders exercises the configured limit
// through the real HTTP server parser instead of only inspecting the struct
// field. The header arrives before routing or authentication, so leaving it at
// net/http's 1 MiB default lets an unauthenticated request force a much larger
// allocation than any legitimate API call needs.
func TestNewServer_RejectsOversizedRequestHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	ts := httptest.NewTestServer(t, handler)
	ts.Config = newServer("", handler)
	client := ts.Client()

	normalResp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = normalResp.Body.Close()
	if normalResp.StatusCode != http.StatusNoContent {
		t.Fatalf("normal request status = %d, want %d", normalResp.StatusCode, http.StatusNoContent)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Fill", strings.Repeat("x", 2*maxHeaderBytes))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusRequestHeaderFieldsTooLarge)
	}
}
