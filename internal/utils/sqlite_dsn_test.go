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
