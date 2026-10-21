// Package aurora embeds the shipped Web UI so a binary can serve it without a
// checkout.
//
// It sits at the repository root because go:embed only reaches files below the
// embedding package's directory, and web/ is a top-level directory. Copying
// web/ under internal/ would create a second, drift-prone copy of the UI.
//
// Only the desktop module (desktop/) imports this package; it reaches it
// through the module's `replace github.com/pplmx/aurora => ../`. The CLI and
// the REST gateway keep reading web/ from disk (config.WebRoot) so an operator
// can edit the UI without rebuilding.
//
// Embedding here rather than in desktop/ is what keeps MyGo's Go 1.27
// requirement out of the core module: embed is stdlib, so this package adds no
// dependency to go.mod.
package aurora

import (
	"embed"
	"io/fs"
)

// `all:` includes files whose names start with "_" or ".", which fs.Sub would
// otherwise hide.
//
//go:embed all:web
var embedded embed.FS

// WebFS returns the Web UI rooted at its files (index.html, lottery.html,
// css/, js/, vendor/), so a desktop window and the REST gateway serve the same
// relative URLs.
func WebFS() (fs.FS, error) {
	return fs.Sub(embedded, "web")
}
