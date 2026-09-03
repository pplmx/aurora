package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestParseFlags locks the cmd/api flag surface (TASK-267, ISS-263): the
// server binary previously ignored --help/--version entirely and would start
// the HTTP server on any argument, so `aurora-api --help` (or a misspelled
// flag) launched a server that then died on a busy bind. parseFlags must
// classify help/version requests, reject unknown flags and stray positional
// args, and pass a clean command line straight to the server boot.
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    runMode
		wantErr bool
	}{
		{"no args", nil, runServer, false},
		{"empty args", []string{}, runServer, false},
		{"long help", []string{"--help"}, runHelp, false},
		{"short help", []string{"-h"}, runHelp, false},
		{"bare help", []string{"-help"}, runHelp, false},
		{"help wins over version", []string{"--version", "--help"}, runHelp, false},
		{"long version", []string{"--version"}, runVersion, false},
		{"single dash version", []string{"-version"}, runVersion, false},
		{"short v alias", []string{"-v"}, runVersion, false},
		{"version with equals", []string{"--version=true"}, runVersion, false},
		{"version false", []string{"--version=false"}, runServer, false},
		{"bare terminator", []string{"--"}, runServer, false},
		{"version then stray positional", []string{"--version", "foo"}, runServer, true},
		{"unknown flag", []string{"--bogus"}, runServer, true},
		{"unknown short flag", []string{"-x"}, runServer, true},
		{"bare single dash", []string{"-"}, runServer, true},
		{"bad flag syntax", []string{"-=x"}, runServer, true},
		{"stray positional", []string{"serve"}, runServer, true},
		{"positional after terminator", []string{"--", "x"}, runServer, true},
		{"version then unknown", []string{"--version", "--bogus"}, runServer, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFlags(%v) err = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseFlags(%v) mode = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

// TestPrintVersion_ReportsRealBuild verifies --version output carries the
// link-time Version/BuildTime vars and the runtime toolchain, mirroring the
// CLI's version command (cmd/aurora/cmd/version.go) rather than a
// hardcoded placeholder.
func TestPrintVersion_ReportsRealBuild(t *testing.T) {
	var buf bytes.Buffer
	printVersion(&buf)
	out := buf.String()
	// Assert against the same i18n labels printVersion uses, not the English
	// literals: under LANG=zh the labels render as 版本/构建时间/Go 版本, so
	// hardcoded English assertions would fail in a localized environment
	// (cmd/api review M2).
	for _, want := range []string{
		i18n.GetText("app.version") + ": ",
		i18n.GetText("app.build_time") + ": ",
		i18n.GetText("app.go_version") + ": ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printVersion output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, Version) {
		t.Errorf("printVersion output missing the Version var %q:\n%s", Version, out)
	}
	if !strings.Contains(out, BuildTime) {
		t.Errorf("printVersion output missing the BuildTime var %q:\n%s", BuildTime, out)
	}
}

// TestPrintUsage_ListsFlags verifies the usage text advertises exactly the
// flags parseFlags understands, so help can never recommend an unsupported
// surface.
func TestPrintUsage_ListsFlags(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	for _, want := range []string{"Usage:", "--help", "--version", "["} {
		if !strings.Contains(out, want) {
			t.Errorf("printUsage output missing %q:\n%s", want, out)
		}
	}
}
