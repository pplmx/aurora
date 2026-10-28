package utils

import "testing"

func TestSQLiteDSNDropsFileURIFragmentBeforeAddingParameters(t *testing.T) {
	tests := []struct {
		name  string
		db    string
		query string
		want  string
	}{
		{
			name:  "fragment without existing query",
			db:    "file:/tmp/aurora.db#section",
			query: "_foreign_keys=ON",
			want:  "file:/tmp/aurora.db?_foreign_keys=ON",
		},
		{
			name:  "fragment with existing query",
			db:    "file:/tmp/aurora.db?mode=rwc#section",
			query: "_foreign_keys=ON",
			want:  "file:/tmp/aurora.db?mode=rwc&_foreign_keys=ON",
		},
		{
			name:  "question mark in fragment does not select ampersand",
			db:    "file:/tmp/aurora.db#section?mode=ro",
			query: "_foreign_keys=ON",
			want:  "file:/tmp/aurora.db?_foreign_keys=ON",
		},
		{
			name:  "encoded hash stays in the path",
			db:    "file:/tmp/aurora%23copy.db#section",
			query: "_foreign_keys=ON",
			want:  "file:/tmp/aurora%23copy.db?_foreign_keys=ON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SQLiteDSN(tt.db, tt.query); got != tt.want {
				t.Fatalf("SQLiteDSN(%q, %q) = %q, want %q", tt.db, tt.query, got, tt.want)
			}
		})
	}
}

func TestSQLiteFileURIWindowsPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "UNC path",
			path: `\\server\share\aurora.db`,
			want: "file:////server/share/aurora.db",
		},
		{
			name: "UNC path escapes filename characters",
			path: `\\server\share\state #1.db`,
			want: "file:////server/share/state%20%231.db",
		},
		{
			name: "drive path",
			path: `C:\data\aurora.db`,
			want: "file:/C:/data/aurora.db",
		},
		{
			name: "drive path with forward slashes",
			path: "C:/data/aurora.db",
			want: "file:/C:/data/aurora.db",
		},
		{
			name: "relative path",
			path: `data\aurora.db`,
			want: "file:data/aurora.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sqliteFileURIForOS(tt.path, "windows"); got != tt.want {
				t.Fatalf("sqliteFileURIForOS(%q, windows) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
