package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// TestAPIDocsListEveryRoute keeps docs/api.md and the router in step. The API
// reference is the contract integrators read: a handler that ships without its
// row is invisible, and a row left behind after a route moves sends readers to
// a 404. The router is walked without a database, because route registration
// does not need one.
func TestAPIDocsListEveryRoute(t *testing.T) {
	mux, ok := newRouter(&Server{}).(chi.Routes)
	require.True(t, ok, "the gateway router must be a chi router")

	registered := make(map[string]bool)
	require.NoError(t, chi.Walk(mux,
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			// Handle registers an endpoint for every method; the reference
			// lists the metrics endpoints once, as GET.
			if strings.Contains(route, "/metrics") && method != http.MethodGet {
				return nil
			}
			registered[method+" "+route] = true
			return nil
		}))

	documented := documentedAPIRoutes(t)
	for route := range documented {
		require.Truef(t, registered[route],
			"docs/api.md documents %s, but the router does not register it", route)
	}
	for route := range registered {
		require.Truef(t, documented[route],
			"the router registers %s, but docs/api.md does not document it", route)
	}
}

var apiDocsRow = regexp.MustCompile("^\\|\\s*(GET|POST|PUT|DELETE|PATCH)\\s*\\|\\s*`([^`]+)`")

// readAPIDocs returns the reference text. The test file's own location keeps
// the repository path stable for `go test`.
func readAPIDocs(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "the test must know where it lives")
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "api.md"))
	require.NoError(t, err)
	return string(data)
}

// documentedAPIRoutes reads the method and path cells of the reference tables.
func documentedAPIRoutes(t *testing.T) map[string]bool {
	t.Helper()
	routes := make(map[string]bool)
	for _, line := range strings.Split(readAPIDocs(t), "\n") {
		if m := apiDocsRow.FindStringSubmatch(line); m != nil {
			routes[m[1]+" "+m[2]] = true
		}
	}
	require.NotEmpty(t, routes, "the reference tables must have rows")
	return routes
}

// jsonFence matches a fenced json block. Two newlines are not needed: the
// content is everything up to the closing fence.
var jsonFence = regexp.MustCompile("(?s)```json\n(.*?)```")

// TestAPIDocJSONExamplesParse keeps the reference's samples usable: a stray
// comma or a hand-trimmed field would make every copy-paste fail. Only the
// fenced json blocks are checked; the abbreviated hashes stay as written.
func TestAPIDocJSONExamplesParse(t *testing.T) {
	blocks := jsonFence.FindAllStringSubmatch(readAPIDocs(t), -1)
	require.GreaterOrEqual(t, len(blocks), 3,
		"the reference should carry response examples, otherwise this proves nothing")
	for i, block := range blocks {
		var value any
		require.NoErrorf(t, json.Unmarshal([]byte(block[1]), &value),
			"example %d in docs/api.md does not parse:\n%s", i+1, block[1])
	}
}
