package repocheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// markdownLink matches an inline link target: the text in brackets, the target
// in parentheses. Images share the syntax but their target is a file, which is
// checked the same way.
var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// markdownHeading matches an ATX heading and captures its text.
var markdownHeading = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)

// TestHeadingAnchorFollowsGitHubSlugs pins the slug rule itself, because the
// link check is only as good as it. Two of these were wrong before: CJK
// punctuation was kept as if it were a letter (so "模块、工具链" and
// "模块工具链" produced different anchors), and trailing punctuation left a
// dangling hyphen that GitHub does not emit.
func TestHeadingAnchorFollowsGitHubSlugs(t *testing.T) {
	for _, tc := range []struct{ heading, want string }{
		{"快速开始", "快速开始"},
		{"模块、工具链与 CI", "模块工具链与-ci"},
		{"视觉、主题与字号", "视觉主题与字号"},
		{"文件操作与 Settings", "文件操作与-settings"},
		{"macOS 发布签名", "macos-发布签名"},
		{"`token burn` 与 --confirm", "token-burn-与---confirm"},
		{"为什么是嵌套模块？", "为什么是嵌套模块"},
		{"  spaced  out  ", "spaced-out"},
	} {
		require.Equalf(t, tc.want, headingAnchor(tc.heading),
			"headingAnchor(%q)", tc.heading)
	}
}

// TestDocumentationAnchorsResolve keeps a cross-document link usable. A link
// into another file's heading silently rots whenever that heading is renamed —
// which is exactly what happened when the CLI reference's closing section was
// split and the README gained a "#确认门" link: nothing but a reader would have
// noticed a later rename.
//
// Only relative links between repository files are checked; external URLs are
// out of scope and are already left to the reader's browser.
func TestDocumentationAnchorsResolve(t *testing.T) {
	repo := repoRoot(t)
	checked := 0

	err := filepath.WalkDir(repo, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			// Vendored modules, build output and VCS data are not project docs.
			switch name {
			case ".git", ".gocache", ".gomodcache", ".gopath", ".gotmp",
				"node_modules", "dist", ".screens", "logs", "data":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}

		rel, _ := filepath.Rel(repo, path)
		if strings.HasPrefix(rel, "docs/superpowers/") || strings.HasPrefix(rel, ".planning/") {
			// Historical records: kept as written, never treated as current
			// instructions (see AGENTS.md).
			return nil
		}

		source, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, match := range markdownLink.FindAllStringSubmatch(string(source), -1) {
			target := match[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			filePart, anchor, hasAnchor := strings.Cut(target, "#")
			if !hasAnchor {
				// A link without an anchor still has to name a real file.
				if filePart != "" {
					assertFileExists(t, filepath.Dir(path), filePart, rel, target)
					checked++
				}
				continue
			}
			checked++
			destination := path
			if filePart != "" {
				destination = filepath.Join(filepath.Dir(path), filePart)
				if !assertFileExists(t, filepath.Dir(path), filePart, rel, target) {
					continue
				}
			}
			if !strings.HasSuffix(destination, ".md") {
				// Anchors into non-markdown files cannot be verified here.
				continue
			}
			body, err := os.ReadFile(destination)
			require.NoError(t, err)
			assert.Truef(t, hasHeadingAnchor(string(body), anchor),
				"%s links to %q, but no heading in %s produces that anchor",
				rel, target, strings.TrimPrefix(destination, repo+string(filepath.Separator)))
		}
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, checked, 5, "the link audit must see the cross-document links")
}

// assertFileExists reports whether a relative link resolves to a file in the
// repository, and returns that answer so a caller can skip the anchor check.
func assertFileExists(t *testing.T, dir, target, source, fullTarget string) bool {
	t.Helper()
	path := filepath.Join(dir, target)
	info, err := os.Stat(path)
	// A directory link is legitimate: docs/superpowers/lottery is a folder of
	// records, and a reader following it expects the listing.
	ok := err == nil && info != nil
	assert.Truef(t, ok, "%s links to %q, which does not name a file or directory", source, fullTarget)
	if ok && info.IsDir() {
		return false
	}
	return ok
}

// hasHeadingAnchor reports whether any heading in body produces anchor. The
// rule is GitHub's: lowercase the heading text, drop everything that is not a
// letter, digit, space, hyphen or underscore, then join the words with hyphens.
// Chinese headings keep their characters, which is why the check runs on runes
// rather than bytes.
func hasHeadingAnchor(body, anchor string) bool {
	for _, match := range markdownHeading.FindAllStringSubmatch(body, -1) {
		if headingAnchor(match[1]) == anchor {
			return true
		}
	}
	return false
}

// headingAnchor is the slug GitHub derives from a heading's text. Inline code,
// emphasis and link syntax are stripped first, because "## `token burn`" and
// "## token burn" must produce the same anchor.
func headingAnchor(heading string) string {
	heading = strings.NewReplacer("`", "", "*", "", "_", "", "[", "", "]", "").Replace(heading)
	var b strings.Builder
	pendingSpace := false
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r == ' ' || r == '\t':
			pendingSpace = true
		case r == '-' || r == '_' || (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') || (r > 127 && unicode.IsLetter(r)):
			if pendingSpace && b.Len() > 0 {
				b.WriteRune('-')
			}
			pendingSpace = false
			b.WriteRune(r)
		default:
			// Punctuation is dropped without ending the word. That includes
			// CJK punctuation: a heading "模块、工具链与 CI" slugs to
			// "模块工具链与-ci" on GitHub, because the ideographic comma is
			// punctuation there rather than a letter.
		}
	}
	return b.String()
}
