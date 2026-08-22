package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// bodyCapture records the status, headers and body of a downstream handler so
// the caller can post-process the response (used to inject the API key into
// served HTML before it is written to the client).
type bodyCapture struct {
	header http.Header
	status int
	buf    bytes.Buffer
}

func (c *bodyCapture) Header() http.Header { return c.header }
func (c *bodyCapture) WriteHeader(status int) {
	c.status = status
}
func (c *bodyCapture) Write(b []byte) (int, error) { return c.buf.Write(b) }

// injectAPIKey wraps the static web file server so that HTML responses embed
// the resolved API key as a window global (window.AURORA_API_KEY). The web UI
// is served by the same gateway and every /api/v1 call requires this key; a
// static file server cannot know it, so without injection every browser call
// returned 401. Non-HTML assets pass through untouched.
//
// Security note: this hands the key to the (same-origin) browser as a
// client-side credential — acceptable for the single-user dev web UI, which is
// served only by the gateway that owns the key. This is recorded as a decision
// in the RIL graph; production hardening (short-lived session tokens, CSRF
// protection) is future work.
func injectAPIKey(next http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !wantsHTML(r) {
			next.ServeHTTP(w, r)
			return
		}

		rec := &bodyCapture{header: make(http.Header)}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		body := rec.buf.Bytes()
		if isHTML(rec.header.Get("Content-Type")) {
			body = injectKeyIntoHTML(body, apiKey)
		}

		for k, vv := range rec.header {
			// The injected script changes the body length; drop the
			// downstream Content-Length and let the outer writer recompute it.
			if strings.EqualFold(k, "Content-Length") {
				continue
			}
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.status)
		_, _ = w.Write(body)
	})
}

// wantsHTML reports whether the request is likely for an HTML document.
func wantsHTML(r *http.Request) bool {
	p := r.URL.Path
	return p == "/" || strings.HasSuffix(p, ".html")
}

func isHTML(contentType string) bool {
	return strings.Contains(contentType, "text/html")
}

// injectKeyIntoHTML inserts the API-key bootstrap script just before </head>
// (falling back to prepending it) so the value is defined before the page's
// scripts run.
func injectKeyIntoHTML(body []byte, apiKey string) []byte {
	keyJSON, _ := json.Marshal(apiKey)
	script := []byte("<script>window.AURORA_API_KEY = " + string(keyJSON) + ";</script>")

	lower := bytes.ToLower(body)
	if i := bytes.Index(lower, []byte("</head>")); i >= 0 {
		out := make([]byte, 0, len(body)+len(script)+1)
		out = append(out, body[:i]...)
		out = append(out, script...)
		out = append(out, body[i:]...)
		return out
	}

	out := make([]byte, 0, len(body)+len(script)+1)
	out = append(out, script...)
	out = append(out, body...)
	return out
}
