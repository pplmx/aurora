package deskapp

import (
	"github.com/egoist/mygo"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
)

// importReport builds what the user is told after an import. It is the single
// source of that wording: File > Import, a dropped file and a command-line
// argument all import through ImportLotteryFile and all report through here,
// so the three paths cannot drift into telling the user different stories
// about the same file.
//
// The wording mirrors the CLI, including the 0-based indices a user needs to
// find the offending records in the file.
func importReport(result lotteryapp.ImportResult, path string) (title, message, detail string) {
	if len(result.Rejected) > 0 {
		return T("desktop.dialog.import_partial_title"),
			TFormat("desktop.dialog.import_partial_message", result.Imported, result.Total, len(result.Rejected)),
			TFormat("desktop.dialog.import_rejected_detail", result.Rejected, path)
	}
	return T("desktop.dialog.imported_title"), TFormat("desktop.dialog.imported_message", result.Imported), path
}

// importReportType is the dialog severity matching importReport: a partial
// import is a warning because the valid records WERE stored.
func importReportType(result lotteryapp.ImportResult) mygo.MessageType {
	if len(result.Rejected) > 0 {
		return mygo.MessageWarning
	}
	return mygo.MessageInfo
}
