package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestExportDestinationErrorsLocalize: the window shares the export path with
// the CLI, and the two destination checks carry a path, so a full-message lookup
// cannot reach them; the sentence around the path is what translates.
func TestExportDestinationErrorsLocalize(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	require.Equal(t, "导出路径为空", localizeServiceErrorMessage("export path is empty"))
	require.Equal(t, "导出目录 /tmp/missing 不存在",
		localizeServiceErrorMessage("export directory /tmp/missing does not exist"))
	// Anything else keeps its words so a diagnostic still matches the log.
	require.Equal(t, "export directory /tmp/x does not exist (wrapped)",
		localizeServiceErrorMessage("export directory /tmp/x does not exist (wrapped)"))
}
