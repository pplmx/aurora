package deskapp

import (
	"fmt"

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
		return "Import partially failed",
			fmt.Sprintf("Imported %d of %d lottery records (failed: %d).", result.Imported, result.Total, len(result.Rejected)),
			fmt.Sprintf("Rejected record indices (0-based): %v\n\n"+
				"A record is rejected when it fails validation, when its ID already exists "+
				"(re-importing an export is refused rather than overwriting the stored draw), "+
				"or when it cannot be written.\n\n%s", result.Rejected, path)
	}
	return "Lotteries imported", fmt.Sprintf("%d draws imported.", result.Imported), path
}

// importReportType is the dialog severity matching importReport: a partial
// import is a warning because the valid records WERE stored.
func importReportType(result lotteryapp.ImportResult) mygo.MessageType {
	if len(result.Rejected) > 0 {
		return mygo.MessageWarning
	}
	return mygo.MessageInfo
}
