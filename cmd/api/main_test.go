package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/i18n"
)

// TestParseFlags locks the cmd/api flag surface (TASK-267, ISS-263, extended
// for --host/--port in TASK-273): the server binary previously ignored
// --help/--version entirely and would start the HTTP server on any argument,
// so `aurora-api --help` (or a misspelled flag) launched a server that then
// died on a busy bind. parseFlags must classify help/version requests, reject
// unknown flags and stray positional args, parse the host/port overrides
// (with aliases and range validation), and pass a clean command line straight
// to the server boot.
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		want        runMode
		wantErr     bool
		wantHost    string
		wantHostSet bool
		wantPort    int
		wantPortSet bool
	}{
		{"no args", nil, runServer, false, "", false, 0, false},
		{"empty args", []string{}, runServer, false, "", false, 0, false},
		{"long help", []string{"--help"}, runHelp, false, "", false, 0, false},
		{"short help", []string{"-h"}, runHelp, false, "", false, 0, false},
		{"bare help", []string{"-help"}, runHelp, false, "", false, 0, false},
		{"help wins over version", []string{"--version", "--help"}, runHelp, false, "", false, 0, false},
		{"help ignores bad port", []string{"--help", "--port", "99999"}, runHelp, false, "", false, 0, false},
		{"long version", []string{"--version"}, runVersion, false, "", false, 0, false},
		{"single dash version", []string{"-version"}, runVersion, false, "", false, 0, false},
		{"short v alias", []string{"-v"}, runVersion, false, "", false, 0, false},
		{"version with equals", []string{"--version=true"}, runVersion, false, "", false, 0, false},
		{"version false", []string{"--version=false"}, runServer, false, "", false, 0, false},
		{"version ignores bad port", []string{"--version", "--port", "99999"}, runVersion, false, "", false, 0, false},
		{"bare terminator", []string{"--"}, runServer, false, "", false, 0, false},
		{"host flag", []string{"--host", "127.0.0.1"}, runServer, false, "127.0.0.1", true, 0, false},
		{"host equals", []string{"--host=::1"}, runServer, false, "::1", true, 0, false},
		{"host short alias", []string{"-H", "127.0.0.1"}, runServer, false, "127.0.0.1", true, 0, false},
		{"port flag", []string{"--port", "9090"}, runServer, false, "", false, 9090, true},
		{"port equals", []string{"--port=9090"}, runServer, false, "", false, 9090, true},
		{"port short alias", []string{"-p", "9090"}, runServer, false, "", false, 9090, true},
		{"host and port", []string{"--host", "0.0.0.0", "--port", "5000"}, runServer, false, "0.0.0.0", true, 5000, true},
		{"port zero", []string{"--port", "0"}, runServer, true, "", false, 0, true},
		{"port high", []string{"--port", "70000"}, runServer, true, "", false, 70000, true},
		{"port negative", []string{"--port", "-1"}, runServer, true, "", false, -1, true},
		{"port non-numeric", []string{"--port", "abc"}, runServer, true, "", false, 0, false},
		{"version then stray positional", []string{"--version", "foo"}, runServer, true, "", false, 0, false},
		{"unknown flag", []string{"--bogus"}, runServer, true, "", false, 0, false},
		{"unknown short flag", []string{"-x"}, runServer, true, "", false, 0, false},
		{"bare single dash", []string{"-"}, runServer, true, "", false, 0, false},
		{"bad flag syntax", []string{"-=x"}, runServer, true, "", false, 0, false},
		{"stray positional", []string{"serve"}, runServer, true, "", false, 0, false},
		{"positional after terminator", []string{"--", "x"}, runServer, true, "", false, 0, false},
		{"version then unknown", []string{"--version", "--bogus"}, runServer, true, "", false, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, opts, err := parseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFlags(%v) err = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseFlags(%v) mode = %v, want %v", tt.args, got, tt.want)
			}
			if opts == nil {
				if tt.wantHostSet || tt.wantPortSet {
					t.Errorf("parseFlags(%v) opts = nil, want overrides", tt.args)
				}
				return
			}
			if opts.host != tt.wantHost || opts.hostSet != tt.wantHostSet {
				t.Errorf("parseFlags(%v) host = %q (set=%v), want %q (set=%v)",
					tt.args, opts.host, opts.hostSet, tt.wantHost, tt.wantHostSet)
			}
			if opts.port != tt.wantPort || opts.portSet != tt.wantPortSet {
				t.Errorf("parseFlags(%v) port = %d (set=%v), want %d (set=%v)",
					tt.args, opts.port, opts.portSet, tt.wantPort, tt.wantPortSet)
			}
		})
	}
}

// TestResolveAddr_Precedence pins the listen-address chain: an explicit
// --host/--port flag wins over the config values (env/default resolution
// happened inside config.Load), and an absent flag never overrides. The
// config's own defaults are asserted here with a hand-built Config so the
// helper is tested without touching viper.
func TestResolveAddr_Precedence(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Host: "0.0.0.0", Port: 8080}}

	tests := []struct {
		name string
		opts *serverOptions
		want string
	}{
		{"nil opts", nil, "0.0.0.0:8080"},
		{"empty opts", &serverOptions{}, "0.0.0.0:8080"},
		{"host only", &serverOptions{host: "127.0.0.1", hostSet: true}, "127.0.0.1:8080"},
		{"port only", &serverOptions{port: 9090, portSet: true}, "0.0.0.0:9090"},
		{"both", &serverOptions{host: "::1", hostSet: true, port: 9090, portSet: true}, "::1:9090"},
		{"set flag keeps config port", &serverOptions{host: "127.0.0.1", hostSet: true}, "127.0.0.1:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAddr(tt.opts, cfg); got != tt.want {
				t.Errorf("resolveAddr(%v, cfg) = %q, want %q", tt.opts, got, tt.want)
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
	for _, want := range []string{"Usage:", "--help", "--version", "--host", "--port", "["} {
		if !strings.Contains(out, want) {
			t.Errorf("printUsage output missing %q:\n%s", want, out)
		}
	}
}
