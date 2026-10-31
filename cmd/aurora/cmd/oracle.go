// The oracle command group: the repository helper each subcommand opens, the
// group itself, and the single init that registers every subcommand and flag.
// The commands live by what they act on — feeds in oracle_source.go, reads in
// oracle_read.go, presets in oracle_template.go.
package cmd

import (
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
	"github.com/spf13/cobra"
)

// newOracleRepo opens a persistent SQLite-backed oracle repository for one
// CLI command and returns a cleanup that closes it. The CLI previously used a
// package-level in-memory repository that was reset on every process exit, so
// `aurora oracle source add` never survived into a later `aurora oracle fetch`
// — the documented multi-command workflow could not work. Every other module
// (lottery/token/nft/voting) persists via store.Path(); oracle now does
// too.
func newOracleRepo() (*sqlite.OracleRepository, func(), error) {
	repo, err := sqlite.NewOracleRepository(store.Path())
	if err != nil {
		return nil, nil, wrapCLIError("oracle.error.open_repo", err)
	}
	return repo, func() { _ = repo.Close() }, nil
}

func init() {
	rootCmd.AddCommand(oracleCmd)

	oracleCmd.AddCommand(sourceCmd)
	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceListCmd)
	addJSONFlag(sourceListCmd)
	sourceCmd.AddCommand(sourceDeleteCmd)
	sourceCmd.AddCommand(sourceEnableCmd)
	sourceCmd.AddCommand(sourceDisableCmd)

	oracleCmd.AddCommand(fetchCmd)

	oracleCmd.AddCommand(dataCmd)
	addJSONFlag(dataCmd)

	oracleCmd.AddCommand(latestCmd)
	addJSONFlag(latestCmd)

	oracleCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	addJSONFlag(templateListCmd)
	templateCmd.AddCommand(templateAddCmd)
	addJSONFlag(templateAddCmd)

	sourceAddCmd.Flags().StringP("name", "n", "", i18n.GetText("oracle.source_name"))
	sourceAddCmd.Flags().StringP("url", "u", "", i18n.GetText("oracle.source_url"))
	sourceAddCmd.Flags().StringP("type", "t", "custom", i18n.GetText("oracle.source_type"))
	sourceAddCmd.Flags().StringP("method", "m", "GET", i18n.GetText("oracle.source_method"))
	sourceAddCmd.Flags().String("headers", "", i18n.GetText("oracle.source_headers"))
	sourceAddCmd.Flags().StringP("path", "p", "", i18n.GetText("oracle.source_path"))
	// On `add`, -i/--interval is the refresh interval — deliberately *not* the
	// family's source-id meaning (-i = --id on delete/enable/disable/fetch/data/
	// latest, TASK-193). It is NOT remapped (DEC-008: no breaking change to
	// documented scripts without operator input); the localized usage text
	// below spells out the divergence instead.
	sourceAddCmd.Flags().IntP("interval", "i", 60, i18n.GetText("oracle.source_interval"))
	_ = sourceAddCmd.MarkFlagRequired("name")
	_ = sourceAddCmd.MarkFlagRequired("url")
	sourceListCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	sourceListCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))

	sourceDeleteCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	addConfirmFlag(sourceDeleteCmd, "cli.confirm.oracle_del")
	_ = sourceDeleteCmd.MarkFlagRequired("id")

	sourceEnableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	_ = sourceEnableCmd.MarkFlagRequired("id")

	sourceDisableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	_ = sourceDisableCmd.MarkFlagRequired("id")

	// fetch/data/latest select a source with --id/-i, the same spelling as the
	// source management commands, so an id printed by `oracle source list` can
	// be pasted into any of them.
	fetchCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))

	dataCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	dataCmd.Flags().IntP("limit", "l", 10, i18n.GetText("oracle.limit"))

	latestCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))

	templateAddCmd.Flags().StringP("template", "t", "", i18n.GetText("oracle.template"))
	_ = templateAddCmd.MarkFlagRequired("template")
}

var oracleCmd = &cobra.Command{
	Use:   "oracle",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.cmd"),
	Long:  i18n.GetText("oracle.cmd"),
}
