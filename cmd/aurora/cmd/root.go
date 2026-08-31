package cmd

import (
	"errors"
	"fmt"
	"os"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	tokenerrors "github.com/pplmx/aurora/internal/domain/token"
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
	Long: `Aurora is a blockchain-based digital systems suite: VRF lottery,
signed voting, data oracle, NFT and fungible-token ledgers.

Features:
  - VRF random number generation
  - Ed25519-signed voting and NFT transfers
  - Blockchain storage with integrity verification
  - CLI, TUI and web interfaces
  - Database migrations and backups

Use "aurora <module> --help" for module commands (lottery, voting, oracle,
nft, token).`,
	Example: `  aurora lottery create -p "Alice,Bob,Charlie" -s "my-seed" -c 2
  aurora lottery tui
  aurora nft mint -n "MyNFT" -c "creator-key"
  aurora token create -n "MyToken" -s "MTK" --supply 1000000
  aurora voting candidate add -n "Alice" -p "Party"`,
	// SilenceUsage AND SilenceErrors: with only SilenceUsage set, cobra printed
	// "Error: ..." to stderr on every failure and Execute() then printed the
	// same message again as "❌ Error: ..." — every CLI error appeared twice.
	// The single formatted error line from Execute() (plus the structured
	// logger line) is the one true error surface.
	SilenceErrors: true,
	SilenceUsage:  true,
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

// addConfirmFlag registers the standard -y/--confirm guard on a destructive
// command, mirroring `backup restore --confirm` (backup.go:91) and
// `lottery reset --yes` (lottery.go:557). Burn/delete/down permanently destroy
// value or data, so they must not run without an explicit confirmation —
// otherwise a typo like a mis-ordered flag silently destroys assets.
//
// actionKey names a cli.confirm.* i18n key whose localized value describes
// what the command destroys; the flag help renders it through the shared
// cli.confirm.flag template so all six destructive commands speak one
// localized wording (TASK-186, ISS-182).
func addConfirmFlag(cmd *cobra.Command, actionKey string) {
	cmd.Flags().BoolP("confirm", "y", false, fmt.Sprintf(i18n.GetText("cli.confirm.flag"), i18n.GetText(actionKey)))
}

// requireConfirm returns an error (→ non-zero exit) when the caller's --confirm
// flag is not set, so scripts that forgot it can detect the refusal. The refusal
// renders the localized cli.confirm.required template with the command's
// localized action phrase (TASK-186, ISS-182).
func requireConfirm(cmd *cobra.Command, actionKey string) error {
	ok, _ := cmd.Flags().GetBool("confirm")
	if !ok {
		return fmt.Errorf(i18n.GetText("cli.confirm.required"), i18n.GetText(actionKey))
	}
	return nil
}

// formatCLIError renders the single error line Execute() writes to stderr.
// A token operation that COMMITTED but lost its post-commit audit event must
// never be shown as a plain failure — that framing invites a retry that would
// repeat already-committed money movement. It gets a distinct do-not-retry
// warning instead (exit code stays nonzero so the audit gap surfaces in
// scripts; the message is explicit that the operation did commit).
func formatCLIError(err error) string {
	if errors.Is(err, tokenerrors.ErrAuditPublishFailed) {
		return fmt.Sprintf(
			"⚠ %v\n  The operation DID commit. Do NOT retry it — inspect the event store.\n", err,
		)
	}
	return fmt.Sprintf("❌ Error: %v\n", err)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logger.Error().Err(err).Msg("Application error")
		fmt.Fprint(os.Stderr, formatCLIError(err))
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
