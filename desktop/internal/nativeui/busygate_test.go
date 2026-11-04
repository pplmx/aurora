package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEveryBusyGateIsWired collects the controls that must stop working while a
// write owns the database. Each one is a doorway to storage: the menus and
// shortcuts start file operations, Refresh starts a read, the palette runs a
// ledger search, and the page body holds the write forms.
//
// The list is checked against the source because ui.Disabled is a paint-time
// state: the tester can still click a disabled control, so a window test would
// pass whether or not the gate was wired.
func TestEveryBusyGateIsWired(t *testing.T) {
	sources := map[string]string{
		"actions.go":       readFile(t, "actions.go"),
		"chrome.go":        readFile(t, "chrome.go"),
		"app.go":           readFile(t, "app.go"),
		"palettesearch.go": readFile(t, "palettesearch.go"),
	}

	for _, tc := range []struct{ file, needle, why string }{
		{"actions.go", "Disabled(a.busy != \"\")", "the File menu button starts backups and imports"},
		{"actions.go", "if a.busy != \"\" {\n\t\treturn\n\t}", "the export/import/backup shortcuts must not fire"},
		{"chrome.go", "Disabled(a.busy != \"\")", "Refresh must not read while a write owns the store"},
		{"app.go", "Disabled(a.busy != \"\").Children", "the page body holds the write forms"},
		{"palettesearch.go", "a.busy != \"\"", "the palette's ledger search is a read"},
	} {
		require.Containsf(t, sources[tc.file], tc.needle,
			"%s: %s", tc.file, tc.why)
	}
}

// TestThePageBodyIsDisabledNotJustTheButtons: the forms are inside one disabled
// subtree rather than each button carrying its own flag. A form that added a
// control without a Disabled call would otherwise be live during a write.
func TestThePageBodyIsDisabledNotJustTheButtons(t *testing.T) {
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()

	app.busy = "Creating token…"
	tester.Frame()

	// The subtitle under the title still draws, so the window is not blank; the
	// forms simply cannot be operated.
	require.True(t, tester.HasText("Token"), "the page still names itself while busy")
	require.NotEmpty(t, app.busy, "and the banner names the running write")
}
