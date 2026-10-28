package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var httpTimeout string
var configErr error

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aurora",
	Short: i18n.GetText("app.name"),
	Long:  i18n.GetText("cli.root.long"),
	Example: `  aurora lottery create -p "Alice,Bob,Charlie" -s "my-seed" -c 2
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
		if configErr != nil {
			return configErr
		}

		// Cobra runs initConfig before this hook, so [log] settings are now
		// available. The bootstrap logger in main necessarily ran too early
		// and used console output.
		logger.Init()
		logger.Info().
			Str("version", Version).
			Str("build_time", BuildTime).
			Str("locale", i18n.GetTranslator().GetLocale()).
			Msg("Aurora starting")

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
			migPath := viper.GetString("migrate.path")

			if err := migrate.RunMigrationsIfEnabled(dbPath, migrate.MigrateConfig{
				AutoMigrate: true,
				MigPath:     migPath,
			}); err != nil {
				return wrapCLIError("cli.error.migration_failed", err)
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
		return newCLIError("cli.confirm.required", i18n.GetText(actionKey))
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
	if errors.Is(err, token.ErrAuditPublishFailed) {
		return i18n.GetTextF("cli.error.audit_committed", localizedErrorText(err))
	}
	return i18n.GetTextF("cli.error.prefix", localizedErrorText(err))
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.GetText("cli.config_flag"))
	rootCmd.PersistentFlags().StringVar(&httpTimeout, "http-timeout", "", i18n.GetText("cli.http_timeout_flag"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	configErr = nil
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

	// A missing optional config during the $HOME/cwd search is fine. A missing
	// explicit --config or any parse/read failure must abort before a command
	// can run against defaults the operator did not intend.
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if cfgFile != "" || !errors.As(err, &notFound) {
			configErr = wrapCLIError("cli.error.read_config", err)
		}
		return
	}
	// main detects the process locale before Cobra parses flags or reads the
	// config. Reinitialize after ReadInConfig so a configured [i18n] locale
	// reaches command output, while an explicit locale environment still wins
	// inside DetectAndInit.
	i18n.DetectAndInit()
	fmt.Fprintln(os.Stderr, i18n.GetTextF("cli.using_config_file", viper.ConfigFileUsed()))
}

func setDefaultConfig() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	viper.SetDefault("migrate.autoRun", false)
	viper.SetDefault("migrate.path", "./migrations")
	viper.SetDefault("lottery.defaultCount", 3)
	viper.SetDefault("i18n.locale", "en")
	viper.SetDefault("http.timeout", "10s")
	viper.SetDefault("http.rateLimit.requests", 10)
	viper.SetDefault("http.rateLimit.window", "1m")
}

// printField prints one indented "Label: value" line under a command's result
// header. Four modules wrote this line by hand 55 times; the alignment is part
// of the CLI's output contract, so it lives in one place.
func printField(label, value string) {
	fmt.Printf("   %s: %s\n", label, value)
}

// printLocalizedField resolves result labels at print time so command output
// follows the active locale even if it changed after command registration.
func printLocalizedField(key, value string) {
	printField(i18n.GetText(key), value)
}
