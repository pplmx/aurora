package deskapp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
)

// TestImportReadErrorIsLocalized: an oversized export is a user mistake, so the
// dialog names the limit in the UI's language, and the size text comes from the
// same constant the reader enforces. A read failure the user cannot fix (a
// permission error) stays as the operating system reported it.
func TestImportReadErrorIsLocalized(t *testing.T) {
	withLocale(t, "en", func() {
		err := localizeImportReadError(fmt.Errorf("%w: max %d bytes",
			lotteryapp.ErrImportFileTooLarge, lotteryapp.MaxImportFileSize))
		require.Error(t, err)
		assert.Equal(t, "Import file is larger than 64.0 MiB; choose an export Aurora wrote.", err.Error())

		other := errors.New("open /tmp/x.json: permission denied")
		assert.Equal(t, other, localizeImportReadError(other))
	})

	withLocale(t, "zh", func() {
		err := localizeImportReadError(fmt.Errorf("%w: max %d bytes",
			lotteryapp.ErrImportFileTooLarge, lotteryapp.MaxImportFileSize))
		assert.Equal(t, "导入文件超过 64.0 MiB，请选择 Aurora 导出的文件。", err.Error())
	})

	withLocale(t, "zh-Hant", func() {
		err := localizeImportReadError(fmt.Errorf("%w: max %d bytes",
			lotteryapp.ErrImportFileTooLarge, lotteryapp.MaxImportFileSize))
		assert.Equal(t, "導入文件超過 64.0 MiB，請選擇 Aurora 導出的文件。", err.Error())
	})
}
