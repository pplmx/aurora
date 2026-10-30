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

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/logger"
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
	err := fmt.Errorf("failed to transfer: %w", fmt.Errorf("%w: %v", token.ErrAuditPublishFailed, errors.New("disk full")))

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
		"log.level":               "info",
		"log.path":                "./logs",
		"migrate.autoRun":         false,
		"migrate.path":            "./migrations",
		"lottery.defaultCount":    3,
		"i18n.locale":             "en",
		"http.timeout":            "10s",
		"http.rateLimit.requests": 10,
		"http.rateLimit.window":   "1m",
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
	assert.NoError(t, configErr, "a config absent from the optional search paths must use defaults")
}

func TestCliConfigLocaleAppliesWithoutLocaleEnvironment(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "aurora.toml")
		require.NoError(t, os.WriteFile(configPath, []byte("[i18n]\nlocale = \"zh-Hans\"\n"), 0o600))

		for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
			t.Setenv(key, "")
		}
		viper.Reset()
		cfgFile = ""
		t.Cleanup(func() {
			i18n.Init("en")
			logger.Close()
		})

		resetFlags(rootCmd)
		rootCmd.SetArgs([]string{"--config", configPath, "version"})
		capture := captureStdout(t)
		captureErr := captureStderr(t)
		err := rootCmd.Execute()
		out := capture()
		errOut := captureErr()

		require.NoError(t, err)
		assert.Contains(t, out, "版本:")
		assert.NotContains(t, out, "Version:")
		assert.Contains(t, errOut, "使用配置文件：")
		assert.NotContains(t, errOut, "Using config file:")
	})
}

// TestRootCommandInitializesFileLoggingFromConfig exercises the real Cobra
// lifecycle with an alternate config file. main used to initialize the logger
// before Cobra loaded [log], so the empty level disabled output and the
// configured path was never opened.
func TestRootCommandInitializesFileLoggingFromConfig(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		viper.Reset()
		logPath := filepath.Join(t.TempDir(), "configured-logs")
		configPath := filepath.Join(t.TempDir(), "aurora.toml")
		config := fmt.Appendf(nil, "[log]\nlevel = \"info\"\npath = %q\n", logPath)
		require.NoError(t, os.WriteFile(configPath, config, 0o600))

		previousConfig := cfgFile
		cfgFile = ""
		t.Cleanup(func() { cfgFile = previousConfig })
		t.Cleanup(logger.Close)

		resetFlags(rootCmd)
		rootCmd.SetArgs([]string{"--config", configPath, "version"})
		require.NoError(t, rootCmd.Execute())

		data, err := os.ReadFile(filepath.Join(logPath, "aurora.log"))
		require.NoError(t, err, "the configured CLI log file must be created")
		assert.Contains(t, string(data), "Aurora starting")
	})
}

// TestRootCommandRejectsInvalidExplicitConfig makes sure a bad operator config
// cannot silently fall back to default DB and logging paths. Only the absence
// of an optional config during the normal $HOME/cwd search is benign.
func TestRootCommandRejectsInvalidExplicitConfig(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		malformed := filepath.Join(t.TempDir(), "malformed.toml")
		require.NoError(t, os.WriteFile(malformed, []byte("[db\npath = \"unterminated\"\n"), 0o600))

		for _, tc := range []struct {
			name string
			path string
		}{
			{name: "malformed", path: malformed},
			{name: "explicitly missing", path: filepath.Join(t.TempDir(), "missing.toml")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				viper.Reset()
				resetFlags(rootCmd)
				rootCmd.SetArgs([]string{"--config", tc.path, "version"})

				capture := captureStdout(t)
				err := rootCmd.Execute()
				stdout := capture()

				require.Error(t, err)
				assert.Contains(t, err.Error(), "config")
				assert.Empty(t, stdout, "a config error must not write a structured log into command output")
			})
		}
	})
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

// TestRootHelpDescribesCurrentSurfaces keeps the first help screen from
// advertising the Web UI deleted when the desktop moved to native widgets.
// Aurora now ships a native desktop window and a JSON-only REST API.
func TestRootHelpDescribesCurrentSurfaces(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	tr.SetLocale("en")
	t.Cleanup(func() { tr.SetLocale(before) })

	out := renderCommandHelp(t, rootCmd)
	assert.Contains(t, out, "native desktop window and REST API")
	assert.NotContains(t, strings.ToLower(out), "web")
}

// TestRootHelpLocalizesCobraFrame covers the labels generated by Cobra rather
// than declared in Aurora's command files. Without a custom usage renderer,
// Chinese command summaries were surrounded by "Usage:" / "Available
// Commands:" / "Flags:" and the built-in help/completion descriptions.
func TestRootHelpLocalizesCobraFrame(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	tr.SetLocale("zh")
	t.Cleanup(func() { tr.SetLocale(before) })

	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	out := renderCommandHelp(t, rootCmd)
	for _, want := range []string{
		"用法：",
		"可用命令：",
		"参数：",
		"显示 aurora 的帮助",
		"显示任意命令的帮助",
		"生成指定 shell 的自动补全脚本",
		"显示版本信息",
		`运行 "aurora [command] --help" 查看命令的详细帮助。`,
	} {
		assert.Contains(t, out, want)
	}
	for _, unwanted := range []string{
		"Usage:",
		"Available Commands:",
		"Flags:",
		"Help about any command",
		"Show version information",
	} {
		assert.NotContains(t, out, unwanted)
	}
}

func TestChildHelpLocalizesGeneratedSections(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	tr.SetLocale("zh")
	t.Cleanup(func() { tr.SetLocale(before) })

	resetFlags(lotteryCmd)
	out := renderCommandHelp(t, lotteryCmd)
	assert.Contains(t, out, "用法：")
	assert.Contains(t, out, "可用命令：")
	assert.Contains(t, out, "参数：")
	assert.Contains(t, out, "全局参数：")
	assert.Contains(t, out, `运行 "aurora lottery [command] --help" 查看命令的详细帮助。`)
	assert.NotContains(t, out, "Global Flags:")
}

func renderCommandHelp(t *testing.T, c *cobra.Command) string {
	t.Helper()
	var buf bytes.Buffer
	c.SetOut(&buf)
	t.Cleanup(func() { c.SetOut(nil) })
	require.NoError(t, c.Help())
	return buf.String()
}

// TestFlagOnlyCommandsRejectStrayArgs is the ISS-179 regression at the
// structure level: every subcommand under token/nft/oracle/voting previously
// used Cobra's default ArbitraryArgs, silently ACCEPTING AND IGNORING stray
// positional args — a typo'd/misplaced argument (a copy-pasted id next to a
// flag-only command, or flags mixed from two commands) ran with surprising
// behavior and no diagnostic. Each of these commands now declares
// Args: cobra.NoArgs (lottery/backup/migrate already validated their own), and
// this pins that any future subcommand in the four flag-only modules carries
// an explicit validator instead of the silent-default.
func TestFlagOnlyCommandsRejectStrayArgs(t *testing.T) {
	var missing []string
	for _, root := range []*cobra.Command{tokenCmd, nftCmd, oracleCmd, votingCmd} {
		walkCommands(root, func(name string, c *cobra.Command) {
			if c.Args == nil {
				missing = append(missing, root.Name()+"/"+name)
			}
		})
	}
	require.Empty(t, missing, "flag-only subcommands must declare an Args validator so stray args error instead of being silently dropped: %v", missing)
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
	viper.Set("log.path", filepath.Join(t.TempDir(), "logs"))
	t.Cleanup(logger.Close)

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

// TestValidateHTTPTimeout covers the explicit flag the fetcher cannot use:
// config.DurationSeconds quietly falls back to the default, so a typo would
// otherwise show up as a fetch that failed for "network" reasons.
func TestValidateHTTPTimeout(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"", true},
		{"30s", true},
		{"1m", true},
		{"10", true},
		{"nonsense", false},
		{"1am", false},
		{"0", false},
		{"0s", false},
		{"-5s", false},
	}
	for _, tc := range cases {
		httpTimeout = tc.value
		err := validateHTTPTimeout()
		if tc.ok {
			require.NoErrorf(t, err, "%q must be accepted", tc.value)
			continue
		}
		require.Errorf(t, err, "%q must be rejected", tc.value)
		assert.Contains(t, err.Error(), "--http-timeout")
	}
	httpTimeout = ""
}

// TestInvalidConfigDurationAbortsBeforeCommand: a typo in a duration key must
// stop the command, like an unreadable config, instead of running on a default
// the operator did not choose.
func TestInvalidConfigDurationAbortsBeforeCommand(t *testing.T) {
	resetCliForTest()
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, "aurora.toml"),
		[]byte("[http]\ntimeout = \"1am\"\n"), 0o644))
	t.Setenv("HOME", home)
	t.Cleanup(viper.Reset)

	resetFlags(rootCmd)
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http.timeout")
	assert.Contains(t, err.Error(), "duration")
}

// TestInvalidHTTPTimeoutFlagNamesTheFlag: the config-file validator must not
// blame the file for a bad --http-timeout; the flag has its own message.
func TestInvalidHTTPTimeoutFlagNamesTheFlag(t *testing.T) {
	resetCliForTest()
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(viper.Reset)

	resetFlags(rootCmd)
	rootCmd.SetArgs([]string{"--http-timeout", "1am", "version"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--http-timeout")
	assert.NotContains(t, err.Error(), "config file")
}
