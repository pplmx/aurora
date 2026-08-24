package cmd

// ISS-083 regressions (v1.76): the `nft tui` / `oracle tui` commands used
// Run: and swallowed failures with fmt.Println, exiting 0 — so a broken
// operation reported success to $?-checking scripts. They now use RunE, so
// cobra propagates the error to Execute() → stderr + os.Exit(1).

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// findCommand walks the tree from rootCmd to locate the command at path.
func findCommand(t *testing.T, path ...string) *cobra.Command {
	t.Helper()
	cur := rootCmd
	for _, name := range path {
		var next *cobra.Command
		for _, sub := range cur.Commands() {
			if sub.Name() == name {
				next = sub
				break
			}
		}
		require.NotNil(t, next, "command %q must be registered under %q", name, cur.Name())
		cur = next
	}
	return cur
}

// TestTUISubcommandsUseRunE pins the two TUI commands to RunE. A Run-based TUI
// cannot fail: the error is printed to stdout and the process still exits 0.
// With RunE, cobra guarantees Execute() sees the error → os.Exit(1) with the
// message on stderr.
func TestTUISubcommandsUseRunE(t *testing.T) {
	for _, path := range [][]string{{"nft", "tui"}, {"oracle", "tui"}} {
		cmd := findCommand(t, path...)
		require.NotNil(t, cmd.RunE, "%s tui must define RunE (a bare Run would swallow failures and exit 0)", path[0])
		require.Nil(t, cmd.Run, "%s tui must not define a bare Run alongside RunE", path[0])
	}
}
