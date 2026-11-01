package repocheck

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/app/oracleapp"
)

// TestTheHealthReferenceNamesItsOptionalFields covers the one endpoint whose
// JSON keys come and go: /oracle/health omits last_success_at until a fetch has
// succeeded and last_error once a fetch recovers. A client written from a
// description that listed the fields as if they were always there reads a
// missing key as zero — "succeeded in 1970" or "no error" for a source that has
// never been reachable.
//
// The check marshals the real type, so the document and the tags cannot drift.
func TestTheHealthReferenceNamesItsOptionalFields(t *testing.T) {
	// A source that has never succeeded and currently has an error.
	failing, err := json.Marshal(oracleapp.SourceStat{
		SourceID: "s1", Name: "btc", Attempts: 2, Failures: 2, LastError: "refused",
	})
	require.NoError(t, err)
	var failingKeys map[string]any
	require.NoError(t, json.Unmarshal(failing, &failingKeys))
	assert.Contains(t, failingKeys, "last_error")
	assert.NotContains(t, failingKeys, "last_success_at",
		"a source that never succeeded must not carry last_success_at: %s", failing)

	// A source that has succeeded and has no outstanding error.
	healthy, err := json.Marshal(oracleapp.SourceStat{
		SourceID: "s2", Name: "eth", Attempts: 3, Successes: 3, LastSuccessAt: 1700000000,
	})
	require.NoError(t, err)
	var healthyKeys map[string]any
	require.NoError(t, json.Unmarshal(healthy, &healthyKeys))
	assert.Contains(t, healthyKeys, "last_success_at")
	assert.NotContains(t, healthyKeys, "last_error",
		"a recovered source must not carry last_error: %s", healthy)

	// The required keys are in both.
	for _, keys := range []map[string]any{failingKeys, healthyKeys} {
		for _, key := range []string{"source_id", "attempts", "successes", "failures"} {
			assert.Contains(t, keys, key, "health entry must always carry %q", key)
		}
	}

	// And the reference has to say so.
	doc := readRepoFile(t, "docs", "api.md")
	for _, phrase := range []string{"last_success_at", "last_error", "省略"} {
		assert.Containsf(t, doc, phrase,
			"docs/api.md must explain that the health fields are optional")
	}
	assert.True(t, strings.Contains(doc, "不要指望它们恒为"),
		"docs/api.md must warn that the zero value is not what a missing key means")
}
