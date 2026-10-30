package deskapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseSizeTextCountsSidecars: the number a reader sees must match the
// disk usage `du` reports, so committed pages living in the -wal file count
// too; a missing sidecar or database adds nothing.
func TestDatabaseSizeTextCountsSidecars(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	require.NoError(t, os.WriteFile(dbPath, make([]byte, 100), 0o644))
	require.NoError(t, os.WriteFile(dbPath+"-wal", make([]byte, 50), 0o644))

	assert.Equal(t, "150 B", DatabaseSizeText(dbPath))
	assert.Equal(t, "", DatabaseSizeText(""))

	require.NoError(t, os.Remove(dbPath+"-wal"))
	assert.Equal(t, "100 B", DatabaseSizeText(dbPath))

	assert.Equal(t, "0 B", DatabaseSizeText(filepath.Join(dir, "missing.db")))
}

// TestAboutDialogNamesTheDatabaseSize: the native About dialog now reports the
// same disk usage as the Settings readout, and the format string takes a third
// argument; a wrong placeholder count would surface as "%!s(MISSING)" in the
// dialog.
func TestAboutDialogNamesTheDatabaseSize(t *testing.T) {
	withLocale(t, "en", func() {
		detail := TFormat("desktop.dialog.about_detail",
			"2026-01-01T00:00:00Z", "/tmp/aurora.db", "332.0 KiB")
		if !strings.Contains(detail, "Size: 332.0 KiB") {
			t.Fatalf("about detail must name the size, got %q", detail)
		}
	})
	withLocale(t, "zh", func() {
		detail := TFormat("desktop.dialog.about_detail",
			"2026-01-01T00:00:00Z", "/tmp/aurora.db", "332.0 KiB")
		if !strings.Contains(detail, "占用：332.0 KiB") {
			t.Fatalf("about detail must name the size in Chinese, got %q", detail)
		}
	})
}
