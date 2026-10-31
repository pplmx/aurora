package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// asciiAfterChinese matches a Chinese character glued to an ASCII comma, colon
// or semicolon. Chinese prose and labels use the full-width forms, and thirteen
// label strings had drifted to the ASCII colon while forty-seven used the
// full-width one — the migrate table printed both styles on adjacent lines.
var asciiAfterChinese = regexp.MustCompile(`[\p{Han}][,:;]`)

// TestChineseCatalogsUseFullWidthPunctuation keeps a new label from picking the
// ASCII colon: a value whose Chinese character is followed by ASCII punctuation
// fails here, while a Latin label such as "ID: %s" or a URL is untouched.
// Traditional tables are generated from the simplified ones, so they are not
// scanned.
func TestChineseCatalogsUseFullWidthPunctuation(t *testing.T) {
	for _, rel := range [][]string{
		{"internal", "i18n", "messages_zh.go"},
		{"desktop", "internal", "nativeui", "messages_zh.go"},
	} {
		content := readRepoFile(t, rel...)
		for i, line := range strings.Split(content, "\n") {
			if asciiAfterChinese.MatchString(line) {
				t.Errorf("%s:%d uses ASCII punctuation after a Chinese character: %s",
					filepath.ToSlash(filepath.Join(rel...)), i+1, strings.TrimSpace(line))
			}
		}
	}

	// The matcher must recognise the shape it exists for, and must leave a Latin
	// label alone; otherwise a green run would prove nothing.
	if !asciiAfterChinese.MatchString(`"key": "当前版本: %d"`) {
		t.Fatal("the matcher no longer recognises an ASCII colon after Chinese")
	}
	if asciiAfterChinese.MatchString(`"key": "ID: %s"`) {
		t.Fatal("the matcher must not flag a Latin label")
	}
}
