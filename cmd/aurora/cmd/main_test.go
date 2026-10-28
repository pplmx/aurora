package cmd

import (
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the CLI's language.
//
// Commands print through i18n.GetText, which follows the locale environment;
// the tests assert the English wording. Without this, a full run under a
// Chinese locale failed a whole file of output assertions (backup, lottery,
// ...) that had nothing to do with the change under test (measured on 821b585,
// before the desktop localization work). Tests that are about localization set
// the locale or the translator themselves.
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}
