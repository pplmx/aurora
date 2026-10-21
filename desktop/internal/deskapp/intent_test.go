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

	// The echo is consumed, not remembered forever: clicking the same link
	// again later is a new request and must work. A permanent set swallowed
	// every repeat for the life of the process.
	shell.OpenURL("aurora://lottery?id=abc")
	require.Len(t, navigated, 1)
	assert.Equal(t, Intent{Page: "lottery", LotteryID: "abc", Raw: "aurora://lottery?id=abc"}, navigated[0])

	// Without Consume (e.g. a second instance's URL) it runs every time.
	shell.OpenURL("aurora://token")
	shell.OpenURL("aurora://token")
	require.Len(t, navigated, 3)
	assert.Equal(t, Intent{Page: "token", Raw: "aurora://token"}, navigated[1])
	assert.Equal(t, Intent{Page: "token", Raw: "aurora://token"}, navigated[2])
	assert.Empty(t, reports, "a valid link produces no error dialog")

	// Nothing is left behind in the echo bookkeeping either: a map that only
	// ever grows would leak one entry per link for the life of the app.
	shell.mu.Lock()
	defer shell.mu.Unlock()
	assert.Empty(t, shell.expectEcho, "the echo bookkeeping must not grow without bound")
}

// TestShell_ForwardedLaunchEchoIsAbsorbed is the second-instance counterpart:
// MyGo runs its own deliverArgs immediately after OnSecondInstance, so the
// link the shell just acted on arrives once more. That one echo must be a
// no-op, and the link must still work the next time the user opens it.
func TestShell_ForwardedLaunchEchoIsAbsorbed(t *testing.T) {
	shell := NewShell(About{})
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	shell.HandleLaunchArgs([]string{"aurora://token"}, "")
	require.Len(t, navigated, 1, "the forwarded link navigates once")

	// MyGo's echo of that same argument.
	shell.OpenURL("aurora://token")
	require.Len(t, navigated, 1, "MyGo's echo must not reload the page")

	// A later, genuine second launch of the same link must still work.
	shell.HandleLaunchArgs([]string{"aurora://token"}, "")
	require.Len(t, navigated, 2)
}

// TestShell_FileIsImportedOnceFromTheCommandLine is the cold-start file path:
// the argument is on our command line, and because Aurora declares no file
// association MyGo never echoes it through OnOpenFile - so the shell must act
// on it exactly once itself, with no dedupe needed.
//
// The earlier version of this test pinned a permanent "already imported" set
// instead, which silently swallowed the user's next import of the same file
// and grew one entry per path for the life of the app.
func TestShell_FileIsImportedOnceFromTheCommandLine(t *testing.T) {
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

	shell.HandleLaunchArgs([]string{exportPath}, dir)

	require.Len(t, reports, 1, "the export must be imported once, not twice")
	assert.Contains(t, reports[0], "Lotteries imported")
	assert.Contains(t, reports[0], "1 draws imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	record, err := stored.GetByID("deep-link-import")
	require.NoError(t, err)
	assert.Equal(t, []string{"A"}, record.Winners)

	// Opening the same export again is a new request. The records are already
	// stored, so the app-layer overwrite guard refuses them - as a reported
	// failure, not a silent no-op at the shell level.
	shell.HandleLaunchArgs([]string{exportPath}, dir)
	require.Len(t, reports, 2, "a second import attempt must be reported, not dropped")
	assert.Contains(t, reports[1], "Import partially failed")
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
	shell.HandleLaunchArgs([]string{"--data-dir", "/tmp/x", "aurora://voting", exportPath}, dir)

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
	shell.HandleLaunchArgs([]string{"aurora://settings"}, dir)
	require.Len(t, reports, 2)
	assert.Contains(t, reports[1], "could not open that link")

	// A second launch with no arguments just raises the window; without one
	// attached that must be a no-op rather than a panic.
	require.NotPanics(t, func() { shell.HandleLaunchArgs(nil, "") })
}

// TestShell_HandleLaunchArgsResolvesTheLaunchersDirectory pins the relative
// path rule. MyGo forwards the launching process's working directory with the
// arguments, and its own file delivery joins relative paths onto it; a shell
// that parsed the arguments but dropped workDir looked for `draws.json` in the
// FIRST instance's directory and never found the file - a silent no-op that
// looks exactly like "the forward did nothing".
func TestShell_HandleLaunchArgsResolvesTheLaunchersDirectory(t *testing.T) {
	launcherDir := t.TempDir()
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "aurora.db")

	document := `[{"id":"relative-path-import","block_height":3,"seed":"relative-seed","participants":["A","B"],"winners":["A"],"winner_addresses":["addr-a"],"vrf_output":"0011223344556677889900aabbccddeeff00112233445566778899aabbccddee","vrf_proof":"ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100","verified":true}]`
	require.NoError(t, os.WriteFile(filepath.Join(launcherDir, "draws.json"), []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NoError(t, repo.Close())

	shell := NewShell(About{DBPath: dbPath})
	var reports []string
	shell.report = func(title, message, _ string) { reports = append(reports, title+" | "+message) }

	// The first instance runs elsewhere; only workDir can find the file.
	shell.HandleLaunchArgs([]string{"draws.json"}, launcherDir)

	require.Len(t, reports, 1, "the relative export must be found and imported")
	assert.Contains(t, reports[0], "Lotteries imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	_, err = stored.GetByID("relative-path-import")
	assert.NoError(t, err, "the draw from the launcher's directory must be stored")

	// The same file via a file:// URL, which Linux launchers also send.
	other := `[{"id":"file-url-import","block_height":4,"seed":"file-url-seed","participants":["C","D"],"winners":["D"],"winner_addresses":["addr-d"],"vrf_output":"0011223344556677889900aabbccddeeff00112233445566778899aabbccddee","vrf_proof":"ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100","verified":true}]`
	urlPath := filepath.Join(launcherDir, "from-url.json")
	require.NoError(t, os.WriteFile(urlPath, []byte(other), 0o644))

	shell.HandleLaunchArgs([]string{"file://" + filepath.ToSlash(urlPath)}, launcherDir)
	require.Len(t, reports, 2)
	assert.Contains(t, reports[1], "Lotteries imported")

	_, err = stored.GetByID("file-url-import")
	assert.NoError(t, err, "a file:// argument must import like a plain path")
}

// TestParseIntentIn_RejectsWhatIsNotThere keeps the resolver from turning a
// stray word into an import: only a regular .json file, absolute or under the
// launcher's directory, is an intent.
func TestParseIntentIn_RejectsWhatIsNotThere(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644))

	for _, arg := range []string{"", "   ", "notes.txt", "--data-dir", "aurora://settings", "/no/such/draws.json"} {
		_, ok := ParseIntentIn(arg, dir)
		assert.False(t, ok, "ParseIntentIn(%q) must be rejected", arg)
	}

	// An absolute path that does exist still works, and keeps its absolute
	// Raw so the take-once bookkeeping matches the argument that was sent.
	abs := filepath.Join(dir, "draws.json")
	require.NoError(t, os.WriteFile(abs, []byte("[]"), 0o644))
	in, ok := ParseIntentIn(abs, dir)
	require.True(t, ok)
	assert.Equal(t, Intent{File: abs, Raw: abs}, in)

	// A relative one records the path as sent plus where it resolved to.
	in, ok = ParseIntentIn("draws.json", dir)
	require.True(t, ok)
	assert.Equal(t, Intent{File: abs, Raw: "draws.json"}, in)

	// Without a working directory a relative path is not resolvable; an
	// absolute one is.
	_, ok = ParseIntentIn("draws.json", "")
	assert.False(t, ok)
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
