package deskapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/backup"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// About is what the About dialog and the backup action report.
type About struct {
	Version   string
	BuildTime string
	WebRoot   string
	DBPath    string
}

// NewMenu builds the application menu. It is safe to call before App.Run;
// the click handlers run on the main thread once the app is ready.
func NewMenu(meta About) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{
				ID:          "export-lotteries",
				Label:       "Export Lotteries...",
				Accelerator: "CmdOrCtrl+E",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					exportLotteries(meta)
				},
			},
			{
				ID:          "import-lotteries",
				Label:       "Import Lotteries...",
				Accelerator: "CmdOrCtrl+I",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					importLotteries(meta)
				},
			},
			mygo.Separator(),
			{Label: "Back Up Database...", Accelerator: "CmdOrCtrl+B", Click: func(*mygo.MenuItem, *mygo.Window) {
				backUpDatabase(meta)
			}},
			{Label: "Show Data Folder", Click: func(*mygo.MenuItem, *mygo.Window) {
				mygo.Shell.ShowItemInFolder(meta.DBPath)
			}},
			mygo.Separator(),
			{Role: mygo.RoleClose},
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
		{Label: "Help", Submenu: []*mygo.MenuItem{
			{Label: "About Aurora", Click: func(*mygo.MenuItem, *mygo.Window) {
				showAbout(meta)
			}},
		}},
	})
}

// lotteryJSONFilter is the file-dialog filter for the portable document
// `aurora lottery export` writes and `aurora lottery import` reads.
var lotteryJSONFilter = []mygo.FileFilter{{Name: "Lottery export", Extensions: []string{"json"}}}

// The file dialogs are variables so a test can drive a failed picker without
// a live window. A bare error return is easy to drop on the floor, which is
// exactly what made a failed folder picker look like Cancel.
var (
	openDialog = mygo.Dialog.Open
	saveDialog = mygo.Dialog.Save
)

// exportLotteries writes the portable document through the same app-layer
// serializer the CLI uses, so a desktop export and `aurora lottery export`
// produce byte-identical files for the same database.
func exportLotteries(meta About) {
	path, err := saveDialog(mygo.SaveDialogOptions{
		Title:       "Export Lotteries",
		DefaultPath: "aurora-lotteries.json",
		ButtonLabel: "Export",
		Filters:     lotteryJSONFilter,
	})
	if err != nil {
		showError("Export failed", err)
		return
	}
	if path == "" {
		return // canceled
	}

	repo, closeRepo, err := openLotteryRepo(meta)
	if err != nil {
		showError("Export failed", err)
		return
	}
	defer closeRepo()

	document, count, err := lotteryapp.ExportRecords(repo)
	if err != nil {
		showError("Export failed", err)
		return
	}
	if err := os.WriteFile(path, document, 0o644); err != nil {
		showError("Export failed", err)
		return
	}

	showMessage(mygo.MessageInfo, "Lotteries exported",
		fmt.Sprintf("%d draws written.", count), path)
}

// importLotteries stores a portable document through the same app-layer
// importer the CLI uses: identical validation, identical overwrite guard
// (TASK-069), identical partial-failure accounting. A partial import reports a
// warning — the valid records ARE stored — matching the CLI's "⚠" message.
func importLotteries(meta About) {
	paths, err := openDialog(mygo.OpenDialogOptions{
		Title:   "Import Lotteries",
		Filters: lotteryJSONFilter,
	})
	if err != nil {
		showError("Import failed", err)
		return
	}
	if len(paths) == 0 || paths[0] == "" {
		return // canceled
	}

	// The same call a dropped file and a command-line argument go through, so
	// all three entry points report identically (see Shell.OpenFile).
	result, err := ImportLotteryFile(meta, paths[0])
	if err != nil {
		showError("Import failed", err)
		return
	}

	title, message, detail := importReport(result, paths[0])
	showMessage(importReportType(result), title, message, detail)
}

// openLotteryRepo opens the same database the shell's gateway uses. The
// repository is opened per action and closed immediately: a long-lived second
// handle would hold a WAL reader open against the shell's own writer for no
// benefit, and each action is a one-shot file operation.
func openLotteryRepo(meta About) (lottery.Repository, func(), error) {
	if meta.DBPath == "" {
		return nil, nil, fmt.Errorf("no database path")
	}
	if _, err := os.Stat(meta.DBPath); err != nil {
		return nil, nil, fmt.Errorf("database %s: %w", filepath.Base(meta.DBPath), err)
	}
	repo, err := sqlite.NewLotteryRepository(meta.DBPath)
	if err != nil {
		return nil, nil, err
	}
	return repo, func() { _ = repo.Close() }, nil
}

// showError reports a failure in a native dialog: the shell may be launched
// from a file manager, where stderr goes nowhere.
func showError(title string, err error) {
	showMessage(mygo.MessageError, title, err.Error(), "")
}

// backUpDatabase snapshots the live SQLite database through the same
// BackupService the CLI uses (WAL-complete, checksummed metadata).
func backUpDatabase(meta About) {
	if meta.DBPath == "" {
		showError("Backup failed", errors.New("this window has no database path"))
		return
	}
	dirs, err := openDialog(mygo.OpenDialogOptions{
		Title:     "Choose where to store the backup",
		Directory: true,
	})
	if err != nil {
		// Cancel is "no error, no directory"; a picker that failed is a
		// failure, and silently returning made it look like the user had
		// cancelled.
		showError("Backup failed", err)
		return
	}
	if len(dirs) == 0 || dirs[0] == "" {
		return
	}
	dest := filepath.Join(dirs[0], "aurora-backup-"+BackupStamp(time.Now()))

	svc := backup.NewBackupService(map[string]string{"aurora": meta.DBPath})
	res, err := svc.Create(context.Background(), dest)
	if err != nil {
		showMessage(mygo.MessageError, "Backup failed",
			"Aurora could not back up the database.", err.Error())
		return
	}
	showMessage(mygo.MessageInfo, "Backup complete", "The database backup was written.", res.File)
}

// showAbout reports what this build is running against.
func showAbout(meta About) {
	ui := meta.WebRoot
	if ui == "" {
		ui = "(embedded in the app)"
	}
	showMessage(mygo.MessageInfo, "About Aurora", "Aurora Desktop "+meta.Version,
		"Built: "+meta.BuildTime+
			"\nDatabase: "+meta.DBPath+
			"\nWeb UI: "+ui)
}
