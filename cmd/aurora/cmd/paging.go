package cmd

import "github.com/spf13/cobra"

// pagingFlags reads the -l/--limit and -p/--offset flags every list command
// registers. A negative value is a usage error rather than a silent default,
// so a script can tell a typo from an empty page. Missing flags are an error
// too: the command declared them, and a nil result would hide that mistake.
func pagingFlags(cmd *cobra.Command) (limit, offset int, err error) {
	if limit, err = cmd.Flags().GetInt("limit"); err != nil {
		return 0, 0, err
	}
	if offset, err = cmd.Flags().GetInt("offset"); err != nil {
		return 0, 0, err
	}
	if limit < 0 {
		return 0, 0, newCLIError("cli.error.invalid_limit", limit)
	}
	if offset < 0 {
		return 0, 0, newCLIError("cli.error.invalid_offset", offset)
	}
	return limit, offset, nil
}
