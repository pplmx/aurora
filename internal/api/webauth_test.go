package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectKeyIntoHTML_BeforeHead(t *testing.T) {
	body := []byte("<html><head><title>x</title></head><body>hi</body></html>")
	out := injectKeyIntoHTML(body, "my-secret", "abc123")

	s := string(out)
	assert.Contains(t, s, `window.AURORA_API_KEY = "my-secret";`)
	assert.Contains(t, s, `<script nonce="abc123">`, "bootstrap script must carry the CSP nonce")
	headIdx := strings.Index(s, "</head>")
	scriptIdx := strings.Index(s, "window.AURORA_API_KEY")
	assert.True(t, scriptIdx < headIdx, "bootstrap should be injected before </head>")
}

func TestInjectKeyIntoHTML_NoHead(t *testing.T) {
	out := injectKeyIntoHTML([]byte("<html><body>hi</body></html>"), "k", "n0")
	assert.Contains(t, string(out), `window.AURORA_API_KEY = "k";`)
	assert.Contains(t, string(out), `<script nonce="n0">`)
}

func TestInjectAPIKey_InjectIntoHTML_NotIntoCSS(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"),
		[]byte("<html><head><title>t</title></head><body>hello</body></html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{color:red}"), 0o644))

	handler := injectAPIKey(http.FileServer(http.Dir(dir)), "top-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `window.AURORA_API_KEY = "top-secret";`)
	assert.Contains(t, rr.Body.String(), "hello")

	req = httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "body{color:red}", rr.Body.String())
	assert.NotContains(t, rr.Body.String(), "AURORA_API_KEY")
}

// TestInjectAPIKey_SetsCSPOnHTML locks the round-151 hardening: every HTML
// document the gateway serves must carry a Content-Security-Policy whose
// script nonce matches the injected bootstrap script, and non-HTML assets
// (which carry no inline script and no key) must not. The strict
// connect-src 'self' + script-src 'self' 'nonce-...' set means a knocked-in
// XSS payload cannot load remote script or exfiltrate the embedded
// window.AURORA_API_KEY to an external origin.
func TestInjectAPIKey_SetsCSPOnHTML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"),
		[]byte("<html><head><title>t</title></head><body>hello</body></html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o644))

	handler := injectAPIKey(http.FileServer(http.Dir(dir)), "top-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'", "CSP must default to same-origin")
	assert.Contains(t, csp, "connect-src 'self'", "CSP must block external exfiltration (connect-src)")
	assert.Contains(t, csp, "script-src 'self' 'nonce-", "CSP must nonce the bootstrap inline script")
	assert.Contains(t, csp, "frame-ancestors 'none'", "CSP must forbid framing (with X-Frame-Options)")

	// The nonce in the header must be the SAME nonce as the injected script,
	// else the browser rejects the bootstrap and every page call 401s.
	nonceRE := regexp.MustCompile(`'nonce-([A-Za-z0-9]+)'`)
	m := nonceRE.FindStringSubmatch(csp)
	require.NotNil(t, m, "CSP must carry a nonce", csp)
	assert.Contains(t, rr.Body.String(), `<script nonce="`+m[1]+`">`,
		"injected bootstrap must carry the CSP header's nonce")

	// A non-HTML asset gets no CSP (nothing inline to protect there).
	req = httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Content-Security-Policy"),
		"non-HTML assets carry no CSP (set only with the key-bootstrap injection)")
}

func TestInjectAPIKey_EmptyKey_PassesThrough(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"),
		[]byte("<html><head></head><body>x</body></html>"), 0o644))

	handler := injectAPIKey(http.FileServer(http.Dir(dir)), "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotContains(t, rr.Body.String(), "AURORA_API_KEY")
}
