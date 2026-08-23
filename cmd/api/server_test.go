package main

import (
	"net/http"
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
	if srv.Addr == "" {
		t.Error("Addr must be propagated")
	}
	_ = time.Second // keep time import intent explicit via a bounded check below
	if readHeaderTimeout < 1*time.Second || readTimeout < 1*time.Second {
		t.Error("configured timeouts are unreasonably small")
	}
}
