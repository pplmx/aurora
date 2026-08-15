package cmd

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGoVersion(t *testing.T) {
	assert.Equal(t, "1.26+", getGoVersion())
}

func TestSetDefaultConfig(t *testing.T) {
	viper.Reset()
	setDefaultConfig()

	for key, want := range map[string]any{
		"log.level":                 "info",
		"log.path":                  "./logs",
		"data.dir":                  "",
		"migrate.autoRun":           false,
		"migrate.path":              "./migrations",
		"lottery.defaultCount":      3,
		"lottery.defaultSeedPrefix": "aurora-vrf-",
		"i18n.locale":               "en",
		"http.timeout":              "10s",
		"http.rateLimit.requests":   10,
		"http.rateLimit.window":     "1m",
	} {
		assert.Equal(t, want, viper.Get(key), "default %q", key)
	}
}

// TestInitConfig_FallbackPath exercises the no-flag path of initConfig:
// it reads defaults from the process environment and the cwd without a
// --config flag, and must not fail while doing so.
func TestInitConfig_FallbackPath(t *testing.T) {
	cfgFile = ""
	viper.Reset()

	// initConfig calls viper.AddConfigPath(home) and ./config/, then
	// ReadInConfig. Without an aurora.toml present it is a benign no-op
	// (ReadInConfig returns an error that initConfig ignores). We only
	// assert it runs without panicking and leaves defaults in place.
	initConfig()

	assert.Equal(t, "info", viper.GetString("log.level"), "defaults should be set by initConfig")
}

// TestRootCommandTree asserts that all five module commands plus version
// are registered under rootCmd. A command added to a module file but
// forgotten in init() would go undetected here; wait — the modules call
// rootCmd.AddCommand from their own init(), so the tree is the source of
// truth for "what the user can invoke".
func TestRootCommandTree(t *testing.T) {
	cmds := map[string]*cobra.Command{}
	for _, c := range rootCmd.Commands() {
		cmds[c.Name()] = c
	}

	for _, name := range []string{"lottery", "nft", "oracle", "token", "voting", "version"} {
		assert.Contains(t, cmds, name, "root command should register %q", name)
	}

	// version is a leaf with no subcommands.
	v := cmds["version"]
	require.NotNil(t, v)
	assert.NotEmpty(t, v.Short)

	// each module command should have at least one RunE-bearing child or be
	// itself executable; sanity check the two richest modules.
	uilottery := cmds["lottery"]
	names := childNames(uilottery)
	assert.Subset(t, names, []string{"create", "history", "verify", "export", "import", "stats"})
}

func childNames(c *cobra.Command) []string {
	var names []string
	for _, sub := range c.Commands() {
		names = append(names, sub.Name())
	}
	sort.Strings(names)
	return names
}

func TestVersionCmd_HappyPath(t *testing.T) {
	resetCliForTest()
	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "Aurora")
	assert.Contains(t, out, "Version: 0.0.1")
}
