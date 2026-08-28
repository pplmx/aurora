package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	tokenerrors "github.com/pplmx/aurora/internal/domain/token"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatCLIError_CommittedAuditFailure locks the CLI half of TASK-117 /
// ISS-109: a token op that COMMITTED but whose post-commit audit publish
// failed must render as a do-not-retry warning, never as a bare "❌ Error:"
// failure line (which invites a retry that repeats the money movement).
func TestFormatCLIError_CommittedAuditFailure(t *testing.T) {
	// Wrapped exactly as internal/domain/token/service.go wraps its
	// post-commit publish sites: sentinel as %w, cause as %v.
	err := fmt.Errorf("failed to transfer: %w", fmt.Errorf("%w: %v", tokenerrors.ErrAuditPublishFailed, errors.New("disk full")))

	out := formatCLIError(err)
	if !strings.Contains(out, "committed") {
		t.Errorf("audit-failure line must say the op committed, got: %q", out)
	}
	if !strings.Contains(out, "Do NOT retry") {
		t.Errorf("audit-failure line must warn against retrying, got: %q", out)
	}
	if strings.Contains(out, "❌ Error:") {
		t.Errorf("audit-failure must not render as a plain failure line, got: %q", out)
	}

	// Ordinary failures keep the established single error line.
	if got := formatCLIError(errors.New("boom")); got != "❌ Error: boom\n" {
		t.Errorf("ordinary error rendered as %q, want the standard error line", got)
	}
}

// TestVersionCmdReportsRealValues pins the TASK-125 fix: the version command
// must surface the related link-time/build-time variables and the real Go
// toolchain — not hardcoded placeholders ("0.0.1"/"1.26+") that ignored
// ldflags and misrepresented the build.
func TestVersionCmdReportsRealValues(t *testing.T) {
	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	// SetOut(nil) clears the outWriter override so later tests (which may swap
	// os.Stdout via runCmd's capture) are routed to the *current* stdout, not
	// the one observed here.
	t.Cleanup(func() { versionCmd.SetOut(nil) })
	require.NoError(t, versionCmd.RunE(versionCmd, nil))
	out := buf.String()

	assert.Contains(t, out, "Version: "+Version)
	assert.Contains(t, out, "Go Version: "+runtime.Version())
	// Regression guard: the fabricated string must never reappear.
	assert.NotContains(t, out, "1.26+")
}

func TestSetDefaultConfig(t *testing.T) {
	viper.Reset()
	setDefaultConfig()

	for key, want := range map[string]any{
		"log.level":                 "info",
		"log.path":                  "./logs",
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

// TestNoCommandSilentlySwallowsErrors is the ISS-083 regression at the
// structure level: a command that defines Run (instead of RunE) and does
// fmt.Println("Error:", err) inside can never fail — Execute() sees nil and the
// process exits 0, so $?-checking scripts/CI report success on a failed run
// (the `nft tui` / `oracle tui` pattern). Every executable command in the tree
// must use RunE; a bare Run whose errors it could swallow is forbidden.
func TestNoCommandSilentlySwallowsErrors(t *testing.T) {
	var runOnly []string
	walkCommands(rootCmd, func(name string, c *cobra.Command) {
		if c.RunE == nil && c.Run != nil {
			runOnly = append(runOnly, name)
		}
	})
	require.Empty(t, runOnly, "commands must use RunE so failures exit 1; Run-only: %v", runOnly)
}

func walkCommands(c *cobra.Command, fn func(name string, cmd *cobra.Command)) {
	for _, sub := range c.Commands() {
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		fn(sub.Name(), sub)
		walkCommands(sub, fn)
	}
}

func childNames(c *cobra.Command) []string {
	var names []string
	for _, sub := range c.Commands() {
		names = append(names, sub.Name())
	}
	sort.Strings(names)
	return names
}

// TestNoPhantomHomeDataDir locks TASK-103/ISS-095: the previous
// PersistentPreRunE ran app.Wire(dataDir) on every subcommand and stashed it
// in the never-read GlobalApp, so even read-only commands created a phantom
// $HOME/.aurora/data with an unused tokens/events/nonces .db triple. With the
// dead wiring removed, a full command run (PersistentPreRunE intact) must not
// create anything under $HOME.
func TestNoPhantomHomeDataDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	viper.Reset()
	viper.Set("migrate.autoRun", false)

	prev := rootCmd.PersistentPreRunE
	t.Cleanup(func() { rootCmd.PersistentPreRunE = prev })
	rootCmd.PersistentPreRunE = prev // ensure the real hook is in place

	resetFlags(rootCmd)
	rootCmd.SetArgs([]string{"version"})
	capture := captureStdout(t)
	err := rootCmd.Execute()
	_ = capture()
	require.NoError(t, err)

	if _, statErr := os.Stat(filepath.Join(home, ".aurora")); statErr == nil {
		t.Fatal("read-only command must not create $HOME/.aurora (phantom app.Wire data dir removed)")
	}
}

func TestVersionCmd_HappyPath(t *testing.T) {
	resetCliForTest()
	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "Aurora")
	assert.Contains(t, out, "Version: 0.0.1")
}
