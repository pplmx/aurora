package deskapp

import (
	"os"
	"path/filepath"
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
