package deskapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntent_DeepLinks(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want Intent
	}{
		{name: "page in the host", arg: "aurora://lottery", want: Intent{Page: "lottery", Raw: "aurora://lottery"}},
		{name: "page and draw", arg: "aurora://lottery?id=abc123", want: Intent{Page: "lottery", LotteryID: "abc123", Raw: "aurora://lottery?id=abc123"}},
		{name: "bare scheme is the dashboard", arg: "aurora://", want: Intent{Page: "dashboard", Raw: "aurora://"}},
		{name: "window URL spelling", arg: "aurora://localhost/token.html", want: Intent{Page: "token", Raw: "aurora://localhost/token.html"}},
		{
			name: "window URL with a draw",
			arg:  "aurora://localhost/lottery.html?id=deadbeef",
			want: Intent{Page: "lottery", LotteryID: "deadbeef", Raw: "aurora://localhost/lottery.html?id=deadbeef"},
		},
		{name: "explicit dashboard", arg: "aurora://index", want: Intent{Page: "dashboard", Raw: "aurora://index"}},
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
	shell.report = func(_ mygo.MessageType, title, message, _ string) { reports = append(reports, title+": "+message) }
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

	draw := genuineDesktopLotteryRecord(t, "seed-value")
	draw.BlockHeight = 1
	document, err := json.Marshal([]lottery.LotteryRecord{draw})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	// Start() opens this database before WhenReady, which is what makes the
	// import path valid: the file always exists by the time a dialog is shown.
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) { reports = append(reports, title+" | "+message) }

	shell.HandleLaunchArgs([]string{exportPath}, dir)

	require.Len(t, reports, 1, "the export must be imported once, not twice")
	assert.Contains(t, reports[0], "Lotteries imported")
	assert.Contains(t, reports[0], "1 draws imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	record, err := stored.GetByID(draw.ID)
	require.NoError(t, err)
	assert.Equal(t, draw.Winners, record.Winners)

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
	shell.report = func(_ mygo.MessageType, title, message, _ string) { got = title + " | " + message }

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
	draw := genuineDesktopLotteryRecord(t, "second-seed")
	draw.BlockHeight = 2
	document, err := json.Marshal([]lottery.LotteryRecord{draw})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) { reports = append(reports, title+" | "+message) }
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	// What a second `aurora-desktop --data-dir X aurora://voting draws.json`
	// forwards. Flags are not intents; the data directory names the one this
	// instance already uses, so it is not a conflict either.
	shell.HandleLaunchArgs([]string{"--data-dir", dir, "aurora://voting", exportPath}, dir)

	require.Len(t, navigated, 1, "the forwarded link must reach the window")
	assert.Equal(t, Intent{Page: "voting", Raw: "aurora://voting"}, navigated[0])
	require.Len(t, reports, 1, "the forwarded export must be imported once")
	assert.Contains(t, reports[0], "Lotteries imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	_, err = stored.GetByID(draw.ID)
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

	draw := genuineDesktopLotteryRecord(t, "relative-seed")
	draw.BlockHeight = 3
	document, err := json.Marshal([]lottery.LotteryRecord{draw})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(launcherDir, "draws.json"), []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) { reports = append(reports, title+" | "+message) }

	// The first instance runs elsewhere; only workDir can find the file.
	shell.HandleLaunchArgs([]string{"draws.json"}, launcherDir)

	require.Len(t, reports, 1, "the relative export must be found and imported")
	assert.Contains(t, reports[0], "Lotteries imported")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	_, err = stored.GetByID(draw.ID)
	assert.NoError(t, err, "the draw from the launcher's directory must be stored")

	// The same file via a file:// URL, which Linux launchers also send.
	urlDraw := genuineDesktopLotteryRecord(t, "file-url-seed")
	urlDraw.BlockHeight = 4
	other, err := json.Marshal([]lottery.LotteryRecord{urlDraw})
	require.NoError(t, err)
	urlPath := filepath.Join(launcherDir, "from-url.json")
	require.NoError(t, os.WriteFile(urlPath, []byte(other), 0o644))

	shell.HandleLaunchArgs([]string{"file://" + filepath.ToSlash(urlPath)}, launcherDir)
	require.Len(t, reports, 2)
	assert.Contains(t, reports[1], "Lotteries imported")

	_, err = stored.GetByID(urlDraw.ID)
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

// TestParseIntentIn_DoesNotResolveAgainstTheWrongDirectory pins the path a
// forwarded relative argument names. ParseIntentIn used to try the argument
// as written first, so a file of the same name in the running instance's own
// working directory won: a second launch from ~/Downloads imported the wrong
// export, and the one the user named was never opened. A relative argument
// can only be relative to the launcher's directory.
func TestParseIntentIn_DoesNotResolveAgainstTheWrongDirectory(t *testing.T) {
	launcherDir := t.TempDir()
	want := filepath.Join(launcherDir, "draws.json")
	require.NoError(t, os.WriteFile(want, []byte(`[{"id":"launcher"}]`), 0o644))

	runningDir := t.TempDir()
	decoy := filepath.Join(runningDir, "draws.json")
	require.NoError(t, os.WriteFile(decoy, []byte(`[{"id":"decoy"}]`), 0o644))

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(runningDir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(prev)) })

	in, ok := ParseIntentIn("draws.json", launcherDir)
	require.True(t, ok)
	assert.Equal(t, Intent{File: want, Raw: "draws.json"}, in,
		"the argument is relative to the launcher, not to this process")
}

// TestParseIntentIn_ResolvesRelativeFileURLsAgainstTheLauncher pins the other
// half of the same rule: a launcher may hand over `file:draws.json`, and that
// URL is relative too. Treating its path as absolute (or trying it against
// this process) opens the wrong file or nothing at all.
func TestParseIntentIn_ResolvesRelativeFileURLsAgainstTheLauncher(t *testing.T) {
	launcherDir := t.TempDir()
	want := filepath.Join(launcherDir, "draws.json")
	require.NoError(t, os.WriteFile(want, []byte(`[{"id":"launcher"}]`), 0o644))

	in, ok := ParseIntentIn("file:draws.json", launcherDir)
	require.True(t, ok)
	assert.Equal(t, Intent{File: want, Raw: "file:draws.json"}, in)

	// file:./draws.json is the same spelling with an explicit dot, and it is
	// still relative.
	in, ok = ParseIntentIn("file:./draws.json", launcherDir)
	require.True(t, ok)
	assert.Equal(t, Intent{File: want, Raw: "file:./draws.json"}, in)
}

func TestFileURLPathPlatformPaths(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		goos string
		want string
		ok   bool
	}{
		{
			name: "POSIX absolute file URL",
			arg:  "file:///tmp/draws.json",
			goos: "linux",
			want: "/tmp/draws.json",
			ok:   true,
		},
		{
			name: "POSIX localhost file URL",
			arg:  "file://localhost/tmp/draws.json",
			goos: "linux",
			want: "/tmp/draws.json",
			ok:   true,
		},
		{
			name: "POSIX remote authority is not local",
			arg:  "file://server/share/draws.json",
			goos: "linux",
		},
		{
			name: "Windows drive file URL",
			arg:  "file:///C:/Users/aurora/draws.json",
			goos: "windows",
			want: `C:\Users\aurora\draws.json`,
			ok:   true,
		},
		{
			name: "Windows localhost drive file URL",
			arg:  "file://localhost/C:/Users/aurora/draws.json",
			goos: "windows",
			want: `C:\Users\aurora\draws.json`,
			ok:   true,
		},
		{
			name: "Windows UNC file URL",
			arg:  "file://server/share/draws.json",
			goos: "windows",
			want: `\\server\share\draws.json`,
			ok:   true,
		},
		{
			name: "Windows IPv6 authority is not UNC",
			arg:  "file://[::1]/share/draws.json",
			goos: "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := fileURLPathForOS(tt.arg, tt.goos)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestForwardedPaths covers the flags a second launch can carry. They are
// scanned by hand because flag.Parse stops at the first positional argument,
// so `aurora://voting --data-dir X` must find the directory the same way
// `--data-dir X aurora://voting` does.
func TestForwardedPaths(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		dataDir string
		webRoot string
	}{
		{name: "none", args: []string{"aurora://voting"}},
		{name: "before the link", args: []string{"--data-dir", "/d", "aurora://voting"}, dataDir: "/d"},
		{name: "after the link", args: []string{"aurora://voting", "--data-dir", "/d"}, dataDir: "/d"},
		{name: "equals form", args: []string{"--data-dir=/d"}, dataDir: "/d"},
		{name: "short dashes", args: []string{"-data-dir", "/d"}, dataDir: "/d"},
		{name: "missing value", args: []string{"--data-dir"}},
		{name: "unrelated flags", args: []string{"--version", "draws.json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.dataDir, ForwardedPaths(tt.args))
		})
	}
}

// TestShell_RefusesAForeignDataDir pins what a second launch with a different
// startup path must do: say so, and act on nothing. Importing its export into
// this instance's database would write the draws into a directory the user
// did not name, silently.
func TestShell_RefusesAForeignDataDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "aurora.db")
	exportPath := filepath.Join(dir, "draws.json")
	draw := genuineDesktopLotteryRecord(t, "wrong-db-seed")
	draw.BlockHeight = 5
	document, err := json.Marshal([]lottery.LotteryRecord{draw})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) {
		reports = append(reports, title+" | "+message)
	}
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	other := filepath.Join(t.TempDir(), "other")
	shell.HandleLaunchArgs([]string{"--data-dir", other, "aurora://voting", exportPath}, dir)

	require.Len(t, reports, 1, "a launch naming another database must be reported, not half-applied")
	assert.Contains(t, reports[0], "already running")
	assert.Contains(t, reports[0], other)
	assert.Contains(t, reports[0], "quit Aurora first",
		"the message must name the app: Linux shows the box without its title")
	assert.Empty(t, navigated, "the refused launch must not navigate either")

	// MyGo delivers the launch's own arguments again right after the
	// OnSecondInstance callback. That echo must not sneak the refused page in
	// through the back door.
	shell.OpenURL("aurora://voting")
	assert.Empty(t, navigated, "MyGo's echo of the refused link must not navigate")

	stored, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = stored.Close() })
	_, err = stored.GetByID(draw.ID)
	assert.Error(t, err, "the export must not be imported into the database the launch did not name")

	// The same directory this instance is using is not a conflict: a launcher
	// that spells out --data-dir on every start keeps working.
	shell.HandleLaunchArgs([]string{"--data-dir", dataDir, "aurora://voting"}, dir)
	require.Len(t, reports, 1, "a matching --data-dir must not warn")
	require.Len(t, navigated, 1)

	// The same directory spelled relative to the launcher is the same
	// directory, not a conflict.
	launcher := t.TempDir()
	relData := filepath.Join(launcher, "db")
	require.NoError(t, os.MkdirAll(relData, 0o755))
	relShell := NewShell(About{DBPath: filepath.Join(relData, "aurora.db")})
	relShell.navigate = func(Intent) {}
	var relReports []string
	relShell.report = func(_ mygo.MessageType, title, message, _ string) {
		relReports = append(relReports, title+" | "+message)
	}
	relShell.HandleLaunchArgs([]string{"--data-dir", "db", "aurora://voting"}, launcher)
	assert.Empty(t, relReports, "a relative --data-dir naming the running database must not warn")
}

// TestShell_AcceptsTheRelativeDataDirItStartedWith is the other half of the
// relative-path contract. Start records the database path, and a later launch
// forwards the same relative flag resolved against its launcher directory. If
// the first path stays relative, the two spellings describe one database but
// compare unequal, so every normal relaunch is refused as a conflict.
func TestShell_AcceptsTheRelativeDataDirItStartedWith(t *testing.T) {
	launcher := t.TempDir()
	t.Chdir(launcher)

	dbPath, err := prepareDataDir("db", nil)
	require.NoError(t, err)

	shell := NewShell(About{DBPath: dbPath})
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) {
		reports = append(reports, title+" | "+message)
	}
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	shell.HandleLaunchArgs([]string{"--data-dir", "db", "aurora://voting"}, launcher)

	assert.Empty(t, reports, "the same relative --data-dir must not look like a foreign database")
	require.Len(t, navigated, 1)
	assert.Equal(t, Intent{Page: "voting", Raw: "aurora://voting"}, navigated[0])
}

// TestShell_AcceptsSymlinkAliasesOfRunningPaths pins filesystem identity, not
// string identity. A launcher may reach the same database through a symlinked
// directory; refusing that launch would claim it names another database even
// though both paths resolve to the file the instance already uses.
func TestShell_AcceptsSymlinkAliasesOfRunningPaths(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	alias := filepath.Join(root, "alias")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	dbPath := filepath.Join(dataDir, "aurora.db")
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) {
		reports = append(reports, title+" | "+message)
	}
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	shell.HandleLaunchArgs([]string{
		"--data-dir", filepath.Join(alias, "data"),
		"aurora://voting",
	}, root)

	assert.Empty(t, reports, "a symlink alias names the database already running")
	require.Len(t, navigated, 1)
}

// TestShell_ImportDropped pins what a drop does with each kind of file.
func TestShell_ImportDropped(t *testing.T) {
	shell := NewShell(About{DBPath: filepath.Join(t.TempDir(), "aurora.db")})
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) { reports = append(reports, title+" | "+message) }

	other := filepath.Join(t.TempDir(), "report.csv")
	require.NoError(t, os.WriteFile(other, []byte("a,b"), 0o644))
	shell.ImportDropped([]string{other})
	require.Len(t, reports, 1)
	assert.Contains(t, reports[0], "Unsupported file")
	assert.Contains(t, reports[0], "report.csv")
}

func TestAboutCarriesTheShellIdentity(t *testing.T) {
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

// TestShellMainWindowRecreatesAClosedWindow pins the macOS reopen path: the
// app stays alive with no window, and a Dock click must run the window factory
// rather than return nil and leave the user with an app that cannot be opened.
//
// A real *mygo.Window needs the running app, so the factory here returns nil;
// what is being pinned is that it is called at all, and that the shell does
// not claim success without a window.
func TestShellMainWindowRecreatesAClosedWindow(t *testing.T) {
	shell := NewShell(About{})

	calls := 0
	assert.Nil(t, shell.MainWindow(func() *mygo.Window { calls++; return nil }))
	assert.Equal(t, 1, calls, "with no window the factory must run")

	// Without a factory (tests, a shell that cannot create windows) there is
	// nothing to reopen and no panic.
	assert.NotPanics(t, func() { assert.Nil(t, shell.MainWindow(nil)) })
}

// TestShellEnsureWindowWaitsForReady keeps a startup request from asking MyGo
// for a window before the app can create one: NewWindow panics in that gap.
// Once the app is ready, the same call runs the factory.
func TestShellEnsureWindowWaitsForReady(t *testing.T) {
	originalReady := appReady
	t.Cleanup(func() { appReady = originalReady })

	shell := NewShell(About{})
	calls := 0
	shell.SetWindowFactory(func() *mygo.Window {
		calls++
		return nil
	})

	appReady = func() bool { return false }
	assert.Nil(t, shell.EnsureWindow())
	assert.Zero(t, calls, "a window must not be created before WhenReady")

	appReady = func() bool { return true }
	assert.Nil(t, shell.EnsureWindow())
	assert.Equal(t, 1, calls, "a request after the app is ready must try to reopen the window")
}

// TestShellForwardedLaunchRecreatesWindow pins the macOS lifecycle: Cmd+W
// leaves the process running, and the next launch must reopen the window
// before routing its link. Without this, OnSecondInstance reached Navigate
// with a destroyed window and the page request disappeared.
func TestShellForwardedLaunchRecreatesWindow(t *testing.T) {
	originalReady := appReady
	t.Cleanup(func() { appReady = originalReady })
	appReady = func() bool { return true }

	shell := NewShell(About{})
	factoryCalls := 0
	shell.SetWindowFactory(func() *mygo.Window {
		factoryCalls++
		return nil // a real window needs the running MyGo app
	})
	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }

	shell.HandleLaunchArgs([]string{"aurora://lottery?id=abc"}, "")

	assert.Equal(t, 2, factoryCalls,
		"both the launch and its link must ask for a live window")
	require.Len(t, navigated, 1, "the link must still reach the shell")
	assert.Equal(t, Intent{Page: "lottery", LotteryID: "abc", Raw: "aurora://lottery?id=abc"}, navigated[0])
}

// TestShellOpenedFileRecreatesWindow covers macOS delivering a file directly
// through OnOpenFile, without an OnSecondInstance callback. Importing it while
// the last window is closed would update the database with no visible result.
func TestShellOpenedFileRecreatesWindow(t *testing.T) {
	originalReady, originalAdd := appReady, addRecentDocument
	t.Cleanup(func() { appReady, addRecentDocument = originalReady, originalAdd })
	appReady = func() bool { return true }
	addRecentDocument = func(string) error { return nil }

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	exportPath := filepath.Join(dir, "draws.json")
	draw := genuineDesktopLotteryRecord(t, "reopen-seed")
	draw.BlockHeight = 6
	document, err := json.Marshal([]lottery.LotteryRecord{draw})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(exportPath, []byte(document), 0o644))

	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)
	factoryCalls := 0
	shell.SetWindowFactory(func() *mygo.Window {
		factoryCalls++
		return nil
	})
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) {
		reports = append(reports, title+" | "+message)
	}

	shell.OpenFile(exportPath)

	assert.Equal(t, 1, factoryCalls, "a successfully imported file must recreate the window")
	require.Len(t, reports, 1)
	assert.Contains(t, reports[0], "Lotteries imported")
}

// TestShellFailedImportStillRecreatesWindow keeps a failed file open from
// becoming invisible. The dialog needs a parent and the user needs to see why
// the file was refused, so the window must exist on the failure path too.
func TestShellFailedImportStillRecreatesWindow(t *testing.T) {
	originalReady := appReady
	t.Cleanup(func() { appReady = originalReady })
	appReady = func() bool { return true }

	shell := NewShell(About{DBPath: filepath.Join(t.TempDir(), "missing", "aurora.db")})
	factoryCalls := 0
	shell.SetWindowFactory(func() *mygo.Window {
		factoryCalls++
		return nil
	})
	var reports []string
	shell.report = func(_ mygo.MessageType, title, message, _ string) {
		reports = append(reports, title+" | "+message)
	}

	shell.OpenFile(filepath.Join(t.TempDir(), "draws.json"))

	assert.Equal(t, 1, factoryCalls, "a failed import must still reopen the window")
	require.Len(t, reports, 1)
	assert.Contains(t, reports[0], "Import failed")
}

// TestShellDeliversALinkThatArrivesBeforeTheWindow pins the startup window: a
// second launch can reach OnSecondInstance after the single-instance lock is
// taken but before WhenReady created the window. Navigate had nowhere to go
// and the link was dropped, so `aurora-desktop aurora://lottery?id=x` started
// against an instance that was still booting opened the dashboard instead.
func TestShellDeliversALinkThatArrivesBeforeTheWindow(t *testing.T) {
	shell := NewShell(About{})

	// No window yet, and nothing to observe navigation with: the link must be
	// held, not forgotten.
	shell.Navigate(Intent{Page: "lottery", LotteryID: "abc", Raw: "aurora://lottery?id=abc"})

	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }
	shell.Attach(nil)

	require.Len(t, navigated, 1, "the link held before the window existed must be loaded")
	assert.Equal(t, Intent{Page: "lottery", LotteryID: "abc", Raw: "aurora://lottery?id=abc"}, navigated[0])

	// A second early link replaces the first: the window can show one page,
	// and the newest request is the one the user just made.
	shell.navigate = nil
	shell.Attach(nil)
	shell.Navigate(Intent{Page: "token", Raw: "aurora://token"})
	shell.Navigate(Intent{Page: "nft", Raw: "aurora://nft"})
	var second []Intent
	shell.navigate = func(in Intent) { second = append(second, in) }
	shell.Attach(nil)
	require.Len(t, second, 1)
	assert.Equal(t, "nft", second[0].Page)
}

// TestShellHoldsANavigationForAWindowThatCannotShowIt covers the other half
// of the lifecycle: a window object can still be attached after it stopped
// being displayable. The newest intent must survive until Attach receives the
// replacement window instead of being dropped by Navigate.
func TestShellHoldsANavigationForAWindowThatCannotShowIt(t *testing.T) {
	shell := NewShell(About{})
	shell.Attach(&mygo.Window{}) // no page: not displayable

	shell.Navigate(Intent{Page: "voting", Raw: "aurora://voting"})

	var navigated []Intent
	shell.navigate = func(in Intent) { navigated = append(navigated, in) }
	shell.Attach(nil)

	require.Len(t, navigated, 1, "the replacement window must receive the held page")
	assert.Equal(t, Intent{Page: "voting", Raw: "aurora://voting"}, navigated[0])
}
