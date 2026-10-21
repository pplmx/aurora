package deskapp

import (
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the package's language.
//
// Every user-facing string in this package now comes from internal/i18n,
// which follows LANG. Without this, the suite's result depended on the
// developer's environment: a full run under LANG=zh_CN failed a dozen
// assertions on English wording that had nothing to do with the change being
// tested (measured). Tests that are about localization set the language
// themselves (withLocale).
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}
