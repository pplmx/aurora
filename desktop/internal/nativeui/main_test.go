package nativeui

import (
	"os"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMain pins the package's language. Every page label now comes from
// internal/i18n and follows the locale environment; without this, a developer
// running the suite under a Chinese locale would see hundreds of English
// assertions fail for reasons unrelated to the change under test. The
// localization tests set the language themselves and restore it.
func TestMain(m *testing.M) {
	i18n.Init("en")
	// A view test has no running app, so the native clipboard would wait for a
	// main thread that never runs. Fail fast instead of hanging the suite when a
	// test copies a private key without recordClipboard.
	writeClipboard = func(c *ui.Context, text string, concealed bool) {
		if concealed {
			panic("nativeui: a test copied confidential text without recordClipboard")
		}
		c.Services().WriteClipboard(text)
	}
	os.Exit(m.Run())
}
