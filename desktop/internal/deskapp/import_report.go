package deskapp

import (
	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
)

// importReport builds what the user is told after an import. It is the single
// source of that wording: File > Import, a dropped file and a command-line
// argument all import through importLotteryFile and all report through here,
// so the three paths cannot drift into telling the user different stories
// about the same file.
//
// The wording mirrors the CLI, including the 0-based indices a user needs to
// find the offending records in the file.
func importReport(result lotteryapp.ImportResult, path string) (title, message, detail string) {
	if len(result.Rejected) > 0 {
		// "Partially failed" is only true when something was stored. Re-importing
		// an export whose records are all already stored rejects every one of them,
		// and the partial wording then read "Import partially failed. Imported 0 of
		// 3 lottery records" — a heading that contradicts its own sentence.
		if result.Imported == 0 {
			return T("desktop.dialog.import_rejected_title"),
				TFormat("desktop.dialog.import_rejected_message", result.Total),
				TFormat("desktop.dialog.import_rejected_detail", result.Rejected, path)
		}
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
