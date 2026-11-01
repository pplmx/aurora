package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the CLI's language and guards the checkout against a test
// leaving a database behind.
//
// Commands print through i18n.GetText, which follows the locale environment;
// the tests assert the English wording. Without this, a full run under a
// Chinese locale failed a whole file of output assertions (backup, lottery,
// ...) that had nothing to do with the change under test (measured on 821b585,
// before the desktop localization work). Tests that are about localization set
// the locale or the translator themselves.
//
// The second guard is about isolation. The store resolves db.path on first open
// and defaults to ./data, so a test that opens it without first moving to a
// temp directory writes aurora.db into the checkout. That is how an earlier
// flaky repocheck failure started: a sibling test's ResetForTest removed ./data
// mid-walk while another binary was still reading it. Every test here uses
// withTempDir, but nothing enforced the convention — a new test could forget it
// and pass. A data directory that this run created did not exist before it
// started, so its appearance is attributable.
func TestMain(m *testing.M) {
	// initConfig can re-detect the locale after loading a config file, so
	// pin the environment as well as the translator for deterministic tests.
	os.Setenv("LC_ALL", "en_US.UTF-8")
	os.Unsetenv("LC_MESSAGES")
	os.Unsetenv("LANGUAGE")
	os.Unsetenv("LANG")
	i18n.Init("en")

	_, err := os.Stat("data")
	dataExistedBefore := err == nil

	code := m.Run()

	// os.Exit skips deferred functions, so the check runs before it rather than
	// as a cleanup. A checkout that already had ./data is left alone: a
	// developer who ran the CLI is not the suite's problem.
	if !dataExistedBefore {
		if _, err := os.Stat("data"); err == nil {
			fmt.Fprintln(os.Stderr,
				"cmd tests wrote ./data into the checkout: a test opened the store "+
					"without moving to a temp directory first (see withTempDir)")
			code = 1
		}
	}
	os.Exit(code)
}
