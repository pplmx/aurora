package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCmd executes a subcommand through rootCmd (the real CLI path) with
// the given args, capturing stdout. rootCmd.PersistentPreRunE normally
// wires the full app (app.Wire + migrations); those tests exercise the
// composition root, not individual command bodies, so we neutralise it
// here. None of the module subcommands depend on GlobalApp (set by that
// pre-run), so this is safe.
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

	out, err = runCmd(t, "oracle", "source", "delete", "--id", ids[0])
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")

	out, err = runCmd(t, "oracle", "source", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, ids[0])
	assert.Contains(t, out, ids[1])

	// delete a non-existent id: the in-memory repository's DeleteSource is
	// idempotent (no-op for unknown ids), so the CLI reports success. The
	// DB-backed repo (sqlite) behaves the same (DELETE of a missing row is
	// not an error). We assert the no-error contract rather than a leaky
	// "not found" message.
	out, err = runCmd(t, "oracle", "source", "delete", "--id", "no-such-id")
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")
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

	// No data stored yet. GetDataUseCase tolerates an unknown source (returns
	// an empty list), so `data` succeeds and prints "(none)".
	out, err := runCmd(t, "oracle", "data", "--source", "missing", "--limit", "5")
	require.NoError(t, err)
	assert.Contains(t, out, "(none)")

	// GetLatestDataUseCase, in contrast, surfaces the data store's
	// ErrNotFound for an unknown/empty source, so `latest` errors rather
	// than printing "No data found" (that branch is only reachable when the
	// repo returns nil data with no error — not the case for in-memory).
	out, err = runCmd(t, "oracle", "latest", "--source", "missing")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "not found")
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
