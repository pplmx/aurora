package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/oracle"
	"github.com/spf13/cobra"
)

var oracleCmd = &cobra.Command{
	Use:   "oracle",
	Short: "Oracle data service",
	Long:  "Fetch and store external data on blockchain",
}

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Data source management",
}

var sourceAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		dataType, _ := cmd.Flags().GetString("type")
		interval, _ := cmd.Flags().GetInt("interval")

		ds, err := oracle.RegisterDataSource(name, url, dataType, interval)
		if err != nil {
			return fmt.Errorf("failed to register data source: %w", err)
		}

		fmt.Printf("✅ Data source created: %s\n", ds.Name)
		fmt.Printf("   ID: %s\n", ds.ID)
		fmt.Printf("   URL: %s\n", ds.URL)
		return nil
	},
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List data sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := oracle.ListDataSources()
		if err != nil {
			return fmt.Errorf("failed to list data sources: %w", err)
		}

		fmt.Println("\n📡 Data Sources:")
		if len(list) == 0 {
			fmt.Println("   (none)")
		}
		for _, ds := range list {
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
	Short: "Delete a data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := oracle.DeleteDataSource(id); err != nil {
			return fmt.Errorf("failed to delete data source: %w", err)
		}
		fmt.Println("✅ Data source deleted!")
		return nil
	},
}

var sourceEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable a data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := oracle.EnableDataSource(id); err != nil {
			return fmt.Errorf("failed to enable data source: %w", err)
		}
		fmt.Println("✅ Data source enabled!")
		return nil
	},
}

var sourceDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable a data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if err := oracle.DisableDataSource(id); err != nil {
			return fmt.Errorf("failed to disable data source: %w", err)
		}
		fmt.Println("✅ Data source disabled!")
		return nil
	},
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch data from source",
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID, _ := cmd.Flags().GetString("source")

		chain := blockchain.InitBlockChain()
		data, err := oracle.FetchAndSave(sourceID, chain)
		if err != nil {
			return fmt.Errorf("failed to fetch data: %w", err)
		}

		fmt.Println("✅ Data fetched successfully!")
		fmt.Printf("   Value: %s\n", data.Value)
		fmt.Printf("   Timestamp: %d\n", data.Timestamp)
		fmt.Printf("   Block Height: %d\n", data.BlockHeight)
		return nil
	},
}

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Query oracle data",
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID, _ := cmd.Flags().GetString("source")
		limit, _ := cmd.Flags().GetInt("limit")

		list, err := oracle.GetOracleData(sourceID, limit)
		if err != nil {
			return fmt.Errorf("failed to get oracle data: %w", err)
		}

		fmt.Println("\n📊 Oracle Data:")
		if len(list) == 0 {
			fmt.Println("   (none)")
		}
		for _, d := range list {
			fmt.Printf("   [%d] %s - Block #%d\n", d.Timestamp, d.Value, d.BlockHeight)
		}
		return nil
	},
}

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Get latest data from source",
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID, _ := cmd.Flags().GetString("source")

		data, err := oracle.GetLatestOracleData(sourceID)
		if err != nil {
			return fmt.Errorf("failed to get latest data: %w", err)
		}
		if data == nil {
			fmt.Println("No data found")
			return nil
		}

		fmt.Println("\n📈 Latest Data:")
		fmt.Printf("   Value: %s\n", data.Value)
		fmt.Printf("   Timestamp: %d\n", data.Timestamp)
		fmt.Printf("   Block Height: %d\n", data.BlockHeight)
		return nil
	},
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Data source templates",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		templates := oracle.ListTemplates()
		fmt.Println("\n📋 Available Templates:")
		for _, t := range templates {
			fmt.Printf("   - %s\n", t)
		}
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add template as data source",
	RunE: func(cmd *cobra.Command, args []string) error {
		template, _ := cmd.Flags().GetString("template")

		ds, err := oracle.AddTemplate(template)
		if err != nil {
			return fmt.Errorf("failed to add template: %w", err)
		}

		fmt.Printf("✅ Template added: %s\n", ds.Name)
		fmt.Printf("   ID: %s\n", ds.ID)
		return nil
	},
}

var oracleTuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch TUI interface",
	Run: func(cmd *cobra.Command, args []string) {
		storage := oracle.NewInMemoryStorage()
		oracle.InitOracle(storage)
		if err := oracle.RunOracleTUI(storage); err != nil {
			fmt.Println("Error:", err)
		}
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

	sourceAddCmd.Flags().StringP("name", "n", "", "Data source name")
	sourceAddCmd.Flags().StringP("url", "u", "", "API URL")
	sourceAddCmd.Flags().StringP("type", "t", "custom", "Data type")
	sourceAddCmd.Flags().IntP("interval", "i", 60, "Refresh interval (seconds)")
	sourceAddCmd.MarkFlagRequired("name")
	sourceAddCmd.MarkFlagRequired("url")

	sourceDeleteCmd.Flags().StringP("id", "i", "", "Source ID")
	sourceDeleteCmd.MarkFlagRequired("id")

	sourceEnableCmd.Flags().StringP("id", "i", "", "Source ID")
	sourceEnableCmd.MarkFlagRequired("id")

	sourceDisableCmd.Flags().StringP("id", "i", "", "Source ID")
	sourceDisableCmd.MarkFlagRequired("id")

	fetchCmd.Flags().StringP("source", "s", "", "Source ID")
	fetchCmd.MarkFlagRequired("source")

	dataCmd.Flags().StringP("source", "s", "", "Source ID")
	dataCmd.Flags().IntP("limit", "l", 10, "Limit results")
	dataCmd.MarkFlagRequired("source")

	latestCmd.Flags().StringP("source", "s", "", "Source ID")
	latestCmd.MarkFlagRequired("source")

	templateAddCmd.Flags().StringP("template", "t", "", "Template name")
	templateAddCmd.MarkFlagRequired("template")
}
