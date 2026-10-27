package deskapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportReportIsOneContract pins the shared wording. Before this existed,
// File > Import and a dropped/opened file built their own messages: a partial
// import from the menu listed the rejection reasons and the 0-based indices,
// while the same file dropped on the window omitted both, so the user could
// not tell which records failed or why.
func TestImportReportIsOneContract(t *testing.T) {
	ok := lotteryapp.ImportResult{Total: 3, Imported: 3}
	title, message, detail := importReport(ok, "/tmp/draws.json")
	assert.Equal(t, "Lotteries imported", title)
	assert.Equal(t, "3 draws imported.", message)
	assert.Equal(t, "/tmp/draws.json", detail)
	assert.Equal(t, mygo.MessageInfo, importReportType(ok))

	partial := lotteryapp.ImportResult{Total: 3, Imported: 2, Rejected: []int{1}}
	title, message, detail = importReport(partial, "/tmp/draws.json")
	assert.Equal(t, "Import partially failed", title)
	assert.Equal(t, "Imported 2 of 3 lottery records (failed: 1).", message,
		"a partial import must never read as a plain success")
	assert.Equal(t, mygo.MessageWarning, importReportType(partial),
		"the valid records were stored, so this is a warning, not an error")

	// The detail must carry everything a user needs to act: which indices
	// failed, why, and which file they came from.
	for _, want := range []string{"[1]", "0-based", "already exists", "/tmp/draws.json"} {
		assert.Contains(t, detail, want)
	}
}

// TestShellAndMenuShareImportReporting is the anti-drift guard: the shell's
// dialog must be exactly what the menu would show for the same result.
func TestShellAndMenuShareImportReporting(t *testing.T) {
	shell := NewShell(About{})
	var gotType mygo.MessageType
	var got [3]string
	shell.report = func(typ mygo.MessageType, title, message, detail string) {
		gotType, got = typ, [3]string{title, message, detail}
	}

	result := lotteryapp.ImportResult{Total: 2, Imported: 1, Rejected: []int{0}}
	shell.reportImport("/tmp/draws.json", result)

	title, message, detail := importReport(result, "/tmp/draws.json")
	assert.Equal(t, mygo.MessageWarning, gotType,
		"a dropped partial import must be a warning, like the menu's")
	require.Equal(t, [3]string{title, message, detail}, got,
		"the dropped-file dialog must be built by the same function as the menu's")
}

// TestShellReportsWithTheMenuSeverity closes the other half of the drift: the
// shell used to hand every message to showMessage, which hard-coded
// MessageInfo. A failed import or an unsupported drop therefore appeared under
// a blue info icon, while the same failure from File > Import was a red error
// dialog, and a partial import lost the warning importReportType had just
// computed.
func TestShellReportsWithTheMenuSeverity(t *testing.T) {
	type dialog struct {
		typ     mygo.MessageType
		title   string
		message string
	}
	shell := NewShell(About{DBPath: filepath.Join(t.TempDir(), "aurora.db")})
	var dialogs []dialog
	shell.report = func(typ mygo.MessageType, title, message, _ string) {
		dialogs = append(dialogs, dialog{typ, title, message})
	}

	// An import that cannot even open the database is an error, not a notice.
	shell.OpenFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Len(t, dialogs, 1)
	assert.Equal(t, mygo.MessageError, dialogs[0].typ)
	assert.Equal(t, "Import failed", dialogs[0].title)

	// Dropping something that is not an export is an error too.
	other := filepath.Join(t.TempDir(), "report.csv")
	require.NoError(t, os.WriteFile(other, []byte("a,b"), 0o644))
	shell.ImportDropped([]string{other})
	require.Len(t, dialogs, 2)
	assert.Equal(t, mygo.MessageError, dialogs[1].typ, "an unsupported drop must look like a failure")

	// A bad link is reported by the shell as well.
	shell.OpenURL("https://example.com")
	require.Len(t, dialogs, 3)
	assert.Equal(t, mygo.MessageError, dialogs[2].typ)
}

// TestShowMessageForwardsTheSeverity is the end of the wire the shell tests
// cannot reach: they replace Shell.report, so a dialog constructor that
// hard-coded MessageInfo would still pass them while every real message came
// out blue. This test replaces the dialog itself.
func TestShowMessageForwardsTheSeverity(t *testing.T) {
	original := messageBox
	t.Cleanup(func() { messageBox = original })

	var got mygo.MessageOptions
	messageBox = func(opts mygo.MessageOptions) { got = opts }

	showMessage(nil, mygo.MessageError, "Import failed", "boom", "/tmp/draws.json")
	assert.Equal(t, mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   "Import failed",
		Message: "boom",
		Detail:  "/tmp/draws.json",
	}, got)
}

// TestDialogsWithoutAWindowStandAlone pins the half of the parenting contract
// a unit test can reach: MyGo attaches a dialog with a Parent to that window
// (a sheet on macOS, a modal child elsewhere), and with none it stands alone.
//
// The other half - that the window MyGo hands a menu click reaches the dialog -
// is enforced by the signatures: exportLotteries, importLotteries,
// backUpDatabase and showAbout take the window and pass it through parent(),
// so a handler cannot silently drop it the way the old `func(*mygo.MenuItem,
// *mygo.Window)` closures did.
func TestDialogsWithoutAWindowStandAlone(t *testing.T) {
	originalOpen := openDialog
	t.Cleanup(func() { openDialog = originalOpen })
	originalBox := messageBox
	t.Cleanup(func() { messageBox = originalBox })

	var parentSeen *mygo.Window
	var dialogs []mygo.MessageOptions
	openDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		parentSeen = opts.Parent
		return nil, errors.New("the picker is unavailable")
	}
	messageBox = func(opts mygo.MessageOptions) { dialogs = append(dialogs, opts) }

	backUpDatabase(nil, "/tmp/aurora.db", nil)
	require.Len(t, dialogs, 1)
	assert.Nil(t, parentSeen, "no window means the picker is not parented")
	assert.Nil(t, dialogs[0].Parent, "no window means a standalone dialog")
	assert.Nil(t, parent(nil), "a nil window is never a parent")
}

// TestBackupReportsAPickerFailure pins the distinction a bare `err != nil ||
// empty` test erased: Cancel never returns an error, so a picker that DID fail
// was silently treated as "the user changed their mind" and the backup never
// happened with nothing on screen to say so.
func TestBackupReportsAPickerFailure(t *testing.T) {
	originalOpen := openDialog
	t.Cleanup(func() { openDialog = originalOpen })

	var dialogs []mygo.MessageOptions
	originalBox := messageBox
	t.Cleanup(func() { messageBox = originalBox })
	messageBox = func(opts mygo.MessageOptions) { dialogs = append(dialogs, opts) }

	// A window with no database path must complain instead of doing nothing.
	backUpDatabase(nil, "", nil)
	require.Len(t, dialogs, 1)
	assert.Equal(t, mygo.MessageError, dialogs[0].Type)
	assert.Equal(t, "Backup failed", dialogs[0].Title)

	// A picker that fails must be reported as a failure.
	openDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		return nil, errors.New("the portal is not running")
	}
	backUpDatabase(nil, filepath.Join(t.TempDir(), "aurora.db"), nil)
	require.Len(t, dialogs, 2)
	assert.Equal(t, mygo.MessageError, dialogs[1].Type)
	assert.Contains(t, dialogs[1].Message, "portal", "the reason must reach the user")

	// Cancel stays silent: no error, no path selected.
	openDialog = func(mygo.OpenDialogOptions) ([]string, error) { return nil, nil }
	backUpDatabase(nil, filepath.Join(t.TempDir(), "aurora.db"), nil)
	assert.Len(t, dialogs, 2, "cancel must not raise a dialog")
}

// TestOpenedDocumentsAreRemembered pins the Open Recent integration: an
// import through the shell (a dropped file or a launch argument) records the
// file, so File → Open Recent offers it later. A recording failure must not
// turn a successful import into an error, and nothing is recorded before the
// app is ready (the call would block on the main thread).
func TestOpenedDocumentsAreRemembered(t *testing.T) {
	originalAdd := addRecentDocument
	originalReady := appReady
	t.Cleanup(func() { addRecentDocument = originalAdd; appReady = originalReady })

	var recorded []string
	addRecentDocument = func(path string) error {
		recorded = append(recorded, path)
		return errors.New("the system refused the entry")
	}

	// Before Run: MyGo's path lookup waits for the main thread, so skip it.
	appReady = func() bool { return false }
	rememberDocument("/tmp/early.json")
	assert.Empty(t, recorded, "nothing may touch MyGo before the app is ready")

	appReady = func() bool { return true }
	rememberDocument("/tmp/draws.json")
	require.Equal(t, []string{"/tmp/draws.json"}, recorded)

	// An empty path is not a document (a cancelled picker).
	rememberDocument("")
	assert.Len(t, recorded, 1)
}

// TestAboutNamesOnlyWhatTheBuildHas pins the About dialog's content against
// the UI the app actually ships. It used to end with "Web UI: (embedded in
// the app)" long after the HTML UI was deleted, so a user asking what this
// build contains was told about a surface that no longer exists. The window
// is native now, so the dialog reports the version, the build time and the
// database, and nothing about a UI location.
func TestAboutNamesOnlyWhatTheBuildHas(t *testing.T) {
	originalBox := messageBox
	t.Cleanup(func() { messageBox = originalBox })
	var got mygo.MessageOptions
	messageBox = func(opts mygo.MessageOptions) { got = opts }

	showAbout(About{Version: "v9.9.9", BuildTime: "2026-10-25T00:00:00Z", DBPath: "/data/aurora.db"}, nil)

	assert.Equal(t, "About Aurora", got.Title)
	assert.Contains(t, got.Message, "v9.9.9")
	assert.Contains(t, got.Detail, "2026-10-25T00:00:00Z")
	assert.Contains(t, got.Detail, "/data/aurora.db")
	for _, stale := range []string{"Web UI", "Web 界面", "embedded", "%s"} {
		assert.NotContains(t, got.Detail, stale,
			"the About detail must not mention a UI that does not exist")
	}
}
