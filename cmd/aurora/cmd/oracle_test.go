package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// list shows both
	out, err = runCmd(t, "oracle", "source", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "btc")
	assert.Contains(t, out, "eth")

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
	_, err = runCmd(t, "oracle", "fetch", "--source", id[0])
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

func TestOracleSourceAdd_ValidationErrors(t *testing.T) {
	resetCliForTest()

	// Missing required flag -> cobra validation error before RunE.
	_, err := runCmd(t, "oracle", "source", "add")
	require.Error(t, err)
}

func TestOracleFetch_SourceNotFound(t *testing.T) {
	resetCliForTest()

	// No sources at all: fetch of unknown id returns ErrSourceNotFound
	// before any network call.
	_, err := runCmd(t, "oracle", "fetch", "--source", "missing")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")
}

func TestOracleData_Latest(t *testing.T) {
	resetCliForTest()

	// Unknown sources are rejected consistently across `data` and `latest`
	// (ISS-130). Historically `data` silently returned "(none)" for an unknown
	// source while `latest` surfaced an unclassified error; both now surface
	// the ~source not found~ sentinel (404 on the REST surface).
	_, err := runCmd(t, "oracle", "data", "--source", "missing", "--limit", "5")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")

	_, err = runCmd(t, "oracle", "latest", "--source", "missing")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "source not found")
}

func TestOracleTemplateList_Add(t *testing.T) {
	resetCliForTest()

	out, err := runCmd(t, "oracle", "template", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "btc-price")
	assert.Contains(t, out, "eth-price")

	// template add for a known template -> source created
	out, err = runCmd(t, "oracle", "template", "add", "--template", "btc-price")
	require.NoError(t, err)
	assert.Contains(t, out, "Bitcoin Price")

	// unknown template -> clean error
	_, err = runCmd(t, "oracle", "template", "add", "--template", "no-such-template")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
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

func TestOracleTemplateAdd_SecondTemplate(t *testing.T) {
	resetCliForTest()

	// Interval & type defaulting are exercised in the usecase layer; here we
	// just ensure the CLI accepts the other built-in template.
	out, err := runCmd(t, "oracle", "template", "add", "--template", "eth-price")
	require.NoError(t, err)
	assert.Contains(t, out, "Ethereum Price")
}
