package deskapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

func TestParseIntent_DeepLinks(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want Intent
	}{
		{name: "page in the host", arg: "aurora://lottery", want: Intent{Page: "lottery", Raw: "aurora://lottery"}},
		{name: "page and draw", arg: "aurora://lottery?id=abc123", want: Intent{Page: "lottery", LotteryID: "abc123", Raw: "aurora://lottery?id=abc123"}},
		{name: "bare scheme is the dashboard", arg: "aurora://", want: Intent{Raw: "aurora://"}},
		{name: "window URL spelling", arg: "aurora://localhost/token.html", want: Intent{Page: "token", Raw: "aurora://localhost/token.html"}},
		{
			name: "window URL with a draw",
			arg:  "aurora://localhost/lottery.html?id=deadbeef",
			want: Intent{Page: "lottery", LotteryID: "deadbeef", Raw: "aurora://localhost/lottery.html?id=deadbeef"},
		},
		{name: "explicit dashboard", arg: "aurora://index", want: Intent{Raw: "aurora://index"}},
		{name: "scheme is case-insensitive", arg: "AURORA://Oracle", want: Intent{Page: "oracle", Raw: "AURORA://Oracle"}},
		{name: "surrounding space is trimmed", arg: "  aurora://nft  ", want: Intent{Page: "nft", Raw: "aurora://nft"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseIntent(tt.arg)
			require.True(t, ok, "ParseIntent(%q) must be accepted", tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseIntent_RejectsUnknownTargets pins the boundary: a link may only
// name a page the embedded UI actually serves, or the window would render the
// gateway's 404 inside the app.
func TestParseIntent_RejectsUnknownTargets(t *testing.T) {
	for _, arg := range []string{
		"",
		"   ",
		"aurora://settings",
		"aurora://lottery/extra",
		"aurora://../etc/passwd",
		"https://example.com/lottery",
		"lottery.html",
		"/tmp/does-not-exist.json",
		"not-a-link",
	} {
		_, ok := ParseIntent(arg)
		assert.False(t, ok, "ParseIntent(%q) must be rejected", arg)
	}
}

func TestParseIntent_LotteryExportFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draws.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o644))

	got, ok := ParseIntent(path)
	require.True(t, ok)
	assert.Equal(t, Intent{File: path, Raw: path}, got)

	// A directory, or a non-.json file, is not something the shell imports.
	_, isFile := ParseIntent(t.TempDir())
	assert.False(t, isFile)
	other := filepath.Join(t.TempDir(), "notes.txt")
	require.NoError(t, os.WriteFile(other, []byte("x"), 0o644))
	_, ok = ParseIntent(other)
	assert.False(t, ok)
}

func TestIntentURL(t *testing.T) {
	assert.Equal(t, "aurora://localhost/", Intent{}.URL())
	assert.Equal(t, "aurora://localhost/voting.html", Intent{Page: "voting"}.URL())
	assert.Equal(t, "aurora://localhost/lottery.html?id=a%2Bb", Intent{Page: "lottery", LotteryID: "a+b"}.URL(),
		"the draw id must be query-escaped, not pasted in")
}

// TestShell_IgnoresRepeatedDelivery is the cold-start contract: MyGo delivers
// the launch argument again after the app is ready, so the same link must not
// reload the page and the same file must not be imported twice.
func TestShell_IgnoresRepeatedDelivery(t *testing.T) {
	shell := NewShell(About{})
	var reports []string
	shell.report = func(title, message, _ string) { reports = append(reports, title+": "+message) }
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	// A cold start: ParseFlags already honored the argument, and MyGo
	// re-delivers it once the app is ready. Only the second delivery is
	// filtered, and it must create no navigation at all.
	shell.Consume("aurora://lottery?id=abc")
	shell.OpenURL("aurora://lottery?id=abc")
	assert.Empty(t, navigated, "the argument ParseFlags already honored must not navigate again")
	assert.Empty(t, reports, "the argument ParseFlags already honored must not be re-applied")

	// Without Consume (e.g. a second instance's URL) it runs, but only once.
	shell.OpenURL("aurora://token")
	shell.OpenURL("aurora://token")
	require.Len(t, navigated, 1)
	assert.Equal(t, Intent{Page: "token", Raw: "aurora://token"}, navigated[0])
	assert.Empty(t, reports, "a valid link produces no error dialog")
}

// TestShell_FileIsImportedExactlyOnce covers the cold-start file path: the
// command line names an export and MyGo also delivers it through OnOpenFile,
// so the file must reach the importer once. Both deliveries go through
// Shell.take; this pins that whichever arrives first wins.
func TestShell_FileIsImportedExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	exportPath := filepath.Join(dir, "draws.json")

	document := `[{"id":"deep-link-import","block_height":1,"seed":"seed-value","participants":["A","B"],"winners":["A"],"winner_addresses":["addr"],"vrf_output":"0011223344556677889900aabbccddeeff00112233445566778899aabbccddee","vrf_proof":"ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100","verified":true}]`
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	// Start() opens this database before WhenReady, which is what makes the
	// import path valid: the file always exists by the time a dialog is shown.
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NoError(t, repo.Close())

	shell := NewShell(About{DBPath: dbPath})
	var reports []string
	shell.report = func(title, message, _ string) { reports = append(reports, title+" | "+message) }

	shell.OpenFile(exportPath)
	shell.OpenFile(exportPath) // MyGo's post-ready delivery

	require.Len(t, reports, 1, "the export must be imported once, not twice")
	assert.Contains(t, reports[0], "Lotteries imported")
	assert.Contains(t, reports[0], "1 draws imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	record, err := stored.GetByID("deep-link-import")
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, record.Winners)
}

func TestShell_ReportsBadLinks(t *testing.T) {
	shell := NewShell(About{})
	var got string
	shell.report = func(title, message, _ string) { got = title + " | " + message }

	shell.OpenURL("https://example.com")
	assert.Contains(t, got, "could not open that link")
}

// TestShell_HandleLaunchArgs covers the second-instance path, which MyGo
// reaches through App.OnSecondInstance with the raw command line. MyGo only
// converts arguments it recognises (declared schemes, associated file types)
// into OnOpenURL/OnOpenFile, and Aurora claims no .json association, so a
// forwarded export used to be dropped without a word.
func TestShell_HandleLaunchArgs(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	exportPath := filepath.Join(dir, "draws.json")
	document := `[{"id":"second-instance-import","block_height":2,"seed":"second-seed","participants":["A","B"],"winners":["B"],"winner_addresses":["addr-b"],"vrf_output":"0011223344556677889900aabbccddeeff00112233445566778899aabbccddee","vrf_proof":"ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100","verified":true}]`
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NoError(t, repo.Close())

	shell := NewShell(About{DBPath: dbPath})
	var reports []string
	shell.report = func(title, message, _ string) { reports = append(reports, title+" | "+message) }
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	// What a second `aurora-desktop --data-dir X aurora://voting draws.json`
	// forwards. Flags are not intents and must be skipped, not reported.
	shell.HandleLaunchArgs([]string{"--data-dir", "/tmp/x", "aurora://voting", exportPath})

	require.Len(t, navigated, 1, "the forwarded link must reach the window")
	assert.Equal(t, Intent{Page: "voting", Raw: "aurora://voting"}, navigated[0])
	require.Len(t, reports, 1, "the forwarded export must be imported once")
	assert.Contains(t, reports[0], "Lotteries imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	_, err = stored.GetByID("second-instance-import")
	assert.NoError(t, err, "the export named by the second launch must be stored")

	// A malformed link of our own scheme is still ours to report.
	shell.HandleLaunchArgs([]string{"aurora://settings"})
	require.Len(t, reports, 2)
	assert.Contains(t, reports[1], "could not open that link")

	// A second launch with no arguments just raises the window; without one
	// attached that must be a no-op rather than a panic.
	require.NotPanics(t, func() { shell.HandleLaunchArgs(nil) })
}

// TestShell_ImportDropped pins what a drop does with each kind of file.
func TestShell_ImportDropped(t *testing.T) {
	shell := NewShell(About{DBPath: filepath.Join(t.TempDir(), "aurora.db")})
	var reports []string
	shell.report = func(title, message, _ string) { reports = append(reports, title+" | "+message) }

	other := filepath.Join(t.TempDir(), "report.csv")
	require.NoError(t, os.WriteFile(other, []byte("a,b"), 0o644))
	shell.ImportDropped([]string{other})
	require.Len(t, reports, 1)
	assert.Contains(t, reports[0], "Unsupported file")
	assert.Contains(t, reports[0], "report.csv")
}

func TestAboutCarriesTheShellIdentity(t *testing.T) {
	menu := NewMenu(About{Version: "v1", BuildTime: "t", DBPath: "/d"})
	require.NotNil(t, menu)

	// The window title and app name are what MyGo uses for App.Path and the
	// single-instance key; pin the constant the shell passes to SetName.
	assert.Equal(t, "Aurora", AppName)
}

func TestURLAndDeepLinkShareTheScheme(t *testing.T) {
	assert.Equal(t, "aurora", string(Scheme))
	assert.Equal(t, Scheme, DeepLinkScheme)
	assert.True(t, IsDeepLink(URL()), "the window URL must count as a deep link for OnOpenURL")
}

// TestShellAttachIsSafeWithoutAWindow keeps the shell usable from tests and
// before WhenReady: navigating early must not panic.
func TestShellAttachIsSafeWithoutAWindow(t *testing.T) {
	shell := NewShell(About{})
	require.NotPanics(t, func() {
		shell.Navigate(Intent{Page: "lottery"})
		shell.Attach(nil)
		shell.Navigate(Intent{Page: "lottery"})
	})
	var _ *mygo.Window = nil // the attach signature takes a MyGo window
}
