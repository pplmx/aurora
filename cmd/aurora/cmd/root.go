// Package cmd wires the CLI: one file per command group, each a thin shell over
// the application layer.
//
// A command parses flags, calls one use case and prints through internal/i18n;
// the business rule lives in internal/app, which is why the same operation
// behaves identically from the window and the REST route.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/migrate"
	"github.com/pplmx/aurora/internal/infra/store"
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
		if err := validateHTTPTimeout(); err != nil {
			return err
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
		// migrate.autoRun, which targets the same store.Path() every
		// store uses.
		if viper.GetBool("migrate.autoRun") {
			// Migrate the very same database every store and `aurora migrate`
			// use. store.Path() honors a configured db.path; the previous
			// fallback here (filepath.Join(dataDir, "aurora.db"), i.e.
			// $HOME/.aurora/data/aurora.db) diverged from the stores'
			// ./data/aurora.db — the migrate-vs-stores split-brain (TASK-102,
			// ISS-094).
			dbPath := store.Path()
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
// command, mirroring `backup restore` and `lottery reset`. Burn/delete/down
// permanently destroy
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

// validateHTTPTimeout rejects an explicit --http-timeout the fetcher cannot
// use. config.DurationSeconds quietly falls back to the default for a value it
// cannot parse, so a typo like "1am" left the operator with a 10s timeout and
// fetches that failed as if the network were at fault.
func validateHTTPTimeout() error {
	value := strings.TrimSpace(httpTimeout)
	if value == "" {
		return nil
	}
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return nil
	}
	// The same bare-seconds form the config file accepts ("10" means 10s).
	if n, err := strconv.Atoi(value); err == nil && n > 0 {
		return nil
	}
	return newCLIError("cli.error.invalid_http_timeout", value)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	configErr = nil
	if cfgFile != "" {
		// Use config file from the flag. A directory or an extension-less path
		// would otherwise fail inside Viper as "Unsupported Config Type".
		if err := config.ValidateExplicitConfigPath(cfgFile); err != nil {
			configErr = wrapCLIError("cli.error.read_config", err)
			return
		}
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
	// The flag shadows http.timeout and the root hook validates it with its own
	// message, so a flag typo is never blamed on the config file.
	if httpTimeout == "" {
		if err := config.ValidateDurationValues(); err != nil {
			configErr = wrapCLIError("cli.error.invalid_duration", err)
			return
		}
	}
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
	printFields(field{label, value})
}

// field is one label and its value, for printFields.
type field struct {
	label string
	value string
}

// printFields prints a block of "Label: value" lines with the values in one
// column, the way `kubectl describe` sets a field block. A block whose labels
// differ in length ("ID", "Total Supply", "Owner") printed a ragged left edge
// and the eye had to re-find the value on every line.
//
// The column is measured in display columns, not bytes or runes: a Han label is
// two columns per character, so padding by byte length would leave the Chinese
// block worse aligned than doing nothing. The separator is the language's own,
// so a Chinese block keeps its full-width colon.
func printFields(fields ...field) {
	width := 0
	for _, f := range fields {
		if w := i18n.DisplayWidth(f.label); w > width {
			width = w
		}
	}
	for _, f := range fields {
		pad := strings.Repeat(" ", width-i18n.DisplayWidth(f.label))
		fmt.Printf("   %s%s %s%s\n", f.label, i18n.Colon(), pad, f.value)
	}
}

// markedField is a field whose line opens with a status mark (the doctor's
// ✔/▲/✖). The mark sits in its own column: a block that mixed marked and
// unmarked rows would otherwise start its labels one column apart.
type markedField struct {
	mark  string
	label string
	value string
}

// printMarkedFields prints markedField rows from one indent, with the labels
// lined up under each other.
func printMarkedFields(fields ...markedField) {
	width := 0
	for _, f := range fields {
		if w := i18n.DisplayWidth(f.label); w > width {
			width = w
		}
	}
	for _, f := range fields {
		pad := strings.Repeat(" ", width-i18n.DisplayWidth(f.label))
		fmt.Printf("  %s %s%s %s%s\n", f.mark, f.label, i18n.Colon(), pad, f.value)
	}
}

// addField appends one label/value pair for printFields, skipping an optional
// field that has no value so the block does not print a label with nothing after
// it (an NFT without a description, an oracle source without a path).
func addField(fields []field, label, value string) []field {
	if strings.TrimSpace(value) == "" {
		return fields
	}
	return append(fields, field{label, value})
}

// addLocalizedField is addField with the label resolved through the catalog.
func addLocalizedField(fields []field, key, value string) []field {
	return addField(fields, i18n.GetText(key), value)
}

// printLocalizedField resolves result labels at print time so command output
// follows the active locale even if it changed after command registration.
func printLocalizedField(key, value string) {
	printField(i18n.GetText(key), value)
}
