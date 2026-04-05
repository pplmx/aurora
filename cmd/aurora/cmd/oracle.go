package cmd

import (
	"fmt"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/i18n"
	oracleinfra "github.com/pplmx/aurora/internal/infra/sqlite"
	oracleui "github.com/pplmx/aurora/internal/ui/oracle"
	"github.com/spf13/cobra"
)

var (
	repo oracleinfra.InMemoryOracleRepository
)

func init() {
	repo = *oracleinfra.NewInMemoryOracleRepository()
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
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")
		dataType, _ := cmd.Flags().GetString("type")
		interval, _ := cmd.Flags().GetInt("interval")

		uc := oracleapp.NewAddSourceUseCase(&repo)
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
		uc := oracleapp.NewListSourcesUseCase(&repo)
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
		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDeleteSourceUseCase(&repo)
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
		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewEnableSourceUseCase(&repo)
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
		id, _ := cmd.Flags().GetString("id")

		uc := oracleapp.NewDisableSourceUseCase(&repo)
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
		sourceID, _ := cmd.Flags().GetString("source")

		uc := oracleapp.NewFetchDataUseCase(&repo)
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
		sourceID, _ := cmd.Flags().GetString("source")
		limit, _ := cmd.Flags().GetInt("limit")

		uc := oracleapp.NewGetDataUseCase(&repo)
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
		sourceID, _ := cmd.Flags().GetString("source")

		uc := oracleapp.NewGetLatestDataUseCase(&repo)
		resp, err := uc.Execute(&oracleapp.GetLatestDataRequest{SourceID: sourceID})
		if err != nil {
			return fmt.Errorf("failed to get latest data: %w", err)
		}
		if resp.Data == nil {
			fmt.Println("No data found")
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
		templates := getTemplates()
		fmt.Println("\n📋 Available Templates:")
		for _, t := range templates {
			fmt.Printf("   - %s\n", t)
		}
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add",
	Short: i18n.GetText("oracle.template.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName, _ := cmd.Flags().GetString("template")

		template, ok := getTemplate(templateName)
		if !ok {
			return fmt.Errorf("template not found: %s", templateName)
		}

		uc := oracleapp.NewAddSourceUseCase(&repo)
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
	Run: func(cmd *cobra.Command, args []string) {
		inMemoryRepo := oracleinfra.NewInMemoryOracleRepository()
		if err := oracleui.RunOracleTUI(inMemoryRepo); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

func getTemplates() []string {
	keys := make([]string, 0, len(DataSourceTemplates))
	for k := range DataSourceTemplates {
		keys = append(keys, k)
	}
	return keys
}

func getTemplate(name string) (DataSource, bool) {
	template, ok := DataSourceTemplates[name]
	return template, ok
}

type DataSource struct {
	Name     string
	URL      string
	Type     string
	Method   string
	Path     string
	Interval int
}

var DataSourceTemplates = map[string]DataSource{
	"btc-price": {
		Name:     "Bitcoin Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "bitcoin.usd",
		Interval: 60,
	},
	"eth-price": {
		Name:     "Ethereum Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "ethereum.usd",
		Interval: 60,
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
	sourceAddCmd.MarkFlagRequired("name")
	sourceAddCmd.MarkFlagRequired("url")

	sourceDeleteCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	sourceDeleteCmd.MarkFlagRequired("id")

	sourceEnableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	sourceEnableCmd.MarkFlagRequired("id")

	sourceDisableCmd.Flags().StringP("id", "i", "", i18n.GetText("oracle.source_id"))
	sourceDisableCmd.MarkFlagRequired("id")

	fetchCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	fetchCmd.MarkFlagRequired("source")

	dataCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	dataCmd.Flags().IntP("limit", "l", 10, i18n.GetText("oracle.limit"))
	dataCmd.MarkFlagRequired("source")

	latestCmd.Flags().StringP("source", "s", "", i18n.GetText("oracle.source_id"))
	latestCmd.MarkFlagRequired("source")

	templateAddCmd.Flags().StringP("template", "t", "", i18n.GetText("oracle.template"))
	templateAddCmd.MarkFlagRequired("template")
}
