package repocheck

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequiredFieldMessagesReadAsSentences covers the wording of the one error
// a client hits while wiring its first request. The handlers said four
// different things for the same situation:
//
//	"token_id required"
//	"token_id is required"
//	"token_id and owner required"
//	"token_id, owner and spender required"
//
// while the oracle endpoints already said "source parameter is required". A
// reader comparing two endpoints could not tell whether the missing word meant
// anything, and the shortest form reads as a log key rather than a message sent
// to a person.
//
// The check is deliberately about the shape, not the exact nouns: an endpoint
// gains a field and its list grows, which is fine; dropping the verb is what is
// being prevented.
func TestRequiredFieldMessagesReadAsSentences(t *testing.T) {
	// A message that ends in "required" without a verb before it.
	bare := regexp.MustCompile(`"[^"]*\b(?:is|are)\s+required"|"[^"]*\brequired"`)
	missingVerb := regexp.MustCompile(`"([^"]*\brequired)"`)

	checked := 0
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "internal", "api", "handler"), func(path, body string) {
		for _, m := range regexp.MustCompile(`writeBadRequest\(w,\s*("[^"]+")\)`).FindAllStringSubmatch(body, -1) {
			message := m[1]
			if !regexp.MustCompile(`required`).MatchString(message) {
				continue
			}
			checked++
			assert.Truef(t, bare.MatchString(message),
				"%s: %s is not shaped like a sentence", path, message)
			assert.Falsef(t, missingVerb.MatchString(message) &&
				!regexp.MustCompile(`(is|are)\s+required`).MatchString(message),
				"%s: %s ends in \"required\" with no verb; write \"… is required\"", path, message)
		}
	}))
	require.GreaterOrEqual(t, checked, 4, "the scan must reach the validation messages")
}
