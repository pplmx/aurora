package deskapp

import (
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
	var got [3]string
	shell.report = func(title, message, detail string) { got = [3]string{title, message, detail} }

	result := lotteryapp.ImportResult{Total: 2, Imported: 1, Rejected: []int{0}}
	shell.reportImport("/tmp/draws.json", result)

	title, message, detail := importReport(result, "/tmp/draws.json")
	require.Equal(t, [3]string{title, message, detail}, got,
		"the dropped-file dialog must be built by the same function as the menu's")
}
