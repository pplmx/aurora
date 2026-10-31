package cmd

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/netguard"
)

// runCmd executes a subcommand through rootCmd (the real CLI path) with
// the given args, capturing stdout. rootCmd.PersistentPreRunE runs only the
// optional migrate.autoRun hook — the app.Wire composition root that used to
// run here was removed in v1.80 (it stashed a never-read GlobalApp and created
// a phantom $HOME/.aurora/data; TASK-103, ISS-095) and its dead
// internal/app/Wire code was retired in v1.82 (TASK-115, ISS-107). These tests
// exercise the command bodies without touching the database, so we neutralise
// the remaining migrate hook here.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	prev := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = nil
	t.Cleanup(func() { rootCmd.PersistentPreRunE = prev })

	// Cobra's required-flag validation relies on each flag's Changed bit,
	// which persists across Execute calls within this process. Reset the
	// whole tree before every invocation so one command's flags cannot
	// satisfy another command's required-flag check.
	resetFlags(rootCmd)

	capture := captureStdout(t)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	out := capture()
	// Reset again: a test that invokes a command's RunE directly bypasses this
	// helper, and a leftover --json value would send it down the document path
	// with nowhere to print.
	resetFlags(rootCmd)
	return out, err
}

func TestOracleSourceAdd_List_Delete(t *testing.T) {
	resetCliForTest()

	out, err := runCmd(t, "oracle", "source", "add", "--name", "btc", "--url", "https://api.example.com/price")
	require.NoError(t, err, "source add should succeed")
	assert.Contains(t, out, "Data source created")
	assert.Contains(t, out, "btc")

	// second source, different name
	out, err = runCmd(t, "oracle", "source", "add", "--name", "eth", "--url", "https://api.example.com/eth")
	require.NoError(t, err)
	assert.Contains(t, out, "eth")

	// list shows both; a source without a path omits the field instead of
	// printing "Path: " with nothing after it.
	out, err = runCmd(t, "oracle", "source", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "btc")
	assert.Contains(t, out, "eth")
	assert.NotContains(t, out, "Path:", "a source without a path must not print the label")

	// delete one, list again
	list, _ := runCmd(t, "oracle", "source", "list")
	ids := extractSourceIDs(t, list)
	require.GreaterOrEqual(t, len(ids), 2, "expected at least 2 sources listed")

	out, err = runCmd(t, "oracle", "source", "delete", "--id", ids[0], "--confirm")
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")

	out, err = runCmd(t, "oracle", "source", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, ids[0])
	assert.Contains(t, out, ids[1])

	// delete a non-existent id now fails loudly: DeleteSource reports not-found
	// (mirroring enable/disable and the REST 404) instead of a silent success
	// that lied about the delete (TASK-233, ISS-231).
	_, err = runCmd(t, "oracle", "source", "delete", "--id", "no-such-id", "--confirm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source not found")

	// Without --confirm the delete is refused (non-zero exit).
	_, err = runCmd(t, "oracle", "source", "delete", "--id", "no-such-id")
	require.Error(t, err, "source delete without --confirm must be refused")
	assert.Contains(t, err.Error(), "--confirm")
}

func TestOracleSourceEnable_Disable(t *testing.T) {
	resetCliForTest()

	out, err := runCmd(t, "oracle", "source", "add", "--name", "gas", "--url", "https://api.example.com/gas")
	require.NoError(t, err)
	id := extractSourceIDs(t, out)
	require.Len(t, id, 1, "source add should print the new id")

	_, err = runCmd(t, "oracle", "source", "disable", "--id", id[0])
	require.NoError(t, err)

	// disabled source fetch -> ErrSourceDisabled (pre-network)
	_, err = runCmd(t, "oracle", "fetch", "--id", id[0])
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "disabled")

	_, err = runCmd(t, "oracle", "source", "enable", "--id", id[0])
	require.NoError(t, err)

	// benign: fetch now attempts a real HTTP call via http.NewFetcher; the
	// URL is unreachable so it errors after a network attempt. We only assert
	// an error path here and cover the enable/disable happy path above.
	out, err = runCmd(t, "oracle", "source", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "enabled")
}

func TestOracleSourceList_Pages(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		for _, name := range []string{"alpha", "beta", "gamma"} {
			_, err := runCmd(t, "oracle", "source", "add",
				"--name", name, "--url", "https://api.example.com/"+name)
			require.NoError(t, err)
		}

		out, err := runCmd(t, "oracle", "source", "list", "--limit", "1", "--offset", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Data Sources: 1 (offset 1)")
		assert.Equal(t, 1, strings.Count(out, "   - "), "only the requested page should print")

		_, err = runCmd(t, "oracle", "source", "list", "--limit", "-1")
		require.Error(t, err)
		_, err = runCmd(t, "oracle", "source", "list", "--offset", "-1")
		require.Error(t, err)
	})
}

func TestOracleSourceAdd_ValidationErrors(t *testing.T) {
	resetCliForTest()

	// Missing required flag -> cobra validation error before RunE.
	_, err := runCmd(t, "oracle", "source", "add")
	require.Error(t, err)
}

func TestOracleSourceAdd_PersistsHeaders(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "oracle", "source", "add",
			"--name", "private-feed",
			"--url", "https://api.example.com/private",
			"--headers", `{"Authorization":"Bearer secret","X-API-Key":"feed-key"}`,
		)
		require.NoError(t, err)
		ids := extractSourceIDs(t, out)
		require.Len(t, ids, 1)

		list, err := runCmd(t, "oracle", "source", "list")
		require.NoError(t, err)
		assert.Contains(t, list, "Headers: configured")

		repo, cleanup, err := newOracleRepo()
		require.NoError(t, err)
		defer cleanup()

		source, err := repo.GetSource(ids[0])
		require.NoError(t, err)
		require.NotNil(t, source)
		assert.JSONEq(t, `{"Authorization":"Bearer secret","X-API-Key":"feed-key"}`, source.Headers)
	})
}

func TestOracleSourceAdd_RejectsInvalidHeaders(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "oracle", "source", "add",
			"--name", "private-feed",
			"--url", "https://api.example.com/private",
			"--headers", "not-json",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid headers json")

		repo, cleanup, err := newOracleRepo()
		require.NoError(t, err)
		defer cleanup()
		sources, err := repo.ListSources(0, 0)
		require.NoError(t, err)
		assert.Empty(t, sources)
	})
}

func TestOracleSourceAdd_RejectsInvalidHTTPConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "method", args: []string{"--method", "BAD METHOD"}},
		{name: "header name", args: []string{"--headers", `{"Bad Header":"value"}`}},
		{name: "header value", args: []string{"--headers", `{"X-Test":"bad\u0000value"}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withTempDir(t, func(t *testing.T) {
				args := append([]string{
					"oracle", "source", "add",
					"--name", "private-feed",
					"--url", "https://api.example.com/private",
				}, tc.args...)
				_, err := runCmd(t, args...)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid source")

				repo, cleanup, err := newOracleRepo()
				require.NoError(t, err)
				defer cleanup()
				sources, err := repo.ListSources(0, 0)
				require.NoError(t, err)
				assert.Empty(t, sources)
			})
		})
	}
}

func TestOracleFetch_SourceNotFound(t *testing.T) {
	resetCliForTest()

	// No sources at all: fetch of unknown id returns ErrSourceNotFound
	// before any network call.
	_, err := runCmd(t, "oracle", "fetch", "--id", "missing")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")
}

func TestOracleData_Latest(t *testing.T) {
	resetCliForTest()

	// Unknown sources are rejected consistently across `data` and `latest`
	// (ISS-130). Historically `data` silently returned "(none)" for an unknown
	// source while `latest` surfaced an unclassified error; both now surface
	// the ~source not found~ sentinel (404 on the REST surface).
	_, err := runCmd(t, "oracle", "data", "--id", "missing", "--limit", "5")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")

	_, err = runCmd(t, "oracle", "latest", "--id", "missing")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")
}

// templateAddHostBlocked reports whether this environment resolves the template
// host into private space, which is what a sandbox that maps external DNS to a
// local proxy does. The SSRF guard then rejects the template URL by design, so
// the success assertion cannot hold; a normal network resolves it publicly and
// the tests run their real assertions. Unresolvable hosts are not "blocked"
// (the guard allows them; the fetch would fail at dial time instead).
func templateAddHostBlocked(t *testing.T) bool {
	t.Helper()
	ips, err := net.LookupIP("api.coingecko.com")
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if netguard.IsBlockedIP(ip) {
			return true
		}
	}
	return false
}

func skipWhenTemplateHostIsBlocked(t *testing.T) {
	t.Helper()
	if templateAddHostBlocked(t) {
		t.Skip("this environment resolves api.coingecko.com to a private address; the SSRF guard rejects the template URL by design")
	}
}

func TestOracleTemplateList_Add(t *testing.T) {
	resetCliForTest()

	out, err := runCmd(t, "oracle", "template", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "btc-price")
	assert.Contains(t, out, "eth-price")

	out, err = runCmd(t, "oracle", "template", "list", "--json")
	require.NoError(t, err)
	var templates []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &templates), "got %q", out)
	ids := make([]string, 0, len(templates))
	for _, template := range templates {
		ids = append(ids, template.ID)
	}
	assert.Contains(t, ids, "btc-price")
	assert.Contains(t, ids, "eth-price")

	// unknown template -> clean error, before any URL is validated
	_, err = runCmd(t, "oracle", "template", "add", "--template", "no-such-template")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")

	// template add for a known template -> source created. The skip only
	// applies to this last step: the list and unknown-template assertions above
	// have already run.
	skipWhenTemplateHostIsBlocked(t)
	out, err = runCmd(t, "oracle", "template", "add", "--template", "btc-price")
	require.NoError(t, err)
	assert.Contains(t, out, "Bitcoin Price")
}

// extractSourceIDs pulls every "ID: <uuid>" line out of command output.
func extractSourceIDs(t *testing.T, out string) []string {
	t.Helper()
	var ids []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "ID: "); ok {
			ids = append(ids, after)
		}
	}
	return ids
}

// TestOracleTemplateAdd_JSON: creating a source from a template returns the
// source document, so a script gets the new id without scraping the message.
func TestOracleTemplateAdd_JSON(t *testing.T) {
	resetCliForTest()
	skipWhenTemplateHostIsBlocked(t)

	out, err := runCmd(t, "oracle", "template", "add", "--template", "eth-price", "--json")
	require.NoError(t, err)
	var source struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &source), "got %q", out)
	assert.NotEmpty(t, source.ID)
	assert.Equal(t, "Ethereum Price", source.Name)
	assert.Contains(t, source.URL, "ethereum")
}

func TestOracleTemplateAdd_SecondTemplate(t *testing.T) {
	resetCliForTest()
	skipWhenTemplateHostIsBlocked(t)

	// Interval & type defaulting are exercised in the usecase layer; here we
	// just ensure the CLI accepts the other built-in template.
	out, err := runCmd(t, "oracle", "template", "add", "--template", "eth-price")
	require.NoError(t, err)
	assert.Contains(t, out, "Ethereum Price")
}
