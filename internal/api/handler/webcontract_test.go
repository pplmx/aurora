package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	domainnft "github.com/pplmx/aurora/internal/domain/nft"
	domainoracle "github.com/pplmx/aurora/internal/domain/oracle"
)

// webContractGet parses web/index.html, web/lottery.html, web/voting.html and
// web/js/app.js for /api/v1/... endpoint references and returns them keyed by
// "METHOD path". This guards against the regression AUDITED in v1.7: the
// voting page silently POSTed to /api/v1/voting/create (404) while the page
// model drifted from the real domain. If a page references an endpoint the
// API does not serve, the test fails instead of the feature failing at runtime.
func webContractGet(t *testing.T) map[string]bool {
	t.Helper()
	// Tests run with cwd = internal/api/handler. web/ lives three levels up.
	root := filepath.Join("..", "..", "..", "web")
	files := []string{
		filepath.Join(root, "index.html"),
		filepath.Join(root, "lottery.html"),
		filepath.Join(root, "voting.html"),
		filepath.Join(root, "token.html"),
		filepath.Join(root, "oracle.html"),
		filepath.Join(root, "blockchain.html"),
		filepath.Join(root, "nft.html"),
		filepath.Join(root, "js", "app.js"),
	}

	methodRe := regexp.MustCompile(`method\s*:\s*'([A-Z]+)'`)
	// Match single-quoted '/api/v1/...' literals used in fetch() calls.
	pathRe := regexp.MustCompile(`'(/api/v1/[^']+)'`)

	got := map[string]bool{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err, "read %s", f)
		text := string(data)
		// apiFetch is the shared fetch wrapper (round-97 TASK-124); normalize
		// it to fetch( so the boundary split and nearest-method heuristic below
		// treat helper-wrapped calls exactly like direct calls.
		text = strings.ReplaceAll(text, "apiFetch(", "fetch(")
		// Segment the file by fetch( boundaries so a path maps to the
		// nearest preceding method. Insert a sentinel before each fetch call.
		segments := strings.Split(text, "fetch(")
		for i, seg := range segments {
			// Default to GET, then look for an explicit method in this segment.
			method := "GET"
			if m := methodRe.FindStringSubmatch(seg); m != nil {
				method = m[1]
			}
			// Only the leading fetch's path belongs to this segment; split at
			// the next fetch( boundary so we don't over-capture.
			head := seg
			if n := strings.Index(seg, "fetch("); n >= 0 {
				head = seg[:n]
			}
			if m := pathRe.FindStringSubmatch(head); m != nil {
				got[method+" "+normWebPath(m[1])] = true
			} else if i == 0 {
				// The first segment is everything before any fetch(); also
				// scan it in case an inline call is present without "fetch(".
				if m := pathRe.FindStringSubmatch(seg); m != nil {
					got[method+" "+normWebPath(m[1])] = true
				}
			}
		}
	}

	if len(got) == 0 {
		t.Fatal("no /api/v1 endpoints found in web/ — contract test is vacuous")
	}
	return got
}

// normWebPath strips the query string from a web-referenced /api/v1 path so
// the contract match compares against the route path only (same-origin fetch
// calls build queries like '/api/v1/token/history?token_id=').
func normWebPath(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		return p[:i]
	}
	return p
}

func buildWebRouter() chi.Router {
	r := chi.NewRouter()
	nft := NewNFTHandler(domainnft.NewInmemRepo(), nil)
	voting := NewVotingHandler(fakeVotingRepo{}, nil)
	token := NewTokenHandler(fakeTokenServiceFull{})
	oracle := NewOracleHandler(domainoracle.NewInmemRepo())
	lottery := NewLotteryHandler(nil)
	blockchain := NewBlockchainHandler()

	r.Route("/api/v1/lottery", func(r chi.Router) { lottery.Routes(r) })
	r.Route("/api/v1/voting", func(r chi.Router) { voting.Routes(r) })
	r.Route("/api/v1/nft", func(r chi.Router) { nft.Routes(r) })
	r.Route("/api/v1/token", func(r chi.Router) { token.Routes(r) })
	r.Route("/api/v1/oracle", func(r chi.Router) { oracle.Routes(r) })
	r.Route("/api/v1/blockchain", func(r chi.Router) { blockchain.Routes(r) })
	return r
}

func TestWebUIContract_AllReferencedEndpointsResolve(t *testing.T) {
	router := buildWebRouter()
	refs := webContractGet(t)

	for key := range refs {
		parts := strings.Fields(key)
		require.Len(t, parts, 2, "malformed ref %q", key)
		method, path := parts[0], parts[1]

		rctx := chi.NewRouteContext()
		matched := router.Match(rctx, method, path)
		if !matched && strings.HasSuffix(path, "/") {
			// Pages build dynamic paths like '/api/v1/voting/results/' + id; the
			// captured literal is everything up to the interpolation, so probe
			// with a param segment to resolve {id} routes.
			rctx = chi.NewRouteContext()
			matched = router.Match(rctx, method, path+"1")
		}
		require.Truef(t, matched,
			"web UI calls %s %s but no API route matches (silent 404 regression)",
			method, path)
	}
}

func TestWebUIContract_VotingReferencesRealEndpoints(t *testing.T) {
	refs := webContractGet(t)
	// These are the real registered voting routes; pages may use a subset.
	require.True(t, refs["POST /api/v1/voting/register/voter"], "voting page must register voters via real endpoint")
	require.True(t, refs["POST /api/v1/voting/register/candidate"], "voting page must register candidates via real endpoint")
	require.True(t, refs["POST /api/v1/voting/session"], "voting page must create sessions via real endpoint")
	require.True(t, refs["POST /api/v1/voting/vote"], "voting page must cast votes via real endpoint")
	require.True(t, refs["GET /api/v1/voting/sessions"], "voting page must list sessions via real endpoint")
	require.True(t, refs["GET /api/v1/voting/candidates"], "voting page must list candidates via real endpoint")
}
