package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListCommandsPrintJSON keeps every list command scriptable: --json prints
// one array with no banner, and an empty collection is [] rather than null.
func TestListCommandsPrintJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		for _, prefix := range [][]string{
			{"lottery", "history"},
			{"token", "list"},
			{"nft", "list"},
			{"voting", "candidate", "list"},
			{"voting", "voter", "list"},
			{"voting", "session", "list"},
			{"oracle", "source", "list"},
		} {
			out, err := runCmd(t, append(append([]string{}, prefix...), "--json")...)
			require.NoErrorf(t, err, "%v", prefix)

			var docs []json.RawMessage
			require.NoErrorf(t, json.Unmarshal([]byte(out), &docs),
				"%v must print a JSON array, got %q", prefix, out)
			require.NotNilf(t, docs,
				"%v must print [] for an empty collection, got %q", prefix, out)
		}

		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "json-list", "--count", "1")
		require.NoError(t, err)

		out, err := runCmd(t, "lottery", "history", "--json")
		require.NoError(t, err)
		var records []struct {
			ID   string `json:"id"`
			Seed string `json:"seed"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &records), "got %q", out)
		require.Len(t, records, 1)
		assert.Equal(t, "json-list", records[0].Seed)
	})
}
