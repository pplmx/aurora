// Command coverfloor enforces the documented per-module coverage floor from a
// Go coverage profile.
//
//	CGO_ENABLED=0 go test -coverprofile=coverage.out ./internal/domain/... ./internal/app/...
//	go run ./tools/coverfloor -profile coverage.out -floor 80
//
// The profile's file paths are import paths, so a package's statements and
// covered statements are summed from every block it contains. Packages with no
// statements are skipped; a package under the selected prefixes that has no
// coverage at all fails, because silence there means the profile was empty.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	profile := flag.String("profile", "coverage.out", "coverage profile written by go test -coverprofile")
	floor := flag.Float64("floor", 80, "minimum package coverage percentage")
	prefixes := flag.String("prefixes", "github.com/pplmx/aurora/internal/domain/,github.com/pplmx/aurora/internal/app/",
		"comma-separated import path prefixes to check")
	excludes := flag.String("exclude", "",
		"comma-separated import path prefixes to skip")
	flag.Parse()

	data, err := os.ReadFile(*profile)
	if err != nil {
		fail(err)
	}
	packages, err := parseProfile(string(data))
	if err != nil {
		fail(err)
	}

	wanted := splitPrefixes(*prefixes)
	skipped := splitPrefixes(*excludes)
	checked := 0
	var failures []string
	rows := make([]row, 0, len(packages))
	for name, stats := range packages {
		if !matchesAny(name, wanted) || matchesAny(name, skipped) {
			continue
		}
		checked++
		percent := stats.percent()
		rows = append(rows, row{name: name, covered: stats.covered, total: stats.total, percent: percent})
		if percent < *floor {
			failures = append(failures, fmt.Sprintf("%s: %.1f%% (< %.0f%%)", name, percent, *floor))
		}
	}
	if checked == 0 {
		fail(fmt.Errorf("no package under %s appears in %s", strings.Join(wanted, ", "), *profile))
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	width := 0
	for _, r := range rows {
		if len(r.name) > width {
			width = len(r.name)
		}
	}
	for _, r := range rows {
		fmt.Printf("%-*s  %6d/%-6d  %5.1f%%\n", width, r.name, r.covered, r.total, r.percent)
	}
	if len(failures) > 0 {
		fail(fmt.Errorf("coverage floor %.0f%% missed:\n  %s", *floor, strings.Join(failures, "\n  ")))
	}
	fmt.Printf("floor %.0f%%: ok (%d packages)\n", *floor, checked)
}

type row struct {
	name    string
	covered int
	total   int
	percent float64
}

type stats struct {
	covered int
	total   int
}

// percent is statement coverage: a block counts as covered when it ran at
// least once, whatever its statement count.
func (s stats) percent() float64 {
	if s.total == 0 {
		return 100
	}
	return float64(s.covered) / float64(s.total) * 100
}

// parseProfile sums the statement blocks of a coverage profile per package.
// The header line ("mode: atomic") is ignored; a file line reads
// "import/path/file.go:start,end numStmt count".
func parseProfile(text string) (map[string]*stats, error) {
	packages := make(map[string]*stats)
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected profile line: %q", line)
		}
		file := fields[0]
		colon := strings.Index(file, ":")
		if colon < 0 {
			return nil, fmt.Errorf("unexpected profile path: %q", file)
		}
		file = file[:colon]
		slash := strings.LastIndex(file, "/")
		if slash < 0 {
			return nil, fmt.Errorf("unexpected profile path: %q", file)
		}
		name := file[:slash]

		statements, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("unexpected statement count in %q: %w", line, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("unexpected run count in %q: %w", line, err)
		}
		entry := packages[name]
		if entry == nil {
			entry = &stats{}
			packages[name] = entry
		}
		entry.total += statements
		if count > 0 {
			entry.covered += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return packages, nil
}

// splitPrefixes drops empty entries: an empty prefix would match every package,
// so `-exclude ""` must mean "exclude nothing" rather than "exclude all".
func splitPrefixes(list string) []string {
	parts := strings.Split(list, ",")
	out := parts[:0]
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func matchesAny(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "coverfloor:", err)
	os.Exit(1)
}
