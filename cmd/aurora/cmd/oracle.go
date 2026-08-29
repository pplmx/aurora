package cmd

import (
	"fmt"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/i18n"
	oracleinfra "github.com/pplmx/aurora/internal/infra/sqlite"
	oracleui "github.com/pplmx/aurora/internal/ui/oracle"
	"github.com/spf13/cobra"
)

// newOracleRepo opens a persistent SQLite-backed oracle repository for one
// CLI command and returns a cleanup that closes it. The CLI previously used a
// package-level in-memory repository that was reset on every process exit, so
// `aurora oracle source add` never survived into a later `aurora oracle fetch`
// — the documented multi-command workflow could not work. Every other module
// (lottery/token/nft/voting) persists via blockchain.DBPath(); oracle now does
// too.
func newOracleRepo() (*oracleinfra.OracleRepository, func(), error) {
	repo, err := oracleinfra.NewOracleRepository(blockchain.DBPath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open oracle repository: %w", err)
	}
	return repo, func() { _ = repo.Close() }, nil
}

var oracleCmd = &cobra.Command{
	Use:   "oracle",
	Short: i18n.GetText("oracle.cmd"),
	Long:  i18n.GetText("oracle.cmd"),
}

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: i18n.GetText("oracle.source.cmd"),
}

var sourceAddCmd = &cobra.Command{
	Use:   "add",
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

		uc := oracleapp.NewAddSourceUseCase(repo)
		resp, err := uc.Execute(&oracleapp.AddSourceRequest{
			Name:     name,
			URL:      url,
			Type:     dataType,
			Interval: interval,
		})
		if err != nil {
			return fmt.Errorf("failed to register data source: %w", err)
		}

		fmt.Printf("✅ Data source created: %s\n", resp.Name)
		fmt.Printf("   ID: %s\n", resp.ID)
		fmt.Printf("   URL: %s\n", resp.URL)
		return nil
	},
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.GetText("oracle.source.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := oracleapp.NewListSourcesUseCase(repo)
		resp, err := uc.Execute(&oracleapp.ListSourcesRequest{})
		if err != nil {
			return fmt.Errorf("failed to list data sources: %w", err)
		}

		fmt.Println("\n📡 Data Sources:")
		if len(resp.Sources) == 0 {
			fmt.Println("   (none)")
		}
		for _, ds := range resp.Sources {
			status := "✅ enabled"
			if !ds.Enabled {
				status = "⏳ disabled"
			}
			fmt.Printf("   - %s [%s] %s\n", ds.Name, ds.Type, status)
			fmt.Printf("     ID: %s\n", ds.ID)
			fmt.Printf("     URL: %s\n", ds.URL)
		}
		return nil
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: i18n.GetText("oracle.source.delete"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Deleting a source is permanent; require -y/--confirm first.
		if err := requireConfirm(cmd, "permanently deletes the data source"); err != nil {
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
			return fmt.Errorf("failed to delete data source: %w", err)
		}
		fmt.Println("✅ Data source deleted!")
		return nil
	},
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable",
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
			return fmt.Errorf("failed to enable data source: %w", err)
		}
		fmt.Println("✅ Data source enabled!")
		return nil
	},
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable",
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
			return fmt.Errorf("failed to disable data source: %w", err)
		}
		fmt.Println("✅ Data source disabled!")
		return nil
	},
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: i18n.GetText("oracle.fetch"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, _ := cmd.Flags().GetString("source")

		uc := oracleapp.NewFetchDataUseCase(repo)
		// Record CLI-fetched data on-chain, matching the API/scheduler paths
		// and the package intent; otherwise BlockHeight always printed 0.
		uc.SetChain(blockchain.GetBlockChain())
		resp, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})
		if err != nil {
			return fmt.Errorf("failed to fetch data: %w", err)
		}

		fmt.Println("✅ Data fetched successfully!")
		fmt.Printf("   Value: %s\n", resp.Value)
		fmt.Printf("   Timestamp: %d\n", resp.Timestamp)
		fmt.Printf("   Block Height: %d\n", resp.BlockHeight)
		return nil
	},
}

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: i18n.GetText("oracle.data.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, _ := cmd.Flags().GetString("source")
		limit := clampQueryLimit(cmd)

		uc := oracleapp.NewGetDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetDataRequest{SourceID: sourceID, Limit: limit})
		if err != nil {
			return fmt.Errorf("failed to get oracle data: %w", err)
		}

		fmt.Println("\n📊 Oracle Data:")
		if len(resp.Data) == 0 {
			fmt.Println("   (none)")
		}
		for _, d := range resp.Data {
			fmt.Printf("   [%d] %s - Block #%d\n", d.Timestamp, d.Value, d.BlockHeight)
		}
		return nil
	},
}

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: i18n.GetText("oracle.latest"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()

		sourceID, _ := cmd.Flags().GetString("source")

		uc := oracleapp.NewGetLatestDataUseCase(repo)
		resp, err := uc.Execute(&oracleapp.GetLatestDataRequest{SourceID: sourceID})
		if err != nil {
			return fmt.Errorf("failed to get latest data: %w", err)
		}
		if resp.Data == nil {
			fmt.Println(i18n.GetText("oracle.no_data"))
			return nil
		}

		fmt.Println("\n📈 Latest Data:")
		fmt.Printf("   Value: %s\n", resp.Data.Value)
		fmt.Printf("   Timestamp: %d\n", resp.Data.Timestamp)
		fmt.Printf("   Block Height: %d\n", resp.Data.BlockHeight)
		return nil
	},
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: i18n.GetText("oracle.template.cmd"),
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.GetText("oracle.template.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		templates := oracleapp.ListTemplates()
		fmt.Println("\n📋 Available Templates:")
		for _, t := range templates {
			fmt.Printf("   - %s\n", t.ID)
		}
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add",
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
			return fmt.Errorf("template not found: %s", templateName)
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
			return fmt.Errorf("failed to add template: %w", err)
		}

		fmt.Printf("✅ Template added: %s\n", resp.Name)
		fmt.Printf("   ID: %s\n", resp.ID)
		return nil
	},
}

var oracleTuiCmd = &cobra.Command{
	Use:   "tui",
	Short: i18n.GetText("oracle.tui"),
	// RunE (not Run): repo-open and TUI failures must propagate to Execute()
	// so the process exits 1 and the error goes to stderr — the old Run:
	// printed to stdout and exited 0, so $?-checking scripts/CI reported
	// success on a failed run (v1.76, ISS-083).
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, cleanup, err := newOracleRepo()
		if err != nil {
			return err
		}
		defer cleanup()
		return oracleui.RunOracleTUI(repo)
	},
}

func init() {
	rootCmd.AddCommand(oracleCmd)

	oracleCmd.AddCommand(oracleTuiCmd)

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
	sourceAddCmd.Flags().IntP("interval", "i", 60, i18n.GetText("oracle.interval"))
	_ = sourceAddCmd.MarkFlagRequired("name")
	_ = sourceAddCmd.MarkFlagRequired("url")

	sourceDeleteCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	addConfirmFlag(sourceDeleteCmd, "Permanently delete the data source (requires --confirm)")
	_ = sourceDeleteCmd.MarkFlagRequired("id")

	sourceEnableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	_ = sourceEnableCmd.MarkFlagRequired("id")

	sourceDisableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	_ = sourceDisableCmd.MarkFlagRequired("id")

	fetchCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	_ = fetchCmd.MarkFlagRequired("source")

	dataCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	dataCmd.Flags().IntP("limit", "l", 10, i18n.GetText("oracle.limit"))
	_ = dataCmd.MarkFlagRequired("source")

	latestCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	_ = latestCmd.MarkFlagRequired("source")

	templateAddCmd.Flags().StringP("template", "t", "", i18n.GetText("oracle.template"))
	_ = templateAddCmd.MarkFlagRequired("template")
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
