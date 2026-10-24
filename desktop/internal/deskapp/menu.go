package deskapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
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
//
// The labels are Aurora's own and are taken in the language the app shows, so
// the menu has to be rebuilt (SetMenu again) when that language changes; the
// items that carry a role are relabelled by MyGo itself. See FollowAppLocale.
func NewMenu(meta About) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: T("desktop.menu.file"), Submenu: []*mygo.MenuItem{
			{
				ID:          "export-lotteries",
				Label:       T("desktop.menu.export"),
				Accelerator: "CmdOrCtrl+E",
				Click: func(_ *mygo.MenuItem, win *mygo.Window) {
					exportLotteries(meta, win)
				},
			},
			{
				ID:          "import-lotteries",
				Label:       T("desktop.menu.import"),
				Accelerator: "CmdOrCtrl+I",
				Click: func(_ *mygo.MenuItem, win *mygo.Window) {
					importLotteries(meta, win)
				},
			},
			mygo.Separator(),
			{Label: T("desktop.menu.backup"), Accelerator: "CmdOrCtrl+B", Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				backUpDatabase(meta, win)
			}},
			{Label: T("desktop.menu.show_data_folder"), Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				mygo.Shell.ShowItemInFolder(meta.DBPath)
			}},
			mygo.Separator(),
			// MyGo keeps the list (recent-documents.json in PathUserData),
			// records each file with the system's own recent list, and opens a
			// chosen one through OnOpenFile - which is the import path above.
			{Role: mygo.RoleOpenRecent},
			mygo.Separator(),
			{Role: mygo.RoleClose},
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
		{Label: T("desktop.menu.help"), Submenu: []*mygo.MenuItem{
			{Label: T("desktop.menu.about"), Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				showAbout(meta, win)
			}},
			mygo.Separator(),
			{Label: T("desktop.menu.language"), Submenu: languageMenu()},
		}},
	})
}

// The language items' IDs, so the chosen one can be ticked without asking
// MyGo for the locale while the menu is being built (before Run, App.Locale
// waits for the main thread, which is not running yet).
const (
	languageSystemID      = "language-system"
	languageEnglishID     = "language-en"
	languageSimplifiedID  = "language-zh-hans"
	languageTraditionalID = "language-zh-hant"
)

// languageMenu lets the user choose the language at run time. MyGo keeps the
// choice for the next launch (System Settings on macOS, a file in
// PathUserData elsewhere), and OnLocaleChanged tells FollowAppLocale, which
// relabels Aurora's own items and lets MyGo relabel the roles.
//
// "Follow System" is SetLocale(""), not a tag: MyGo then follows the user's
// languages again, which is the state a first launch starts in.
func languageMenu() []*mygo.MenuItem {
	item := func(id, label, tag string) *mygo.MenuItem {
		return &mygo.MenuItem{
			ID:    id,
			Label: label,
			Type:  mygo.MenuItemRadio,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				mygo.App.SetLocale(tag)
			},
		}
	}
	return []*mygo.MenuItem{
		item(languageSystemID, T("desktop.menu.language_system"), ""),
		mygo.Separator(),
		item(languageEnglishID, T("desktop.menu.language_en"), "en"),
		item(languageSimplifiedID, T("desktop.menu.language_zh"), "zh-Hans"),
		item(languageTraditionalID, T("desktop.menu.language_zh_hant"), "zh-Hant"),
	}
}

// markLanguage ticks the item of the language the app shows. It runs once the
// app is ready, on the menu in use, and after each change: a radio group
// normally ticks itself, but the rebuilt menu starts with none ticked, and a
// pseudo locale (en-XA, a test of translations) matches no item.
//
// "Follow System" is not ticked even when that is how the language was
// chosen: App.Locale reports the resolved language, not whether an override
// is set, so ticking it would be a guess. The language in effect is what the
// menu can state, and it does.
func markLanguage(menu *mygo.Menu) {
	locale := appLocale()
	for id, tag := range map[string]string{
		languageEnglishID:     "en",
		languageSimplifiedID:  "zh-Hans",
		languageTraditionalID: "zh-Hant",
	} {
		if item := menu.ItemByID(id); item != nil {
			item.SetChecked(locale == tag)
		}
	}
}

// MarkLanguage is markLanguage for the app: it runs after the menu is
// installed and after a language change, when App.Locale is available.
func MarkLanguage(menu *mygo.Menu) {
	if menu == nil {
		return
	}
	markLanguage(menu)
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

// exportLotteries writes the portable document through the same app-layer
// serializer the CLI uses, so a desktop export and `aurora lottery export`
// produce byte-identical files for the same database.
func exportLotteries(meta About, win *mygo.Window) {
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

	repo, closeRepo, err := openLotteryRepo(meta)
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
func importLotteries(meta About, win *mygo.Window) {
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
	result, err := ImportLotteryFile(meta, paths[0])
	if err != nil {
		showError(parent(win), T("desktop.error.import_failed"), err)
		return
	}
	rememberDocument(paths[0])

	title, message, detail := importReport(result, paths[0])
	showMessage(parent(win), importReportType(result), title, message, detail)
}

// openLotteryRepo opens the same database the shell's gateway uses. The
// repository is opened per action and closed immediately: a long-lived second
// handle would hold a WAL reader open against the shell's own writer for no
// benefit, and each action is a one-shot file operation.
func openLotteryRepo(meta About) (lottery.Repository, func(), error) {
	if meta.DBPath == "" {
		return nil, nil, fmt.Errorf("%s", T("desktop.error.no_database_path"))
	}
	if _, err := os.Stat(meta.DBPath); err != nil {
		return nil, nil, fmt.Errorf(T("desktop.error.database"), filepath.Base(meta.DBPath), err)
	}
	repo, err := sqlite.NewLotteryRepository(meta.DBPath)
	if err != nil {
		return nil, nil, err
	}
	return repo, func() { _ = repo.Close() }, nil
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

// showAbout reports what this build is running against.
func showAbout(meta About, win *mygo.Window) {
	ui := meta.WebRoot
	if ui == "" {
		ui = T("desktop.dialog.about_embedded")
	}
	showMessage(parent(win), mygo.MessageInfo, T("desktop.dialog.about_title"),
		TFormat("desktop.dialog.about_message", meta.Version),
		TFormat("desktop.dialog.about_detail", meta.BuildTime, meta.DBPath, ui))
}
