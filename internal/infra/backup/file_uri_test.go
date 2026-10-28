package backup

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupServiceSupportsDiskFileURI(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "state with spaces", "aurora.db")
	newDB(t, dbPath,
		"CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)",
		"INSERT INTO schema_migrations (version) VALUES (7)",
		"CREATE TABLE marker (value TEXT NOT NULL)",
		"INSERT INTO marker (value) VALUES ('before backup')",
	)
	dbURI := "file:" + url.PathEscape(filepath.ToSlash(dbPath))
	output := filepath.Join(root, "backup")

	svc := NewBackupService(map[string]string{"aurora": dbURI})
	result, err := svc.Create(context.Background(), output)
	require.NoError(t, err)
	require.Equal(t, uint(7), result.SchemaVersion)
	require.NoError(t, svc.Verify(context.Background(), output))

	source, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = source.Exec("UPDATE marker SET value = 'after backup'")
	require.NoError(t, err)
	require.NoError(t, source.Close())

	require.NoError(t, svc.Restore(context.Background(), output))

	source, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = source.Close() })
	var value string
	require.NoError(t, source.QueryRow("SELECT value FROM marker").Scan(&value))
	require.Equal(t, "before backup", value)
}

func TestBackupServiceRejectsMemoryBackedDatabase(t *testing.T) {
	output := filepath.Join(t.TempDir(), "backup")
	svc := NewBackupService(map[string]string{
		"aurora": "file:shared?mode=memory&cache=shared",
	})

	_, err := svc.Create(context.Background(), output)
	require.ErrorContains(t, err, "not disk-backed")
	require.ErrorContains(t, svc.Restore(context.Background(), output), "not disk-backed")
	require.NoDirExists(t, output)
}
