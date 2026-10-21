package deskapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
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

	showMessage(mygo.MessageError, "Import failed", "boom", "/tmp/draws.json")
	assert.Equal(t, mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   "Import failed",
		Message: "boom",
		Detail:  "/tmp/draws.json",
	}, got)
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
	backUpDatabase(About{})
	require.Len(t, dialogs, 1)
	assert.Equal(t, mygo.MessageError, dialogs[0].Type)
	assert.Equal(t, "Backup failed", dialogs[0].Title)

	// A picker that fails must be reported as a failure.
	openDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		return nil, errors.New("the portal is not running")
	}
	backUpDatabase(About{DBPath: filepath.Join(t.TempDir(), "aurora.db")})
	require.Len(t, dialogs, 2)
	assert.Equal(t, mygo.MessageError, dialogs[1].Type)
	assert.Contains(t, dialogs[1].Message, "portal", "the reason must reach the user")

	// Cancel stays silent: no error, no path selected.
	openDialog = func(mygo.OpenDialogOptions) ([]string, error) { return nil, nil }
	backUpDatabase(About{DBPath: filepath.Join(t.TempDir(), "aurora.db")})
	assert.Len(t, dialogs, 2, "cancel must not raise a dialog")
}
