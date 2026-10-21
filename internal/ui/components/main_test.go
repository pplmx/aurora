package components

import (
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the TUI's language.
//
// The shared views read i18n.GetText, which follows LANG, while the tests
// assert the English wording (e.g. the help screen's "scroll" hint). Without
// this, a full run under LANG=zh_CN failed them for reasons unrelated to the
// change under test (measured on 821b585).
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}
