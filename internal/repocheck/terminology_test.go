package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheCatalogsUseOneWordPerConcept holds the vocabulary the guide and the
// window already share. The two layers drifted once: the CLI said 投票人 and 会话
// for the voter and the voting session, while the window and every document
// said 选民 and 会期. Nobody sees both at once, so nothing fails until an
// operator moves from the command line to the window and reads two names for
// one thing.
func TestTheCatalogsUseOneWordPerConcept(t *testing.T) {
	// The rejected spelling, the accepted one, and where the accepted spelling
	// is the house word.
	// The rejected spelling is written once per script so a rule does not have
	// to guess how a character converts.
	// proseOnly marks a word whose other sense is legitimate in running text:
	// 停用记忆 means "stop remembering", not "a disabled feed". Such a rule
	// judges the catalogs and leaves the documents alone.
	type concept struct {
		rejected, rejectedHant, accepted, why string
		proseOnly                             bool
	}
	concepts := []concept{
		{"投票人", "投票人", "选民", "the guide and the window call the voter 选民", false},
		{"会话", "會話", "会期", "the guide and the window call a voting session 会期", false},
		// A token moves money and an NFT moves an asset; the two were separate
		// house words before the CLI called both of them 转移.
		{"转移", "轉移", "转让", "an NFT transfer is 转让 and a token transfer is 转账", false},
		// The token's precision is 精度 in the guide and the window; the CLI
		// spelled it out as 小数位数.
		{"小数位", "小數位", "精度", "the guide and the window call token decimals 精度", false},
		// Minting is 铸造 everywhere; the CLI had a second word for the same
		// capability in its mintable flag.
		{"增发", "增發", "铸造", "minting a token is 铸造, including its mintable flag", true},
		// A feed that is turned off is 禁用; 停用 appeared once in the window.
		{"停用", "停用", "禁用", "a disabled feed is 禁用", true},
		// The encoding name is written the way the Go package and the guide
		// write it: lowercase.
		{"Base64", "Base64", "base64", "the encoding is written base64, as the guide does", false},
		// Characters that do not belong are 无效; 非法 appeared once.
		{"非法", "非法", "无效", "a rejected character is 无效", true},
	}

	catalogs := [][]string{
		{"internal", "i18n", "messages_zh.go"},
		{"internal", "i18n", "messages_zh_hant.go"},
		{"desktop", "internal", "nativeui", "messages_zh.go"},
		{"desktop", "internal", "nativeui", "messages_zh_hant.go"},
	}
	translations := regexp.MustCompile(`(?m)^\t"[^"]+":\s+"([^"]*)"`)
	checked := 0
	for _, rel := range catalogs {
		body := readRepoFile(t, rel...)
		name := filepath.Join(rel...)
		isTraditional := strings.Contains(name, "hant")
		for _, values := range translations.FindAllStringSubmatch(body, -1) {
			value := values[1]
			checked++
			for _, c := range concepts {
				rejected := c.rejected
				if isTraditional {
					rejected = c.rejectedHant
				}
				// 会话 is also the right word for a process session: the window
				// keeps private keys "for this session", which is not a voting
				// session and must not be renamed. Both scripts are checked
				// because the exclusion is written in either one.
				if c.rejected == "会话" {
					processSession := []string{"会话密钥", "會話密鑰", "本次会话", "本次會話", "本会话", "本會話"}
					excluded := false
					for _, phrase := range processSession {
						if strings.Contains(value, phrase) {
							excluded = true
							break
						}
					}
					if excluded {
						continue
					}
				}
				assert.Falsef(t, strings.Contains(value, rejected),
					"%s uses %q where the house word is %q: %s", name, rejected, c.accepted, c.why)
			}
		}
	}
	require.Greater(t, checked, 500, "the scan must reach the catalogs")

	// The documents are where a retired word comes back: a paragraph written
	// from memory, or a phrase copied from an old release note. They are checked
	// for the same rejected spellings.
	for _, doc := range []string{"README.md", "docs/cli.md", "docs/desktop.md", "docs/api.md"} {
		body := readRepoFile(t, strings.Split(doc, "/")...)
		for _, c := range concepts {
			if c.proseOnly {
				continue
			}
			if c.rejected == "会话" {
				// A process session is still 会话, and the documents speak of
				// 会话密钥 and the current shell session. Only the voting sense
				// was renamed, and a document that says 会话 for a voting session
				// would still carry a word the window does not use — so the
				// check runs on every 会话 that is not one of these phrases.
				if remaining := stripProcessSessions(body); !strings.Contains(remaining, "会话") {
					continue
				}
			}
			assert.Falsef(t, strings.Contains(body, c.rejected),
				"%s uses %q where the house word is %q: %s", doc, c.rejected, c.accepted, c.why)
		}
	}
}

// stripProcessSessions removes the phrases where 会话 is the right word: a
// private key held "for this session" is not a voting session.
func stripProcessSessions(body string) string {
	for _, phrase := range []string{"会话密钥", "會話密鑰", "本次会话", "本次會話", "本会话", "本會話", "当前会话", "當前會話"} {
		body = strings.ReplaceAll(body, phrase, "")
	}
	return body
}
