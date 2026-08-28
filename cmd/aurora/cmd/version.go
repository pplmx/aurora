package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version and BuildTime are overridable at link time, e.g.
//
//	go build -ldflags "-X github.com/pplmx/aurora/cmd/aurora/cmd.Version=v1.2.3
//	                    -X github.com/pplmx/aurora/cmd/aurora/cmd.BuildTime=2026-08-28T10:00:00Z"
//
// They live in the cmd package (not main) so the `version` command can read
// the exact values the binary was built with; previously the command printed
// a hardcoded "0.0.1" and a fabricated Go version, so ldflags had no effect
// and users were shown data the binary never carried (TASK-125, ISS-117).
var (
	Version   = "0.0.1"
	BuildTime = "unknown"
)

// versionCmd reports the build's real version, build time and Go toolchain.
// It moved out of lottery.go (where it had historically lived) and now refers
// to the link-time variables above instead of hardcoded placeholders.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Aurora - VRF Lottery System")
		fmt.Fprintf(out, "Version: %s\n", Version)
		fmt.Fprintf(out, "Build Time: %s\n", BuildTime)
		fmt.Fprintf(out, "Go Version: %s\n", runtime.Version())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
