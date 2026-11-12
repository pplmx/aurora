package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// recordClipboard replaces the native clipboard for one test and records what
// the window copied. The replacement still writes through ui.Services, so an
// assertion on tester.Clipboard() sees the same text; what it adds is the
// confidential mark, which the headless tester does not model.
func recordClipboard(t *testing.T) *[]clipboardWrite {
	t.Helper()
	previous := writeClipboard
	var writes []clipboardWrite
	writeClipboard = func(c *ui.Context, text string, concealed bool) {
		writes = append(writes, clipboardWrite{text: text, concealed: concealed})
		c.Services().WriteClipboard(text)
	}
	t.Cleanup(func() { writeClipboard = previous })
	return &writes
}

type clipboardWrite struct {
	text      string
	concealed bool
}
