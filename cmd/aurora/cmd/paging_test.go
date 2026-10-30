package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pagingCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().IntP("limit", "l", 0, "")
	cmd.Flags().IntP("offset", "p", 0, "")
	require.NoError(t, cmd.Flags().Parse(args))
	return cmd
}

// TestPagingFlags pins the shared contract: the values pass through, and a
// negative number is a usage error naming the flag rather than a silent default.
func TestPagingFlags(t *testing.T) {
	setTestLocale(t, "en")

	limit, offset, err := pagingFlags(pagingCommand(t))
	require.NoError(t, err)
	assert.Equal(t, 0, limit)
	assert.Equal(t, 0, offset)

	limit, offset, err = pagingFlags(pagingCommand(t, "--limit", "5", "--offset", "10"))
	require.NoError(t, err)
	assert.Equal(t, 5, limit)
	assert.Equal(t, 10, offset)

	_, _, err = pagingFlags(pagingCommand(t, "--limit", "-1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --limit -1")

	_, _, err = pagingFlags(pagingCommand(t, "--offset", "-2"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --offset -2")
}
