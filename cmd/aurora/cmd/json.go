package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

// addJSONFlag registers the shared --json switch on a list command. The command
// then prints the same fields the REST API returns, so a script parses one
// shape whether it called the CLI or the API.
func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, i18n.GetText("common.json"))
}

// jsonRequested reports the --json switch; a command without the flag prints
// its table.
func jsonRequested(cmd *cobra.Command) bool {
	value, err := cmd.Flags().GetBool("json")
	return err == nil && value
}

// printJSON emits one indented document on stdout and nothing else, so a caller
// can pipe the command into jq without stripping a banner.
func printJSON(value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return wrapCLIError("common.error.json", err)
	}
	fmt.Println(string(out))
	return nil
}
