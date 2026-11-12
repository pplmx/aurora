package nativeui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// writeClipboard puts text on the system clipboard for a Copy control.
//
// concealed marks content only the operator should keep, such as a private
// key: the platform then adds its confidential mark (nspasteboard's
// ConcealedType on macOS, ExcludeClipboardContentFromMonitorProcessing on
// Windows, x-kde-passwordManagerHint on Linux), which lets clipboard managers
// leave it out of their histories. It is a variable because view tests run in
// MyGo's headless tester rather than a running app, where the native clipboard
// is unavailable; a test can observe the write and its mark instead.
var writeClipboard = func(c *ui.Context, text string, concealed bool) {
	if concealed {
		err := mygo.Clipboard.Write(transfer.TextData(text), mygo.ClipboardOptions{Concealed: true})
		if err == nil {
			return
		}
		// A platform without the clipboard still gets the copy; it loses only
		// the mark that keeps the value out of a manager's history.
	}
	c.Services().WriteClipboard(text)
}
