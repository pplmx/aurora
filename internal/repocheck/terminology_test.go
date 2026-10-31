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
	type concept struct{ rejected, rejectedHant, accepted, why string }
	concepts := []concept{
		{"投票人", "投票人", "选民", "the guide and the window call the voter 选民"},
		{"会话", "會話", "会期", "the guide and the window call a voting session 会期"},
		// A token moves money and an NFT moves an asset; the two were separate
		// house words before the CLI called both of them 转移.
		{"转移", "轉移", "转让", "an NFT transfer is 转让 and a token transfer is 转账"},
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
}
