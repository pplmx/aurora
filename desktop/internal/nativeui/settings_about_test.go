package nativeui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAboutRowsCarryTheRuntimeOnce: the About card and the copied diagnostics
// draw from one list, so the card shows the toolchain line the copy reports
// instead of the two drifting apart.
func TestAboutRowsCarryTheRuntimeOnce(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{
		Version:      "v1.2.3",
		BuildTime:    "2026-10-26T09:00:00Z",
		Runtime:      "go1.27.3 · mygo v0.7.3",
		DBPath:       "/tmp/aurora/aurora.db",
		DatabaseSize: func() string { return "1.2 MiB" },
	})

	rows := app.aboutRows()
	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, row.label)
	}
	require.Equal(t, []string{"Version", "Build time", "Database", "Runtime", "Database size"},
		labels, "the card shows the same facts, in the same order, as the copy")
	require.Equal(t, "go1.27.3 · mygo v0.7.3", rows[3].value)
}
