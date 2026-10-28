package nativeui

import (
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the package's language. Every page label now comes from
// internal/i18n and follows the locale environment; without this, a developer
// running the suite under a Chinese locale would see hundreds of English
// assertions fail for reasons unrelated to the change under test. The
// localization tests set the language themselves and restore it.
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}
