package deskapp

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
)

// DeepLinkScheme is the scheme the operating system hands to the app: an
// aurora://lottery?id=<id> link from a terminal, a chat message or a README
// opens Aurora on that draw. It is declared in desktop/mygo.config.ts
// (urlSchemes) so an installed bundle registers it, and registered at run
// time with App.RegisterURLScheme on platforms without an installer.
//
// It deliberately matches the window's own scheme (Scheme): one name for
// "Aurora", and MyGo keeps the two apart - Scheme is the webview transport
// the app serves, DeepLinkScheme is an operating-system registration whose
// URLs arrive as launch arguments.
const DeepLinkScheme = Scheme

// Intent is what a launch argument asks the shell to show: a page of the Web
// UI, optionally focused on one draw, or a lottery export to import.
//
// MyGo delivers the same arguments through App.OnOpenURL / App.OnOpenFile,
// so an Intent is expected twice when the app starts: once from our own
// command line and once from MyGo's launch delivery. Both paths build it with
// this function, so they cannot disagree about which pages exist.
type Intent struct {
	// Raw is the launch argument this intent was parsed from, so the caller
	// can tell MyGo which argument it has already honored.
	Raw string
	// Page is a page of the Web UI without its ".html": "lottery", "voting",
	// "token", "oracle", "nft", "blockchain". Empty is the dashboard.
	Page string
	// LotteryID is the draw to open on the lottery page, when the link names
	// one. It is a deep link's payload, not a lookup: an unknown id surfaces
	// as the API's normal "lottery not found" in the Verify panel.
	LotteryID string
	// File is a lottery export to import: a file dropped on the window, or a
	// path passed after `aurora-desktop` on the command line.
	File string
}

// pages are the Web UI documents a deep link may name. Keeping the list here
// means a link cannot navigate the window at a path the embedded UI does not
// serve (which would render the gateway's 404 page inside the app).
var pages = map[string]bool{
	"":           true,
	"index":      true,
	"lottery":    true,
	"voting":     true,
	"token":      true,
	"oracle":     true,
	"nft":        true,
	"blockchain": true,
}

// ParseIntent reads a launch argument: an aurora:// link or the path of a
// lottery export. It reports false for anything else, which is how the flag
// parser keeps rejecting unexpected positionals.
func ParseIntent(arg string) (Intent, bool) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return Intent{}, false
	}
	if IsDeepLink(arg) {
		in, ok := parseDeepLink(arg)
		in.Raw = arg
		return in, ok
	}
	if isLotteryDocument(arg) {
		return Intent{File: arg, Raw: arg}, true
	}
	return Intent{}, false
}

// ParseIntentIn is ParseIntent for an argument that came from another process.
//
// A launch argument is relative to the directory the launching process ran
// in, not to ours: `cd ~/Downloads && aurora-desktop draws.json` names
// ~/Downloads/draws.json. MyGo's own file delivery resolves such a path
// against the working directory it forwards with the argument, so a shell
// that parses the arguments itself must do the same or the file never
// resolves. Linux launchers may also hand over file:// URLs, which MyGo
// accepts; so does this.
func ParseIntentIn(arg, workDir string) (Intent, bool) {
	arg = strings.TrimSpace(arg)
	if in, ok := ParseIntent(arg); ok || arg == "" || IsDeepLink(arg) || workDir == "" {
		return in, ok
	}
	if path, ok := fileURLPath(arg); ok {
		if in, ok := ParseIntent(path); ok {
			return in, true
		}
	}
	if filepath.IsAbs(arg) {
		return Intent{}, false
	}
	path := filepath.Join(workDir, arg)
	if !isLotteryDocument(path) {
		return Intent{}, false
	}
	// File is the resolved, importable path; Raw stays as the launching
	// process wrote it, which is what Intent.Raw means.
	return Intent{File: path, Raw: arg}, true
}

// fileURLPath converts a file:// URL to the path it names, for the launchers
// that pass files that way instead of as arguments.
func fileURLPath(arg string) (string, bool) {
	u, err := url.Parse(arg)
	if err != nil || !strings.EqualFold(u.Scheme, "file") || u.Path == "" {
		return "", false
	}
	path := filepath.FromSlash(u.Path)
	if runtime.GOOS == "windows" {
		// file:///C:/x arrives as /C:/x.
		path = strings.TrimPrefix(path, string(filepath.Separator))
	}
	return path, true
}

func parseDeepLink(raw string) (Intent, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return Intent{}, false
	}
	// aurora://lottery?id=x names the page in the host (the Windows and Linux
	// launchers preserve it that way); aurora://localhost/lottery.html?id=x
	// is the same link written as the window's own URL, which people copy
	// out of the address bar. Accept both spellings.
	name := u.Host
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".html")
	switch {
	case name == "" || strings.EqualFold(name, "localhost"):
		name = path
	case path != "":
		name += "/" + path
	}
	name = strings.ToLower(strings.TrimSuffix(name, ".html"))
	if !pages[name] {
		return Intent{}, false
	}
	if name == "index" {
		name = ""
	}
	return Intent{Page: name, LotteryID: u.Query().Get("id")}, true
}

// isLotteryDocument reports whether path is a lottery export this shell knows
// how to import. The extension list mirrors fileAssociations in
// desktop/mygo.config.ts; MyGo only hands us files it matched against that
// list, so this check is what keeps a typo'd positional an error.
func isLotteryDocument(path string) bool {
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// URL is the page to load in the window for this intent.
func (in Intent) URL() string {
	if in.Page == "" {
		return URL()
	}
	raw := Scheme + "://localhost/" + in.Page + ".html"
	if in.LotteryID != "" {
		raw += "?" + url.Values{"id": {in.LotteryID}}.Encode()
	}
	return raw
}

// IsDeepLink reports whether arg is an aurora:// link, so a caller can tell
// links from files before deciding which path of the shell should handle it.
func IsDeepLink(arg string) bool {
	i := strings.Index(arg, "://")
	return i > 0 && strings.EqualFold(arg[:i], DeepLinkScheme)
}

// ImportLotteryFile stores the draws of the export at path through the same
// app-layer importer the CLI and the File menu use, and reports what
// happened. Opening a .json from the file manager and dropping one on the
// window both land here.
func ImportLotteryFile(meta About, path string) (lotteryapp.ImportResult, error) {
	document, err := os.ReadFile(path)
	if err != nil {
		return lotteryapp.ImportResult{}, err
	}
	repo, closeRepo, err := openLotteryRepo(meta)
	if err != nil {
		return lotteryapp.ImportResult{}, err
	}
	defer closeRepo()
	return lotteryapp.ImportRecords(repo, document)
}
