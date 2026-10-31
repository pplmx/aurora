package repocheck

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDesktopFormatStringsKeepTheirArguments covers the desktop catalog, where
// the English literal is the key and the translation receives the same argument
// list. A translation that drops a verb prints "%!d(MISSING)" to the user, and
// one that adds a verb the call does not pass prints "%!d(string=...)". Neither
// fails a test, because fmt.Sprintf does not report a bad format.
//
// Positional verbs are allowed and are the reason the check cannot be a plain
// verb count: 从 %[2]s 销毁 %[1]s？ reorders the two arguments on purpose.
func TestDesktopFormatStringsKeepTheirArguments(t *testing.T) {
	entry := regexp.MustCompile(`(?m)^\t"((?:[^"\\]|\\.)*)":\s+"((?:[^"\\]|\\.)*)",$`)
	verb := regexp.MustCompile(`%(\[(\d+)\])?[-+ #0-9.]*[a-zA-Z]`)

	for _, name := range []string{"messages_zh.go", "messages_zh_hant.go"} {
		body := readRepoFile(t, "desktop", "internal", "nativeui", name)
		checked := 0
		for _, pair := range entry.FindAllStringSubmatch(body, -1) {
			key, value := pair[1], pair[2]
			keyHigh, keyUsed := formatArgs(key, verb)
			valueHigh, valueUsed := formatArgs(value, verb)
			if keyHigh == 0 && keyUsed == 0 && valueHigh == 0 && valueUsed == 0 {
				continue
			}
			checked++
			// Every argument the key supplies must be used, and none beyond it.
			assert.Equalf(t, keyUsed, valueUsed,
				"%s: %q takes %d arguments but %q uses %d", filepath.Join("desktop", name), key, keyUsed, value, valueUsed)
			assert.LessOrEqualf(t, valueHigh, keyUsed,
				"%s: %q refers to argument %d of %d", filepath.Join("desktop", name), value, valueHigh, keyUsed)
		}
		require.Greater(t, checked, 30, "%s must contain format strings", name)
	}
}

// formatArgs reports how many distinct arguments the string refers to and the
// highest index among them. Distinct, not highest: 从 %[2]s 销毁？references two
// arguments' worth of index but supplies only one, and a highest-only count
// would accept it.
func formatArgs(s string, verb *regexp.Regexp) (highest, count int) {
	seen := map[int]bool{}
	for _, m := range verb.FindAllStringSubmatch(s, -1) {
		if m[2] != "" {
			n := 0
			for _, c := range m[2] {
				n = n*10 + int(c-'0')
			}
			if n > highest {
				highest = n
			}
			seen[n] = true
			continue
		}
		count++
	}
	return highest, count + len(seen)
}
