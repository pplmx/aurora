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

// TestAPIRouteIndexListsEveryRoute keeps the index exhaustive. It is the
// document's first screen, so a route missing from it is a route a reader never
// learns exists — the module sections would still document it, but only someone
// who already knew the path would find it.
func TestAPIRouteIndexListsEveryRoute(t *testing.T) {
	mux, ok := newRouter(&Server{}).(chi.Routes)
	require.True(t, ok, "the gateway router must be a chi router")

	registered := make(map[string]bool)
	require.NoError(t, chi.Walk(mux,
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			// Handle registers one endpoint for every method; the index names
			// the metrics endpoints once, as GET.
			if strings.Contains(route, "/metrics") && method != http.MethodGet {
				return nil
			}
			registered[method+" "+route] = true
			return nil
		}))

	indexed := make(map[string]bool)
	for _, match := range apiIndexRow.FindAllStringSubmatch(readAPIDocs(t), -1) {
		indexed[match[1]+" "+match[2]] = true
	}
	require.NotEmpty(t, indexed, "the endpoint index must list routes")
	for route := range registered {
		require.Truef(t, indexed[route], "the endpoint index omits %s", route)
	}
	for route := range indexed {
		require.Truef(t, registered[route], "the endpoint index lists %s, which the router does not register", route)
	}
}

// apiIndexRow matches one bullet of the endpoint index: a method, a space and
// a path, each inside backticks. The multiline flag makes ^ and $ anchor per
// line, which is what a per-line bullet list needs.
var apiIndexRow = regexp.MustCompile("(?m)^- `(GET|POST|PUT|DELETE|PATCH) (/[^`]*)`$")

// TestAPIRouteIndexIsNotTheContract keeps the endpoint index from standing in
// for the module sections. The index lists every route once in bullet form; if
// it were ever rewritten as a table, documentedAPIRoutes would find every route
// there and TestAPIDocsListEveryRoute would pass without any module section
// documenting a single field.
func TestAPIRouteIndexIsNotTheContract(t *testing.T) {
	text := readAPIDocs(t)
	start := strings.Index(text, "## 端点索引")
	require.GreaterOrEqual(t, start, 0, "docs/api.md must keep its endpoint index")
	end := strings.Index(text[start:], "## 鉴权与中间件")
	require.Greater(t, end, 0, "the index must end before the auth section")
	index := text[start : start+end]
	require.NotContains(t, index, "| `GET`", "the index must stay a bullet list, not a route table")
	require.NotContains(t, index, "| `POST`", "the index must stay a bullet list, not a route table")
	require.NotEmpty(t, strings.TrimSpace(index), "the index must list the routes")
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

// TestAPIDocsDescribeAmountsAsWholeNumbers pins the sentence a client reads
// before sending a write. Amounts are parsed with token.NewAmountFromString,
// which takes a base-ten integer: a decimal or a grouped "1,000" is refused with
// INVALID_AMOUNT. The reference used to call them "decimal strings", so a client
// that followed it got a 400.
func TestAPIDocsDescribeAmountsAsWholeNumbers(t *testing.T) {
	docs := readAPIDocs(t)

	require.Falsef(t, strings.Contains(docs, "金额是十进制字符串"),
		"docs/api.md still calls amounts decimal strings; the parser takes base-ten integers")

	// The token section's paragraph, not the voting one that shares the opening
	// words.
	start := strings.Index(docs, "写接口的请求体字段（")
	require.GreaterOrEqual(t, start, 0, "the write-fields paragraph should exist")
	end := strings.Index(docs[start:], "\n\n")
	require.Greater(t, end, 0)
	paragraph := docs[start : start+end]

	require.Contains(t, paragraph, "整数字符串",
		"the paragraph should say what an amount is")
	require.Contains(t, paragraph, "INVALID_AMOUNT",
		"and name the code a rejected value produces")

	// The code side of the same fact, so the sentence cannot outlive the parser
	// it describes.
	entity, err := os.ReadFile(filepath.Join("..", "domain", "token", "entity.go"))
	require.NoError(t, err)
	require.Contains(t, string(entity), "new(big.Int).SetString(s, 10)",
		"a base-ten integer parse is what makes the paragraph accurate")
}
