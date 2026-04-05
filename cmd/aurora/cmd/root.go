package cmd

import (
	"fmt"
	"os"

	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aurora",
	Short: "Aurora - VRF-based transparent lottery system",
	Long: `Aurora is a blockchain-based digital voting system with VRF lottery.

Features:
  - VRF random number generation
  - Blockchain storage
  - CLI and TUI interfaces

Use "aurora lottery --help" for lottery commands.`,
	SilenceUsage: true,
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

	// TODO: THIS IS JUST FOR TESTING PURPOSES, WHICH CAN BE REMOVED
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func setDefaultConfig() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./log")
	viper.SetDefault("lottery.default_count", 3)
	viper.SetDefault("lottery.default_seed_prefix", "aurora-vrf-")
}
