package cmd

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"
	"time"

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

// TestReadCommandsPrintJSON covers the single-record reads: each prints one
// document with the REST field names, and the verify report carries the verdict
// as data rather than prose.
func TestReadCommandsPrintJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		// A created draw is verified by its own id, discovered through --json.
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "json-read", "--count", "1")
		require.NoError(t, err)
		var draws []struct {
			ID string `json:"id"`
		}
		out, err := runCmd(t, "lottery", "history", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &draws))
		require.Len(t, draws, 1)

		out, err = runCmd(t, "lottery", "verify", draws[0].ID, "--json")
		require.NoError(t, err)
		var report struct {
			IntegrityOK bool `json:"integrity_ok"`
			Checks      []struct {
				Status string `json:"status"`
			} `json:"checks"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "got %q", out)
		assert.True(t, report.IntegrityOK)
		assert.NotEmpty(t, report.Checks)

		// Balance and allowance answer the API's fields for a key holding none.
		// The key is non-zero because the NFT mint refuses a zero address.
		key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
		_, err = runCmd(t, "token", "create", "-n", "Gold", "-s", "GLD", "-t", "1000")
		require.NoError(t, err)

		out, err = runCmd(t, "token", "balance", "-t", "GLD", "-o", key, "--json")
		require.NoError(t, err)
		var balance struct {
			TokenID string `json:"token_id"`
			Owner   string `json:"owner"`
			Amount  string `json:"amount"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &balance), "got %q", out)
		assert.Equal(t, "GLD", balance.TokenID)
		assert.Equal(t, key, balance.Owner)
		assert.Equal(t, "0", balance.Amount)

		out, err = runCmd(t, "token", "info", "-t", "GLD", "--json")
		require.NoError(t, err)
		var info struct {
			ID       string `json:"id"`
			Mintable bool   `json:"mintable"`
			Burnable bool   `json:"burnable"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &info), "got %q", out)
		assert.Equal(t, "GLD", info.ID)
		assert.True(t, info.Mintable)
		assert.True(t, info.Burnable)

		out, err = runCmd(t, "token", "allowance", "-t", "GLD", "-o", key, "-s", key, "--json")
		require.NoError(t, err)
		var allowance struct {
			TokenID string `json:"token_id"`
			Spender string `json:"spender"`
			Amount  string `json:"amount"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &allowance), "got %q", out)
		assert.Equal(t, key, allowance.Spender)
		assert.Equal(t, "0", allowance.Amount)

		// A minted asset reads back with its metadata.
		_, err = runCmd(t, "nft", "mint", "-n", "Star", "-c", key)
		require.NoError(t, err)
		var assets []struct {
			ID string `json:"id"`
		}
		out, err = runCmd(t, "nft", "list", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &assets))
		require.Len(t, assets, 1)

		out, err = runCmd(t, "nft", "get", "-i", assets[0].ID, "--json")
		require.NoError(t, err)
		var asset struct {
			Name  string `json:"name"`
			Owner string `json:"owner"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &asset), "got %q", out)
		assert.Equal(t, "Star", asset.Name)
		assert.Equal(t, key, asset.Owner)

		// Oracle readings use the REST wrapper, empty until a fetch stores one.
		_, err = runCmd(t, "oracle", "source", "add",
			"-n", "btc", "-u", "https://api.example.com/price")
		require.NoError(t, err)
		var sources []struct {
			ID string `json:"id"`
		}
		out, err = runCmd(t, "oracle", "source", "list", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &sources))
		require.Len(t, sources, 1)

		out, err = runCmd(t, "oracle", "data", "-i", sources[0].ID, "--json")
		require.NoError(t, err)
		var readings struct {
			Data []any `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &readings), "got %q", out)
		assert.Empty(t, readings.Data)

		// The tally names the session and reports each candidate in roster order.
		_, err = runCmd(t, "voting", "candidate", "add", "-n", "Alice", "-p", "Greens")
		require.NoError(t, err)
		var candidates []struct {
			ID string `json:"id"`
		}
		out, err = runCmd(t, "voting", "candidate", "list", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &candidates))
		require.Len(t, candidates, 1)

		now := time.Now().Unix()
		_, err = runCmd(t, "voting", "session", "create", "-t", "Budget",
			"-c", candidates[0].ID,
			"--start-time", strconv.FormatInt(now, 10),
			"--end-time", strconv.FormatInt(now+86400, 10))
		require.NoError(t, err)
		var sessions []struct {
			ID string `json:"id"`
		}
		out, err = runCmd(t, "voting", "session", "list", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &sessions))
		require.Len(t, sessions, 1)

		out, err = runCmd(t, "voting", "results", "--session", sessions[0].ID, "--json")
		require.NoError(t, err)
		var results struct {
			SessionID  string `json:"session_id"`
			TotalVotes int    `json:"total_votes"`
			Candidates []struct {
				Name      string `json:"name"`
				VoteCount int    `json:"vote_count"`
			} `json:"candidates"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &results), "got %q", out)
		assert.Equal(t, sessions[0].ID, results.SessionID)
		assert.Equal(t, 0, results.TotalVotes)
		require.Len(t, results.Candidates, 1)
		assert.Equal(t, "Alice", results.Candidates[0].Name)
	})
}

// TestHistoryCommandsPrintJSON covers the two operation histories: a token with
// no transfers answers transfers: [], and an NFT's mint appears as one
// operation with the REST field names.
func TestHistoryCommandsPrintJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")
		key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

		_, err := runCmd(t, "token", "create", "-n", "Gold", "-s", "GLD", "-t", "1000")
		require.NoError(t, err)
		out, err := runCmd(t, "token", "history", "-t", "GLD", "--json")
		require.NoError(t, err)
		var history struct {
			Transfers []json.RawMessage `json:"transfers"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &history), "got %q", out)
		require.NotNil(t, history.Transfers, "an empty history is [], not null")
		assert.Empty(t, history.Transfers)

		_, err = runCmd(t, "nft", "mint", "-n", "Star", "-c", key)
		require.NoError(t, err)
		var assets []struct {
			ID string `json:"id"`
		}
		out, err = runCmd(t, "nft", "list", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &assets))
		require.Len(t, assets, 1)

		out, err = runCmd(t, "nft", "history", "-i", assets[0].ID, "--json")
		require.NoError(t, err)
		var ops []struct {
			Type  string `json:"type"`
			NFTID string `json:"nft_id"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &ops), "got %q", out)
		require.Len(t, ops, 1)
		assert.Equal(t, "mint", ops[0].Type)
		assert.Equal(t, assets[0].ID, ops[0].NFTID)
	})
}

// TestCreationCommandsPrintJSON: the creation commands return their REST DTOs,
// so a script captures the new id from the document instead of parsing a
// labeled line. The commands that print a private key (token create, voter
// register) intentionally have no JSON mode.
func TestCreationCommandsPrintJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")
		key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

		out, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "json-write", "--count", "1", "--json")
		require.NoError(t, err)
		var draw struct {
			ID          string   `json:"id"`
			BlockHeight int64    `json:"block_height"`
			Winners     []string `json:"winners"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &draw), "got %q", out)
		assert.NotEmpty(t, draw.ID)
		assert.Greater(t, draw.BlockHeight, int64(0))
		assert.Len(t, draw.Winners, 1)

		out, err = runCmd(t, "nft", "mint", "-n", "Comet", "-c", key, "--json")
		require.NoError(t, err)
		var asset struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Owner string `json:"owner"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &asset), "got %q", out)
		assert.NotEmpty(t, asset.ID)
		assert.Equal(t, "Comet", asset.Name)
		assert.Equal(t, key, asset.Owner)

		out, err = runCmd(t, "voting", "candidate", "add", "-n", "Bob", "-p", "Reds", "--json")
		require.NoError(t, err)
		var candidate struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &candidate), "got %q", out)
		assert.NotEmpty(t, candidate.ID)
		assert.Equal(t, "Bob", candidate.Name)

		now := time.Now().Unix()
		out, err = runCmd(t, "voting", "session", "create", "-t", "Budget",
			"-c", candidate.ID,
			"--start-time", strconv.FormatInt(now, 10),
			"--end-time", strconv.FormatInt(now+3600, 10), "--json")
		require.NoError(t, err)
		var session struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &session), "got %q", out)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "Budget", session.Title)
		assert.Equal(t, "draft", session.Status)
	})
}

// TestVersionCommandPrintsJSON: an ops script reads the build's version, build
// time and toolchain from one document instead of parsing three labeled lines.
func TestVersionCommandPrintsJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		out, err := runCmd(t, "version", "--json")
		require.NoError(t, err)
		var info struct {
			Name      string `json:"name"`
			Version   string `json:"version"`
			BuildTime string `json:"build_time"`
			GoVersion string `json:"go_version"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &info), "got %q", out)
		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Version)
		assert.NotEmpty(t, info.BuildTime)
		assert.Regexp(t, `^go`, info.GoVersion)
	})
}
