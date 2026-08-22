package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectKeyIntoHTML_BeforeHead(t *testing.T) {
	body := []byte("<html><head><title>x</title></head><body>hi</body></html>")
	out := injectKeyIntoHTML(body, "my-secret")

	s := string(out)
	assert.Contains(t, s, `window.AURORA_API_KEY = "my-secret";`)
	headIdx := strings.Index(s, "</head>")
	scriptIdx := strings.Index(s, "window.AURORA_API_KEY")
	assert.True(t, scriptIdx < headIdx, "bootstrap should be injected before </head>")
}

func TestInjectKeyIntoHTML_NoHead(t *testing.T) {
	out := injectKeyIntoHTML([]byte("<html><body>hi</body></html>"), "k")
	assert.Contains(t, string(out), `window.AURORA_API_KEY = "k";`)
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
