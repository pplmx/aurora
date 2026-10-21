package deskapp

import (
	"context"
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

// exportLotteries writes the portable document through the same app-layer
// serializer the CLI uses, so a desktop export and `aurora lottery export`
// produce byte-identical files for the same database.
func exportLotteries(meta About) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
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

	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageInfo,
		Title:   "Lotteries exported",
		Message: fmt.Sprintf("%d draws written.", count),
		Detail:  path,
	})
}

// importLotteries stores a portable document through the same app-layer
// importer the CLI uses: identical validation, identical overwrite guard
// (TASK-069), identical partial-failure accounting. A partial import reports a
// warning — the valid records ARE stored — matching the CLI's "⚠" message.
func importLotteries(meta About) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
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

	document, err := os.ReadFile(paths[0])
	if err != nil {
		showError("Import failed", err)
		return
	}

	repo, closeRepo, err := openLotteryRepo(meta)
	if err != nil {
		showError("Import failed", err)
		return
	}
	defer closeRepo()

	result, err := lotteryapp.ImportRecords(repo, document)
	if err != nil {
		showError("Import failed", err)
		return
	}

	// Report every rejection with the same wording as the CLI, including the
	// 0-based indices a user needs to find the offending records in the file.
	if len(result.Rejected) > 0 {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageWarning,
			Title:   "Import partially failed",
			Message: fmt.Sprintf("Imported %d of %d lottery records (failed: %d).", result.Imported, result.Total, len(result.Rejected)),
			Detail: fmt.Sprintf("Rejected record indices (0-based): %v\n\n"+
				"A record is rejected when it fails validation, when its ID already exists "+
				"(re-importing an export is refused rather than overwriting the stored draw), "+
				"or when it cannot be written.", result.Rejected),
		})
		return
	}

	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageInfo,
		Title:   "Lotteries imported",
		Message: fmt.Sprintf("%d draws imported.", result.Imported),
		Detail:  paths[0],
	})
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
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   title,
		Message: err.Error(),
	})
}

// backUpDatabase snapshots the live SQLite database through the same
// BackupService the CLI uses (WAL-complete, checksummed metadata).
func backUpDatabase(meta About) {
	if meta.DBPath == "" {
		return
	}
	dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:     "Choose where to store the backup",
		Directory: true,
	})
	if err != nil || len(dirs) == 0 || dirs[0] == "" {
		return
	}
	dest := filepath.Join(dirs[0], "aurora-backup-"+BackupStamp(time.Now()))

	svc := backup.NewBackupService(map[string]string{"aurora": meta.DBPath})
	res, err := svc.Create(context.Background(), dest)
	if err != nil {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageError,
			Title:   "Backup failed",
			Message: "Aurora could not back up the database.",
			Detail:  err.Error(),
		})
		return
	}
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageInfo,
		Title:   "Backup complete",
		Message: "The database backup was written.",
		Detail:  res.File,
	})
}

// showAbout reports what this build is running against.
func showAbout(meta About) {
	ui := meta.WebRoot
	if ui == "" {
		ui = "(embedded in the app)"
	}
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageInfo,
		Title:   "About Aurora",
		Message: "Aurora Desktop " + meta.Version,
		Detail: "Built: " + meta.BuildTime +
			"\nDatabase: " + meta.DBPath +
			"\nWeb UI: " + ui,
	})
}
