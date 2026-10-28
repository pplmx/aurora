package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infrahttp "github.com/pplmx/aurora/internal/infra/http"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
)

// newOracleRepo opens a persistent SQLite-backed oracle repository for one
// CLI command and returns a cleanup that closes it. The CLI previously used a
// package-level in-memory repository that was reset on every process exit, so
// `aurora oracle source add` never survived into a later `aurora oracle fetch`
// — the documented multi-command workflow could not work. Every other module
// (lottery/token/nft/voting) persists via blockchain.DBPath(); oracle now does
// too.
func newOracleRepo() (*sqlite.OracleRepository, func(), error) {
	repo, err := sqlite.NewOracleRepository(blockchain.DBPath())
	if err != nil {
		return nil, nil, wrapCLIError("oracle.error.open_repo", err)
	}
	return repo, func() { _ = repo.Close() }, nil
}

var oracleCmd = &cobra.Command{
	Use:   "oracle",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.cmd"),
	Long:  i18n.GetText("oracle.cmd"),
}

var sourceCmd = &cobra.Command{
	Use:   "source",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.cmd"),
}

var sourceAddCmd = &cobra.Command{
	Use:   "add",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		dataType, _ := cmd.Flags().GetString("type")
		interval, _ := cmd.Flags().GetInt("interval")
		method, _ := cmd.Flags().GetString("method")
		headers, _ := cmd.Flags().GetString("headers")
		path, _ := cmd.Flags().GetString("path")

		uc := oracleapp.NewAddSourceUseCase(repo)
		resp, err := uc.Execute(&oracleapp.AddSourceRequest{
			Name:     name,
			URL:      url,
			Type:     dataType,
			Method:   method,
			Headers:  headers,
			Path:     path,
			Interval: interval,
		})
		if err != nil {
			return wrapCLIError("oracle.error.register_source", err)
		}

		fmt.Printf("✅ "+i18n.GetText("oracle.source_added")+"\n", resp.Name)
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.url", resp.URL)
		return nil
	},
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		if limit < 0 {
			return newCLIError("cli.error.invalid_limit", limit)
		}
		if offset < 0 {
			return newCLIError("cli.error.invalid_offset", offset)
		}

		uc := oracleapp.NewListSourcesUseCase(repo)
		resp, err := uc.Execute(&oracleapp.ListSourcesRequest{Limit: limit, Offset: offset})
		if err != nil {
			return wrapCLIError("oracle.error.list_sources", err)
		}

		if limit > 0 {
			fmt.Printf("\n📡 %s: %s\n",
				i18n.GetText("oracle.sources_heading"),
				i18n.GetTextF("common.page_with_offset", len(resp.Sources), offset))
		} else {
			fmt.Printf("\n📡 %s:\n", i18n.GetText("oracle.sources_heading"))
		}
		if len(resp.Sources) == 0 {
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, ds := range resp.Sources {
			status := "✅ " + i18n.GetText("common.enabled")
			if !ds.Enabled {
				status = "⏳ " + i18n.GetText("common.disabled")
			}
			fmt.Printf("   - %s [%s] %s\n", ds.Name, ds.Type, status)
			printField("  "+i18n.GetText("common.id"), ds.ID)
			printField("  "+i18n.GetText("common.url"), ds.URL)
			fmt.Printf("     %s: %s  %s: %s  %s: %ds\n",
				i18n.GetText("common.method"), ds.Method,
				i18n.GetText("common.path"), ds.Path,
				i18n.GetText("common.interval"), ds.Interval)
			if ds.Headers != "" {
				fmt.Printf("     %s: %s\n",
					i18n.GetText("common.headers"),
					i18n.GetText("common.configured"))
			}
		}
		return nil
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.delete"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Deleting a source is permanent; require -y/--confirm first.
		if err := requireConfirm(cmd, "cli.confirm.oracle_del"); err != nil {
			return err
		}
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDeleteSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.delete_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_deleted"))
		return nil
	},
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.enable"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewEnableSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.enable_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_enabled"))
		return nil
	},
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.source.disable"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDisableSourceUseCase(repo)
		if err := uc.Execute(id); err != nil {
			return wrapCLIError("oracle.error.disable_source", err)
		}
		fmt.Println("✅ " + i18n.GetText("oracle.source_disabled"))
		return nil
	},
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.fetch"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}

		fetcher := infrahttp.NewFetcher()
		defer func() { _ = fetcher.Close() }()
		uc := oracleapp.NewFetchDataUseCaseWithDeps(repo, fetcher)
		// Record CLI-fetched data on-chain, matching the API/scheduler paths
		// and the package intent; otherwise BlockHeight always printed 0.
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("oracle.error.init_blockchain", err)
		}
		uc.SetChain(chain)
		resp, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})
		if err != nil {
			return wrapCLIError("oracle.error.fetch", err)
		}

		fmt.Println("✅ " + i18n.GetText("oracle.fetched"))
		printLocalizedField("common.value", resp.Value)
		fmt.Printf("   %s: %d\n", i18n.GetText("common.timestamp"), resp.Timestamp)
		fmt.Printf("   %s: %d\n", i18n.GetText("common.block_height"), resp.BlockHeight)
		return nil
	},
}

var dataCmd = &cobra.Command{
	Use:   "data",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.data.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}
		limit := clampQueryLimit(cmd)

		uc := oracleapp.NewGetDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetDataRequest{SourceID: sourceID, Limit: limit})
		if err != nil {
			return wrapCLIError("oracle.error.get_data", err)
		}

		fmt.Printf("\n📊 %s:\n", i18n.GetText("oracle.data_heading"))
		if len(resp.Data) == 0 {
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, d := range resp.Data {
			fmt.Printf("   %s\n", i18n.GetTextF("oracle.data_entry", d.Timestamp, d.Value, d.BlockHeight))
		}
		return nil
	},
}

var latestCmd = &cobra.Command{
	Use:   "latest",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.latest"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, err := resolveOracleSourceID(cmd)
		if err != nil {
			return err
		}

		uc := oracleapp.NewGetLatestDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetLatestDataRequest{SourceID: sourceID})
		if err != nil {
			return wrapCLIError("oracle.error.get_latest", err)
		}
		if resp.Data == nil {
			fmt.Println(i18n.GetText("oracle.no_data"))
			return nil
		}

		fmt.Printf("\n📈 %s:\n", i18n.GetText("oracle.latest_heading"))
		printLocalizedField("common.value", resp.Data.Value)
		fmt.Printf("   %s: %d\n", i18n.GetText("common.timestamp"), resp.Data.Timestamp)
		fmt.Printf("   %s: %d\n", i18n.GetText("common.block_height"), resp.Data.BlockHeight)
		return nil
	},
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.cmd"),
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		templates := oracleapp.ListTemplates()
		fmt.Printf("\n📋 %s:\n", i18n.GetText("oracle.templates_heading"))
		for _, t := range templates {
			fmt.Printf("   - %s\n", t.ID)
		}
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("oracle.template.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		templateName, _ := cmd.Flags().GetString("template")

		template, ok := oracleapp.GetTemplate(templateName)
		if !ok {
			return newCLIError("oracle.error.template_missing", templateName)
		}

		uc := oracleapp.NewAddSourceUseCase(repo)
		resp, err := uc.Execute(&oracleapp.AddSourceRequest{
			Name:     template.Name,
			URL:      template.URL,
			Type:     template.Type,
			Method:   template.Method,
			Path:     template.Path,
			Interval: template.Interval,
		})
		if err != nil {
			return wrapCLIError("oracle.error.add_template", err)
		}

		fmt.Printf("✅ "+i18n.GetText("oracle.template_added")+"\n", resp.Name)
		printLocalizedField("common.id", resp.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(oracleCmd)

	oracleCmd.AddCommand(sourceCmd)
	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceDeleteCmd)
	sourceCmd.AddCommand(sourceEnableCmd)
	sourceCmd.AddCommand(sourceDisableCmd)

	oracleCmd.AddCommand(fetchCmd)

	oracleCmd.AddCommand(dataCmd)

	oracleCmd.AddCommand(latestCmd)

	oracleCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateAddCmd)

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

	// fetch/data/latest select a source with --id/-i (the canonical spelling,
	// matching the source management commands) OR the legacy --source/-s
	// alias — a user pasting an ID printed by `oracle source list` used to
	// hit "unknown flag: --id" on fetch/data/latest (ISS-189, TASK-193).
	// MarkFlagRequired("source") is dropped because either spelling counts;
	// resolveOracleSourceID errors when neither is given.
	fetchCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	fetchCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id_legacy"))

	dataCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	dataCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id_legacy"))
	dataCmd.Flags().IntP("limit", "l", 10, i18n.GetText("oracle.limit"))

	latestCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	latestCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id_legacy"))

	templateAddCmd.Flags().StringP("template", "t", "", i18n.GetText("oracle.template"))
	_ = templateAddCmd.MarkFlagRequired("template")
}

// resolveOracleSourceID returns the source selector for fetch/data/latest,
// accepting the canonical --id/-i (matching the oracle source management
// commands) and the legacy --source/-s spelling. --id wins when both are
// given, and neither counts as a clean usage error instead of an empty
// lookup (ISS-189, TASK-193).
func resolveOracleSourceID(cmd *cobra.Command) (string, error) {
	if id, _ := cmd.Flags().GetString("id"); id != "" {
		return id, nil
	}
	if src, _ := cmd.Flags().GetString("source"); src != "" {
		return src, nil
	}
	return "", newCLIError("oracle.error.source_required")
}

// maxCLIQueryLimit mirrors the REST API handler cap (maxQueryLimit=100) so the
// CLI cannot drive an unbounded DB scan with `-l 999999999`.
const maxCLIQueryLimit = 100

// clampQueryLimit reads the data command's limit flag and clamps it into
// [1, maxCLIQueryLimit], defaulting to 10 when unset/<=0. This matches the API
// hardening added in v1.9.
func clampQueryLimit(cmd *cobra.Command) int {
	v, err := cmd.Flags().GetInt("limit")
	if err != nil || v <= 0 {
		return 10
	}
	if v > maxCLIQueryLimit {
		return maxCLIQueryLimit
	}
	return v
}
