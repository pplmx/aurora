package cmd

import (
	"fmt"
	"os"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/migrate"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var httpTimeout string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aurora",
	Short: i18n.GetText("app.name"),
	Long: `Aurora is a blockchain-based digital voting system with VRF lottery.

Features:
  - VRF random number generation
  - Blockchain storage
  - CLI and TUI interfaces
  - Database migrations

Use "aurora lottery --help" for lottery commands.`,
	Example: `  aurora lottery create -p "Alice,Bob,Charlie" -s "my-seed" -c 2
  aurora lottery tui
  aurora nft mint -n "MyNFT" -c "creator-key"
  aurora token create -n "MyToken" -s "MTK" --supply 1000000
  aurora voting candidate add -n "Alice" -p "Party"`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// NOTE: the previous version of this hook resolved a `data.dir`
		// (default $HOME/.aurora/data), mkdir'd it, and called app.Wire(dataDir),
		// stashing the result in the never-read GlobalApp. That ran on EVERY
		// subcommand, so even read-only `aurora version` created a phantom
		// $HOME/.aurora/data with an unused tokens.db/events.db/nonces.db trio
		// (TASK-103, ISS-095). The wiring was dead (nothing reads GlobalApp), so
		// it is removed; the only remaining pre-run work is the optional
		// migrate.autoRun, which targets the same blockchain.DBPath() every
		// store uses.
		if viper.GetBool("migrate.autoRun") {
			// Migrate the very same database every store and `aurora migrate`
			// use. blockchain.DBPath() honors a configured db.path; the previous
			// fallback here (filepath.Join(dataDir, "aurora.db"), i.e.
			// $HOME/.aurora/data/aurora.db) diverged from the stores'
			// ./data/aurora.db — the migrate-vs-stores split-brain (TASK-102,
			// ISS-094).
			dbPath := blockchain.DBPath()
			if dbPath == "" {
				return fmt.Errorf("failed to resolve database path: data directory unavailable")
			}
			migPath := viper.GetString("migrate.path")

			if err := migrate.RunMigrationsIfEnabled(dbPath, migrate.MigrateConfig{
				AutoMigrate: true,
				MigPath:     migPath,
			}); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logger.Error().Err(err).Msg("Application error")
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is 1. $HOME/aurora.toml 2. $PWD/config/aurora.toml)")
	rootCmd.PersistentFlags().StringVar(&httpTimeout, "http-timeout", "", "HTTP request timeout (e.g., 30s, 1m)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Add support for multiple config file paths, $HOME and ./config/
		viper.AddConfigPath(home)
		viper.AddConfigPath("./config/")

		// Add support for toml config file
		viper.SetConfigType("toml")
		viper.SetConfigName("aurora")
	}

	setDefaultConfig()

	_ = viper.BindPFlag("http.timeout", rootCmd.PersistentFlags().Lookup("http-timeout"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func setDefaultConfig() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	viper.SetDefault("migrate.autoRun", false)
	viper.SetDefault("migrate.path", "./migrations")
	viper.SetDefault("lottery.defaultCount", 3)
	viper.SetDefault("lottery.defaultSeedPrefix", "aurora-vrf-")
	viper.SetDefault("i18n.locale", "en")
	viper.SetDefault("http.timeout", "10s")
	viper.SetDefault("http.rateLimit.requests", 10)
	viper.SetDefault("http.rateLimit.window", "1m")
}
