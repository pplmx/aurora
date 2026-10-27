package deskapp

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/backup"
)

// About is what the About dialog and the backup action report.
type About struct {
	Version   string
	BuildTime string
	DBPath    string
}

// NewPlatformMenu builds the menu macOS owns: the application menu, a File
// menu with Close, Edit for clipboard roles, and Window. It carries no Aurora
// actions: backup, export/import and recent exports live in the window's File
// menu, while language, data, appearance and build facts live in Settings. The
// bar is therefore the platform's own chrome, which MyGo relabels when the
// locale changes.
func NewPlatformMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	})
}

// lotteryJSONFilter is the file-dialog filter for the portable document
// `aurora lottery export` writes and `aurora lottery import` reads.
func lotteryJSONFilter() []mygo.FileFilter {
	return []mygo.FileFilter{{Name: T("desktop.filter.lottery_export"), Extensions: []string{"json"}}}
}

// The file dialogs are variables so a test can drive a failed picker without
// a live window. A bare error return is easy to drop on the floor, which is
// exactly what made a failed folder picker look like Cancel.
var (
	openDialog = mygo.Dialog.Open
	saveDialog = mygo.Dialog.Save
	// addRecentDocument records a document the user opened or saved. MyGo
	// keeps the list for File → Open Recent and for the system's own recent
	// documents (the Dock menu, the jump list, the file manager); it is a
	// variable so a test does not write into the real user directory.
	addRecentDocument = mygo.App.AddRecentDocument
	// appReady gates that call. App.Path and the system's recent-documents
	// API need the running app: before Run they post to the main thread and
	// wait, which hangs a test (measured: the suite stopped in
	// TestShell_FileIsImportedOnceFromTheCommandLine) and would hang a
	// command-line import that ran too early.
	appReady = mygo.App.IsReady
)

// rememberDocument feeds File → Open Recent. A failure to record the file is
// not a reason to fail the import or export that succeeded: the document is
// already where the user wanted it, and the recent list is a convenience.
func rememberDocument(path string) {
	if path == "" || !appReady() {
		return
	}
	_ = addRecentDocument(path)
}

// parent is the window a dialog belongs to. MyGo attaches a dialog with a
// Parent to that window - a sheet on macOS, a modal child elsewhere - instead
// of letting it float free of the window the action was started from. Linux
// and Windows pass the window whose menu was chosen, macOS the focused one,
// both of which are nil when no window has focus.
func parent(win *mygo.Window) *mygo.Window {
	if win == nil || win.IsDestroyed() {
		return nil
	}
	return win
}

// lotteryRepoProvider resolves the repository for a one-shot file action. The
// shell supplies the container-owned repository.
type lotteryRepoProvider func() (lottery.Repository, func(), error)

// exportLotteriesWith writes the portable document through the same app-layer
// serializer the CLI uses, so a desktop export and `aurora lottery export`
// produce byte-identical files for the same database.
func exportLotteriesWith(repoProvider lotteryRepoProvider, win *mygo.Window) {
	path, err := saveDialog(mygo.SaveDialogOptions{
		Parent:      parent(win),
		Title:       T("desktop.dialog.export_title"),
		DefaultPath: "aurora-lotteries.json",
		ButtonLabel: T("desktop.dialog.export_button"),
		Filters:     lotteryJSONFilter(),
	})
	if err != nil {
		showError(parent(win), T("desktop.error.export_failed"), err)
		return
	}
	if path == "" {
		return // canceled
	}

	repo, closeRepo, err := repoProvider()
	if err != nil {
		showError(parent(win), T("desktop.error.export_failed"), err)
		return
	}
	defer closeRepo()

	count, err := lotteryapp.WriteRecordsToFile(repo, path)
	if err != nil {
		showError(parent(win), T("desktop.error.export_failed"), err)
		return
	}
	rememberDocument(path)

	showMessage(parent(win), mygo.MessageInfo, T("desktop.dialog.exported_title"),
		TFormat("desktop.dialog.exported_message", count), path)
}

// importLotteries stores a portable document through the same app-layer
// importer the CLI uses: identical validation, identical overwrite guard
// (TASK-069), identical partial-failure accounting. A partial import reports a
// warning — the valid records ARE stored — matching the CLI's "⚠" message.
func importLotteriesWith(repoProvider lotteryRepoProvider, win *mygo.Window, reload func()) {
	paths, err := openDialog(mygo.OpenDialogOptions{
		Parent:  parent(win),
		Title:   T("desktop.dialog.import_title"),
		Filters: lotteryJSONFilter(),
	})
	if err != nil {
		showError(parent(win), T("desktop.error.import_failed"), err)
		return
	}
	if len(paths) == 0 || paths[0] == "" {
		return // canceled
	}

	// The same call a dropped file and a command-line argument go through, so
	// all three entry points report identically (see Shell.OpenFile).
	result, err := importLotteryFile(repoProvider, paths[0])
	if err != nil {
		showError(parent(win), T("desktop.error.import_failed"), err)
		return
	}
	rememberDocument(paths[0])
	if result.Imported > 0 {
		// The imported draws belong in the page behind the dialog; a menu click
		// is outside the view's event loop, so reload and redraw explicitly.
		if reload != nil {
			reload()
		}
		if win != nil {
			win.Invalidate()
		}
	}

	title, message, detail := importReport(result, paths[0])
	showMessage(parent(win), importReportType(result), title, message, detail)
}

// showError reports a failure in a native dialog: the shell may be launched
// from a file manager, where stderr goes nowhere.
func showError(win *mygo.Window, title string, err error) {
	showMessage(parent(win), mygo.MessageError, title, err.Error(), "")
}

// backUpDatabase snapshots the live SQLite database through the same
// BackupService the CLI uses (WAL-complete, checksummed metadata).
func backUpDatabase(meta About, win *mygo.Window) {
	if meta.DBPath == "" {
		showError(parent(win), T("desktop.error.backup_failed"), errors.New(T("desktop.error.backup_no_db")))
		return
	}
	dirs, err := openDialog(mygo.OpenDialogOptions{
		Parent:    parent(win),
		Title:     T("desktop.dialog.backup_pick_dir"),
		Directory: true,
	})
	if err != nil {
		// Cancel is "no error, no directory"; a picker that failed is a
		// failure, and silently returning made it look like the user had
		// cancelled.
		showError(parent(win), T("desktop.error.backup_failed"), err)
		return
	}
	if len(dirs) == 0 || dirs[0] == "" {
		return
	}
	dest := filepath.Join(dirs[0], "aurora-backup-"+BackupStamp(time.Now()))

	svc := backup.NewBackupService(map[string]string{"aurora": meta.DBPath})
	res, err := svc.Create(context.Background(), dest)
	if err != nil {
		showMessage(parent(win), mygo.MessageError, T("desktop.error.backup_failed"),
			T("desktop.dialog.backup_could_not"), err.Error())
		return
	}
	showMessage(parent(win), mygo.MessageInfo, T("desktop.dialog.backup_done_title"),
		T("desktop.dialog.backup_done_message"), res.File)
}

// showAbout reports what this build is running against: the version, when it
// was built, and which database it has open. The window is native, so there is
// no UI location to name.
func showAbout(meta About, win *mygo.Window) {
	showMessage(parent(win), mygo.MessageInfo, T("desktop.dialog.about_title"),
		TFormat("desktop.dialog.about_message", meta.Version),
		TFormat("desktop.dialog.about_detail", meta.BuildTime, meta.DBPath))
}
